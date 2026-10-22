// Package aaguid maps passkey AAGUIDs to provider names and icons, using a
// locally cached copy of the community-maintained passkey-authenticator-aaguids
// list. The list is downloaded at runtime and never shipped with Boardly,
// because upstream declares no license.
package aaguid

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Provider is the passkey provider an AAGUID belongs to. Icons are data: URIs.
type Provider struct {
	Name      string  `json:"name"`
	IconLight *string `json:"iconLight"`
	IconDark  *string `json:"iconDark"`
}

// Defaults for downloading the list.
const (
	DefaultMaxAge  = 24 * time.Hour
	DefaultTimeout = 30 * time.Second
	DefaultMaxSize = 20 << 20
)

// Directory holds the provider list. The zero value and a nil *Directory
// know no providers.
type Directory struct {
	// URL is where the list is downloaded from; empty disables downloads.
	URL string
	// Path is the cache file. Its ETag is stored next to it in Path + ".etag".
	Path    string
	Client  *http.Client
	MaxAge  time.Duration
	MaxSize int64
	Now     func() time.Time

	providers atomic.Pointer[map[string]Provider]
}

// New returns a directory that downloads from url and caches in path.
func New(url, path string) *Directory {
	return &Directory{
		URL:     url,
		Path:    path,
		Client:  &http.Client{Timeout: DefaultTimeout},
		MaxAge:  DefaultMaxAge,
		MaxSize: DefaultMaxSize,
		Now:     time.Now,
	}
}

// Lookup returns the provider for a 16-byte AAGUID, or nil when it is unknown,
// all zero, or no list is loaded.
func (d *Directory) Lookup(aaguid []byte) *Provider {
	if d == nil || len(aaguid) != 16 {
		return nil
	}
	id, err := uuid.FromBytes(aaguid)
	if err != nil || id == uuid.Nil {
		return nil
	}
	m := d.providers.Load()
	if m == nil {
		return nil
	}
	if p, ok := (*m)[id.String()]; ok {
		return &p
	}
	return nil
}

// Len returns the number of known providers.
func (d *Directory) Len() int {
	if m := d.providers.Load(); m != nil {
		return len(*m)
	}
	return 0
}

// Load reads the cached list, if there is one. A missing file is not an error.
func (d *Directory) Load() error {
	data, err := os.ReadFile(d.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	m, err := parse(data)
	if err != nil {
		return fmt.Errorf("cached provider list %s: %w", d.Path, err)
	}
	d.providers.Store(&m)
	return nil
}

// Refresh downloads the list with a conditional GET. A valid new list replaces
// the cache file and the in-memory list; on any error both are kept.
func (d *Directory) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.URL, nil)
	if err != nil {
		return err
	}
	if etag, err := os.ReadFile(d.etagPath()); err == nil && d.Len() > 0 {
		req.Header.Set("If-None-Match", strings.TrimSpace(string(etag)))
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading provider list: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		// Mark the cache as fresh, so the next check waits MaxAge again.
		now := d.Now()
		return os.Chtimes(d.Path, now, now)
	case http.StatusOK:
	default:
		return fmt.Errorf("downloading provider list: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, d.MaxSize+1))
	if err != nil {
		return fmt.Errorf("downloading provider list: %w", err)
	}
	if int64(len(data)) > d.MaxSize {
		return fmt.Errorf("provider list is larger than %d bytes", d.MaxSize)
	}
	m, err := parse(data)
	if err != nil {
		return fmt.Errorf("downloaded provider list: %w", err)
	}
	if err := writeAtomic(d.Path, data); err != nil {
		return err
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		if err := writeAtomic(d.etagPath(), []byte(etag)); err != nil {
			return err
		}
	} else {
		_ = os.Remove(d.etagPath())
	}
	d.providers.Store(&m)
	return nil
}

// Run loads the cache, refreshes it when it is older than MaxAge and then
// every MaxAge until ctx is done. Errors are logged; the last good list stays.
func (d *Directory) Run(ctx context.Context) {
	if err := d.Load(); err != nil {
		slog.WarnContext(ctx, "loading passkey provider list failed", "error", err)
	}
	if d.URL == "" {
		return
	}
	wait := time.Duration(0)
	if info, err := os.Stat(d.Path); err == nil && d.Len() > 0 {
		wait = max(0, d.MaxAge-d.Now().Sub(info.ModTime()))
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := d.Refresh(ctx); err != nil && ctx.Err() == nil {
				slog.WarnContext(ctx, "refreshing passkey provider list failed", "error", err)
			} else if err == nil {
				slog.InfoContext(ctx, "passkey provider list refreshed", "providers", d.Len())
			}
			timer.Reset(d.MaxAge)
		}
	}
}

func (d *Directory) etagPath() string { return d.Path + ".etag" }

// parse validates the upstream shape {"<aaguid>": {"name", "icon_light", "icon_dark"}}
// and returns the providers keyed by lowercase hyphenated AAGUID.
func parse(data []byte) (map[string]Provider, error) {
	var raw map[string]struct {
		Name      string  `json:"name"`
		IconLight *string `json:"icon_light"`
		IconDark  *string `json:"icon_dark"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("not a JSON object of providers: %w", err)
	}
	if len(raw) == 0 {
		return nil, errors.New("no providers")
	}
	m := make(map[string]Provider, len(raw))
	for key, v := range raw {
		id, err := normalize(key)
		if err != nil {
			return nil, fmt.Errorf("invalid AAGUID %q", key)
		}
		if strings.TrimSpace(v.Name) == "" {
			return nil, fmt.Errorf("provider %s has no name", key)
		}
		m[id] = Provider{Name: v.Name, IconLight: dataURI(v.IconLight), IconDark: dataURI(v.IconDark)}
	}
	return m, nil
}

// normalize returns the AAGUID in lowercase hyphenated form; it accepts the
// hyphenated and the plain 32-digit hex form.
func normalize(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) == 32 {
		if _, err := hex.DecodeString(s); err != nil {
			return "", err
		}
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// dataURI keeps only icons that are data: image URIs, so the web app never
// loads anything from elsewhere.
func dataURI(s *string) *string {
	if s == nil || !strings.HasPrefix(*s, "data:image/") {
		return nil
	}
	return s
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
