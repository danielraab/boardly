package feeds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"boardly/internal/images"
)

// Limits of fetching a feed.
const (
	FetchTimeout = 20 * time.Second
	MaxFeedSize  = 32 << 20
	MaxRedirects = 5
)

// FetchError is a failed fetch. RetryAfter is what the host asked for, or 0.
type FetchError struct {
	RetryAfter time.Duration
	Err        error
}

func (e *FetchError) Error() string { return e.Err.Error() }
func (e *FetchError) Unwrap() error { return ErrUnreachable }

// Conditional holds the validators of the previous response.
type Conditional struct {
	ETag         string
	LastModified string
}

// Response is a fetched document.
type Response struct {
	// URL is the normalized URL the document came from, after redirects.
	URL string
	// Moved is true when every redirect on the way there was permanent, and
	// there was at least one.
	Moved       bool
	Body        []byte
	ContentType string
	Validators  Conditional
	NotModified bool
}

// Fetcher loads feeds over guarded connections. It never relays anything to
// a browser.
type Fetcher struct {
	Client  *http.Client
	Timeout time.Duration
	MaxSize int64
}

// NewFetcher returns a fetcher whose connections are guarded unless
// allowPrivate is set.
func NewFetcher(allowPrivate bool) *Fetcher {
	return &Fetcher{Client: &http.Client{Transport: images.GuardedTransport(allowPrivate)}, Timeout: FetchTimeout, MaxSize: MaxFeedSize}
}

// Get fetches rawURL once.
func (f *Fetcher) Get(ctx context.Context, rawURL string, cond Conditional) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()
	return f.get(ctx, rawURL, cond)
}

// getUpgraded fetches u, trying https first when u is http, within one time
// limit.
func (f *Fetcher) getUpgraded(ctx context.Context, u *url.URL) (Response, error) {
	if u.Scheme == "http" {
		secure := *u
		secure.Scheme = "https"
		if resp, err := f.get(ctx, secure.String(), Conditional{}); err == nil {
			return resp, nil
		}
	}
	return f.get(ctx, u.String(), Conditional{})
}

func (f *Fetcher) get(ctx context.Context, rawURL string, cond Conditional) (Response, error) {
	moved := true
	redirects := 0
	client := *f.Client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > MaxRedirects {
			return fmt.Errorf("more than %d redirects", MaxRedirects)
		}
		redirects++
		if s := req.Response; s != nil && s.StatusCode != http.StatusMovedPermanently && s.StatusCode != http.StatusPermanentRedirect {
			moved = false
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Response{}, ErrInvalidURL
	}
	req.Header.Set("User-Agent", "Boardly feed reader")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/feed+json, application/xml;q=0.9, text/xml;q=0.9, text/html;q=0.8, */*;q=0.5")
	if cond.ETag != "" {
		req.Header.Set("If-None-Match", cond.ETag)
	}
	if cond.LastModified != "" {
		req.Header.Set("If-Modified-Since", cond.LastModified)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, &FetchError{Err: err}
	}
	defer resp.Body.Close()
	final, err := normalizeURL(resp.Request.URL.String())
	if err != nil {
		return Response{}, &FetchError{Err: err}
	}
	out := Response{URL: final, Moved: moved && redirects > 0, ContentType: resp.Header.Get("Content-Type")}
	if resp.StatusCode == http.StatusNotModified {
		out.NotModified = true
		return out, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Response{}, &FetchError{
			RetryAfter: retryAfter(resp.Header.Get("Retry-After"), time.Now()),
			Err:        fmt.Errorf("status %d", resp.StatusCode),
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.MaxSize+1))
	if err != nil {
		return Response{}, &FetchError{Err: err}
	}
	if int64(len(body)) > f.MaxSize {
		return Response{}, &FetchError{Err: errors.New("feed too large")}
	}
	out.Body = body
	out.Validators = Conditional{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	return out, nil
}

// retryAfter reads a Retry-After header, in seconds or as a date.
func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if n, err := strconv.Atoi(value); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil && t.After(now) {
		return t.Sub(now)
	}
	return 0
}

// normalizeURL returns the form of an absolute http or https URL that
// identifies a feed: lower-case scheme and host, no default port, no
// fragment, and at least "/" as the path. The query is kept, so private feeds
// with a token stay distinct.
func normalizeURL(rawURL string) (string, error) {
	u, err := parseURL(rawURL)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func parseURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, ErrInvalidURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil, ErrInvalidURL
	}
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port := u.Port(); port != "" && !(u.Scheme == "http" && port == "80") && !(u.Scheme == "https" && port == "443") {
		host += ":" + port
	}
	u.Host = host
	u.Fragment = ""
	u.RawFragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}
