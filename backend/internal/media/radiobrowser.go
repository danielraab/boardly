package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"boardly/internal/images"
)

// Limits of the station directory client.
const (
	DirectoryTimeout = 10 * time.Second
	// SearchLimit is how many stations a search returns.
	SearchLimit = 30
	// maxDirectoryBody caps a directory response.
	maxDirectoryBody = 1 << 20
	// serverListTTL is how long the list of mirrors is used.
	serverListTTL = time.Hour
)

// ErrDirectoryUnreachable is returned when radio-browser.info can't be
// reached or answers with an error.
var ErrDirectoryUnreachable = errors.New("station directory unreachable")

// directoryUserAgent identifies Boardly to radio-browser.info, which asks
// clients to send a descriptive agent.
const directoryUserAgent = "Boardly (+https://github.com/danielraab/boardly)"

// Station is a station as a directory search returns it.
type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// URL is the stream URL, resolved by the directory where it could.
	URL string `json:"url"`
	// LogoURL is the directory's logo URL, or nil.
	LogoURL *string `json:"logoUrl"`
	Country *string `json:"country"`
	// Tags are at most three.
	Tags []string `json:"tags"`
	// Bitrate is in kbit/s, nil when unknown.
	Bitrate *int `json:"bitrate"`
	// Playable is whether URL is https; only https streams play on an https page.
	Playable bool `json:"playable"`
}

// Directory searches radio-browser.info, a free community directory of
// internet radio stations, on behalf of the browser, so that the browser
// sends the user's IP address to no one else. Nothing about the user is
// forwarded.
type Directory struct {
	Client *http.Client
	// ServersURL lists the directory's mirrors.
	ServersURL string
	// Fallback is the mirror used when the list can't be loaded.
	Fallback string
	// MirrorURL returns the base URL of a mirror host; https by default.
	MirrorURL func(host string) string
	Now       func() time.Time

	mu        sync.Mutex
	mirrors   []string
	fetchedAt time.Time
}

// NewDirectory returns a client whose connections are guarded unless
// allowPrivate is set.
func NewDirectory(allowPrivate bool) *Directory {
	return &Directory{
		Client:     &http.Client{Transport: images.GuardedTransport(allowPrivate)},
		ServersURL: "https://all.api.radio-browser.info/json/servers",
		Fallback:   "de1.api.radio-browser.info",
		MirrorURL:  func(host string) string { return "https://" + host },
		Now:        time.Now,
	}
}

type rawStation struct {
	StationUUID string `json:"stationuuid"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	URLResolved string `json:"url_resolved"`
	Favicon     string `json:"favicon"`
	Country     string `json:"country"`
	Tags        string `json:"tags"`
	Bitrate     int    `json:"bitrate"`
}

func optional(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return &s
}

// toStation maps a directory entry to a search result.
func toStation(raw rawStation) Station {
	streamURL := strings.TrimSpace(raw.URLResolved)
	if streamURL == "" {
		streamURL = strings.TrimSpace(raw.URL)
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = streamURL
	}
	tags := []string{}
	for t := range strings.SplitSeq(raw.Tags, ",") {
		if t = strings.TrimSpace(t); t != "" && len(tags) < 3 {
			tags = append(tags, t)
		}
	}
	var bitrate *int
	if raw.Bitrate > 0 {
		b := raw.Bitrate
		bitrate = &b
	}
	return Station{
		ID:       raw.StationUUID,
		Name:     name,
		URL:      streamURL,
		LogoURL:  optional(raw.Favicon),
		Country:  optional(raw.Country),
		Tags:     tags,
		Bitrate:  bitrate,
		Playable: strings.HasPrefix(strings.ToLower(streamURL), "https://"),
	}
}

// get requests rawURL and decodes its JSON body into v.
func (d *Directory) get(ctx context.Context, rawURL string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", directoryUserAgent)
	resp, err := d.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status %d", rawURL, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxDirectoryBody)).Decode(v)
}

// mirror picks a mirror at random from the cached list, loading the list
// when it is missing or old, or returns the fallback.
func (d *Directory) mirror(ctx context.Context) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.mirrors == nil || d.Now().Sub(d.fetchedAt) > serverListTTL {
		var list []struct {
			Name string `json:"name"`
		}
		if err := d.get(ctx, d.ServersURL, &list); err == nil {
			var names []string
			for _, s := range list {
				if s.Name = strings.TrimSpace(s.Name); s.Name != "" {
					names = append(names, s.Name)
				}
			}
			if len(names) > 0 {
				d.mirrors, d.fetchedAt = names, d.Now()
			}
		}
	}
	if len(d.mirrors) == 0 {
		return d.Fallback
	}
	return d.mirrors[rand.IntN(len(d.mirrors))]
}

// Search returns at most SearchLimit stations whose name matches query, most
// popular first. It fails with ErrDirectoryUnreachable.
func (d *Directory) Search(ctx context.Context, query string) ([]Station, error) {
	ctx, cancel := context.WithTimeout(ctx, DirectoryTimeout)
	defer cancel()
	params := url.Values{
		"name":       {query},
		"limit":      {fmt.Sprint(SearchLimit)},
		"order":      {"clickcount"},
		"reverse":    {"true"},
		"hidebroken": {"true"},
	}
	var raw []rawStation
	if err := d.get(ctx, d.MirrorURL(d.mirror(ctx))+"/json/stations/search?"+params.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDirectoryUnreachable, err)
	}
	if len(raw) > SearchLimit {
		raw = raw[:SearchLimit]
	}
	out := make([]Station, len(raw))
	for i, r := range raw {
		out[i] = toStation(r)
	}
	return out, nil
}
