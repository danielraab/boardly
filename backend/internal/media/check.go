package media

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"boardly/internal/images"
)

// Limits of the URL check.
const (
	CheckTimeout    = 10 * time.Second
	MaxPlaylistSize = 64 << 10
	MaxRedirects    = 5
	// maxEntries bounds how many entries of a playlist are tried.
	maxEntries = 5
)

// Errors of the URL check.
var (
	ErrInvalidURL  = errors.New("not an absolute http or https URL")
	ErrNotHTTPS    = errors.New("stream not available over https")
	ErrUnreachable = errors.New("stream unreachable or not audio")
)

// errHTTPRedirect stops a redirect to an http URL.
var errHTTPRedirect = errors.New("redirect to http")

// Checker checks that a URL is an audio stream the browser can play over
// https. It reads only response headers and small playlists; it never relays
// audio.
type Checker struct {
	Client  *http.Client
	Timeout time.Duration
}

// NewChecker returns a checker whose connections are guarded unless
// allowPrivate is set.
func NewChecker(allowPrivate bool) *Checker {
	return &Checker{Client: &http.Client{Transport: images.GuardedTransport(allowPrivate)}, Timeout: CheckTimeout}
}

// Check returns the https URL of the stream behind rawURL: an http URL is
// tried as https, and a PLS or M3U playlist is resolved to its first playable
// entry. HLS playlists are streams themselves.
func (c *Checker) Check(ctx context.Context, rawURL string) (string, error) {
	u, err := parseURL(strings.TrimSpace(rawURL), nil)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	return c.check(ctx, u, true)
}

// parseURL parses an absolute http or https URL, or one relative to base.
func parseURL(rawURL string, base *url.URL) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidURL
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrInvalidURL
	}
	u.Fragment = ""
	return u, nil
}

// check checks u, upgraded to https, and resolves a playlist if allowed.
func (c *Checker) check(ctx context.Context, u *url.URL, playlists bool) (string, error) {
	wasHTTP := u.Scheme == "http"
	target := *u
	target.Scheme = "https"
	final, err := c.fetch(ctx, &target, playlists)
	if err != nil && wasHTTP && errors.Is(err, ErrUnreachable) {
		return "", fmt.Errorf("%w: %v", ErrNotHTTPS, err)
	}
	return final, err
}

func (c *Checker) fetch(ctx context.Context, u *url.URL, playlists bool) (string, error) {
	client := *c.Client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > MaxRedirects {
			return fmt.Errorf("more than %d redirects", MaxRedirects)
		}
		if req.URL.Scheme != "https" {
			return errHTTPRedirect
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", ErrInvalidURL
	}
	req.Header.Set("User-Agent", "Boardly stream check")
	req.Header.Set("Icy-MetaData", "0")
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, errHTTPRedirect) {
			return "", fmt.Errorf("%w: %v", ErrNotHTTPS, err)
		}
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("%w: status %d", ErrUnreachable, resp.StatusCode)
	}
	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	kind := playlistKind(contentType, path.Ext(strings.ToLower(resp.Request.URL.Path)))
	switch {
	case kind == "" && isAudio(contentType):
		return u.String(), nil
	case kind == "":
		return "", fmt.Errorf("%w: content type %q", ErrUnreachable, contentType)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxPlaylistSize))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if kind == "m3u" && bytes.Contains(body, []byte("#EXT-X-")) {
		return u.String(), nil // HLS: the browser plays the playlist itself
	}
	if !playlists {
		return "", fmt.Errorf("%w: nested playlist", ErrUnreachable)
	}
	entries := playlistEntries(kind, body)
	lastErr := fmt.Errorf("%w: empty playlist", ErrUnreachable)
	for i, entry := range entries {
		if i == maxEntries {
			break
		}
		eu, err := parseURL(entry, resp.Request.URL)
		if err != nil {
			continue
		}
		final, err := c.check(ctx, eu, false)
		if err == nil {
			return final, nil
		}
		lastErr = err
	}
	// The playlist itself was reachable over https; its entries weren't.
	if errors.Is(lastErr, ErrNotHTTPS) {
		return "", lastErr
	}
	return "", fmt.Errorf("%w: %v", ErrUnreachable, lastErr)
}

// playlistKind returns "pls", "m3u" (also HLS) or "" by content type or, for
// a generic type, by extension.
func playlistKind(contentType, ext string) string {
	switch contentType {
	case "audio/x-scpls", "application/pls+xml", "audio/scpls":
		return "pls"
	case "audio/x-mpegurl", "audio/mpegurl", "application/vnd.apple.mpegurl", "application/x-mpegurl":
		return "m3u"
	}
	if isAudio(contentType) {
		return ""
	}
	switch ext {
	case ".pls":
		return "pls"
	case ".m3u", ".m3u8":
		return "m3u"
	}
	return ""
}

func isAudio(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/") || contentType == "application/ogg"
}

// playlistEntries returns the URLs a PLS or M3U playlist lists, in order.
func playlistEntries(kind string, body []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		switch kind {
		case "pls":
			key, value, ok := strings.Cut(line, "=")
			if ok && strings.HasPrefix(strings.ToLower(key), "file") {
				out = append(out, strings.TrimSpace(value))
			}
		case "m3u":
			if line != "" && !strings.HasPrefix(line, "#") {
				out = append(out, line)
			}
		}
	}
	return out
}
