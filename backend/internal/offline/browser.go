package offline

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"

	"boardly/internal/images"
)

// Limits of the requests a page makes while its screenshot is taken.
const (
	// MaxBrowserRequests is the most requests one screenshot may make.
	MaxBrowserRequests = 300
	// MaxBrowserResponse is the largest response passed to the browser.
	MaxBrowserResponse = 10 << 20
	// browserRequestTimeout is the longest one request may take.
	browserRequestTimeout = 10 * time.Second
	// settleTime is how long the page may render after its load event.
	settleTime = time.Second
	// screenshotQuality is the JPEG quality the browser encodes with.
	screenshotQuality = 85
)

// Browser takes screenshots with a headless browser that runs elsewhere and
// is reached through the Chrome DevTools Protocol. The browser never loads
// anything itself: every request of the page is paused, made by Boardly with
// Client, which follows the private address protection, and answered to the
// browser. One screenshot is taken at a time.
type Browser struct {
	// URL is the DevTools endpoint: ws(s)://host:port/... or http(s)://host:port.
	URL string
	// Client makes the page's requests. It must not follow redirects, so the
	// browser sees and requests each step, and each is checked.
	Client *http.Client

	slot chan struct{}
}

// NewBrowser returns a browser client for the DevTools endpoint browserURL
// whose page requests go through transport.
func NewBrowser(browserURL string, transport http.RoundTripper) *Browser {
	return &Browser{
		URL: browserURL,
		Client: &http.Client{
			Transport:     transport,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		slot: make(chan struct{}, 1),
	}
}

// Screenshot opens pageURL in a new, empty browser context with a
// ScreenshotWidth × ScreenshotHeight viewport and returns a JPEG of the
// viewport.
func (b *Browser) Screenshot(ctx context.Context, pageURL string) ([]byte, error) {
	select {
	case b.slot <- struct{}{}:
		defer func() { <-b.slot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	endpoint, err := resolveEndpoint(ctx, b.URL)
	if err != nil {
		return nil, fmt.Errorf("resolving the browser's address: %w", err)
	}
	allocCtx, cancelAlloc := remote.NewAllocator(ctx, endpoint)
	defer cancelAlloc()
	tabCtx, cancelTab := chromedp.NewContext(allocCtx, chromedp.WithNewBrowserContext(),
		chromedp.WithLogf(func(string, ...any) {}), chromedp.WithErrorf(func(string, ...any) {}))
	defer cancelTab()

	paused := chromedp.Events(tabCtx, fetch.RequestPaused)
	var count atomic.Int32
	go func() {
		for ev, err := range paused {
			if err != nil {
				return
			}
			go b.answer(tabCtx, ev, &count)
		}
	}()
	if _, err := chromedp.Call(tabCtx, fetch.Enable, fetch.EnableParams{
		Patterns: []*fetch.RequestPattern{{URLPattern: "*"}},
	}); err != nil {
		return nil, fmt.Errorf("intercepting requests: %w", err)
	}
	if err := chromedp.Do(tabCtx,
		chromedp.EmulateViewport(ScreenshotWidth, ScreenshotHeight),
		chromedp.Navigate(pageURL),
		chromedp.Sleep(settleTime),
	); err != nil {
		return nil, err
	}
	quality := int64(screenshotQuality)
	res, err := chromedp.Call(tabCtx, page.CaptureScreenshot, page.CaptureScreenshotParams{
		Format: page.CaptureScreenshotFormatJpeg, Quality: &quality,
	})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// resolveEndpoint replaces the host name of the DevTools endpoint with its
// address: Chrome only answers DevTools requests whose Host header is an IP
// address or localhost, so a name such as a Compose service's would be
// refused. It is resolved for every screenshot, because a container's
// address can change.
func resolveEndpoint(ctx context.Context, endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "localhost" || net.ParseIP(host) != nil {
		return endpoint, nil
	}
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("no address for %s", host)
	}
	if port := u.Port(); port != "" {
		u.Host = net.JoinHostPort(addrs[0], port)
	} else {
		u.Host = addrs[0]
		if strings.Contains(addrs[0], ":") {
			u.Host = "[" + addrs[0] + "]"
		}
	}
	return u.String(), nil
}

// answer makes the paused request and fulfils it, or fails it.
func (b *Browser) answer(ctx context.Context, ev fetch.EventRequestPaused, count *atomic.Int32) {
	fulfil, reason := b.Fetch(ctx, ev.Request, count.Add(1))
	if fulfil != nil {
		fulfil.RequestID = ev.RequestID
		if _, err := chromedp.Call(ctx, fetch.FulfillRequest, *fulfil); err == nil {
			return
		}
		reason = network.ErrorReasonFailed
	}
	_, _ = chromedp.Call(ctx, fetch.FailRequest, fetch.FailRequestParams{RequestID: ev.RequestID, ErrorReason: reason})
}

// Fetch makes the browser's request req, the n-th of the screenshot, and
// returns the response for the browser, or the reason it fails: requests
// beyond MaxBrowserRequests, of a scheme other than http and https, other
// methods than GET and HEAD, to blocked addresses, larger than
// MaxBrowserResponse, and requests that fail.
func (b *Browser) Fetch(ctx context.Context, req *network.Request, n int32) (*fetch.FulfillRequestParams, network.ErrorReason) {
	if n > MaxBrowserRequests {
		return nil, network.ErrorReasonBlockedByClient
	}
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, network.ErrorReasonBlockedByClient
	}
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return nil, network.ErrorReasonBlockedByClient
	}
	ctx, cancel := context.WithTimeout(ctx, browserRequestTimeout)
	defer cancel()
	out, err := http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
	if err != nil {
		return nil, network.ErrorReasonFailed
	}
	for k, v := range req.Headers {
		switch strings.ToLower(k) {
		case "host", "cookie", "accept-encoding", "connection", "content-length", "proxy-authorization", "authorization":
			continue
		}
		if s, ok := v.(string); ok {
			out.Header.Set(k, s)
		}
	}
	resp, err := b.Client.Do(out)
	if err != nil {
		if errors.Is(err, images.ErrBlocked) {
			return nil, network.ErrorReasonBlockedByClient
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, network.ErrorReasonTimedOut
		}
		return nil, network.ErrorReasonFailed
	}
	defer resp.Body.Close()
	body, tooLarge, err := readAll(resp.Body, MaxBrowserResponse)
	if err != nil || tooLarge {
		return nil, network.ErrorReasonFailed
	}
	var headers []*fetch.HeaderEntry
	for k, vs := range resp.Header {
		switch strings.ToLower(k) {
		case "set-cookie", "content-encoding", "content-length", "transfer-encoding", "connection", "alt-svc":
			continue
		}
		for _, v := range vs {
			headers = append(headers, &fetch.HeaderEntry{Name: k, Value: v})
		}
	}
	return &fetch.FulfillRequestParams{ResponseCode: int64(resp.StatusCode), ResponseHeaders: headers, Body: body}, ""
}
