package offline

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chromedp/cdproto/network"

	"boardly/internal/images"
)

func TestBrowserFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/page", http.StatusFound)
		case "/big":
			_, _ = w.Write(make([]byte, MaxBrowserResponse+1))
		default:
			if r.Header.Get("Cookie") != "" {
				t.Errorf("cookie sent: %q", r.Header.Get("Cookie"))
			}
			http.SetCookie(w, &http.Cookie{Name: "track", Value: "1"})
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<p>hi</p>")
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	get := func(u string) *network.Request {
		return &network.Request{URL: u, Method: http.MethodGet, Headers: network.Headers{"Accept": "text/html", "Cookie": "a=b"}}
	}

	open := NewBrowser("ws://unused", images.GuardedTransport(true))
	res, reason := open.Fetch(ctx, get(srv.URL+"/page"), 1)
	if res == nil || reason != "" || res.ResponseCode != 200 || string(res.Body) != "<p>hi</p>" {
		t.Fatalf("public page: %+v, %q", res, reason)
	}
	for _, h := range res.ResponseHeaders {
		if strings.EqualFold(h.Name, "Set-Cookie") {
			t.Error("Set-Cookie passed to the browser")
		}
	}
	// Redirects go back to the browser, which requests (and so checks) the target.
	if res, _ := open.Fetch(ctx, get(srv.URL+"/redirect"), 1); res == nil || res.ResponseCode != http.StatusFound {
		t.Fatalf("redirect: %+v", res)
	}

	for name, tt := range map[string]struct {
		b    *Browser
		req  *network.Request
		n    int32
		want network.ErrorReason
	}{
		"private address": {NewBrowser("ws://unused", images.GuardedTransport(false)), get(srv.URL + "/page"), 1, network.ErrorReasonBlockedByClient},
		"file URL":        {open, get("file:///etc/passwd"), 1, network.ErrorReasonBlockedByClient},
		"too many":        {open, get(srv.URL + "/page"), MaxBrowserRequests + 1, network.ErrorReasonBlockedByClient},
		"post":            {open, &network.Request{URL: srv.URL + "/page", Method: http.MethodPost}, 1, network.ErrorReasonBlockedByClient},
		"too large":       {open, get(srv.URL + "/big"), 1, network.ErrorReasonFailed},
	} {
		if res, reason := tt.b.Fetch(ctx, tt.req, tt.n); res != nil || reason != tt.want {
			t.Errorf("%s: %+v, %q; want %q", name, res, reason, tt.want)
		}
	}
}

// TestBrowserScreenshot runs against a real headless browser when
// BOARDLY_TEST_BROWSER_URL names its DevTools endpoint, for example
//
//	docker run --rm -p 9222:9222 chromedp/headless-shell:latest
//	BOARDLY_TEST_BROWSER_URL=ws://127.0.0.1:9222 go test ./internal/offline -run Browser
//
// The browser reaches the test page only through Boardly, so it works even
// when the browser runs in a container without access to the host.
func TestBrowserScreenshot(t *testing.T) {
	browserURL := os.Getenv("BOARDLY_TEST_BROWSER_URL")
	if browserURL == "" {
		t.Skip("BOARDLY_TEST_BROWSER_URL not set")
	}
	var private atomic.Int32
	// The "private" server stands for an internal service: the guard below
	// blocks only its port.
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { private.Add(1) }))
	defer internal.Close()
	_, internalPort, _ := net.SplitHostPort(internal.Listener.Addr().String())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body style="background:#c00"><h1>Hello</h1><img src="`+internal.URL+`/x.png"></body></html>`)
	}))
	defer srv.Close()

	transport := images.GuardedTransport(true)
	dial := transport.DialContext
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if _, port, _ := net.SplitHostPort(addr); port == internalPort {
			return nil, images.ErrBlocked
		}
		return dial(ctx, network, addr)
	}
	b := NewBrowser(browserURL, transport)
	shot, err := b.Screenshot(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(shot))
	if err != nil || format != "jpeg" || cfg.Width != ScreenshotWidth || cfg.Height != ScreenshotHeight {
		t.Fatalf("screenshot %s %dx%d, %v", format, cfg.Width, cfg.Height, err)
	}
	// The page's red background shows that the browser got the page from
	// Boardly, and not an error page.
	img, _, err := image.Decode(bytes.NewReader(shot))
	if err != nil {
		t.Fatal(err)
	}
	r, g, bl, _ := img.At(ScreenshotWidth-10, ScreenshotHeight-10).RGBA()
	if r>>8 < 0xb0 || g>>8 > 0x30 || bl>>8 > 0x30 {
		t.Fatalf("corner colour = %d %d %d, want the page's red", r>>8, g>>8, bl>>8)
	}
	if n := private.Load(); n != 0 {
		t.Fatalf("requests to the blocked address = %d", n)
	}

	// A page on a blocked address gets no screenshot.
	if _, err := b.Screenshot(context.Background(), internal.URL); err == nil {
		t.Fatal("screenshot of a blocked page succeeded")
	}
	if n := private.Load(); n != 0 {
		t.Fatalf("requests to the blocked address = %d", n)
	}
}

func TestResolveEndpoint(t *testing.T) {
	ctx := context.Background()
	for in, want := range map[string]string{
		"ws://127.0.0.1:9222":      "ws://127.0.0.1:9222",
		"http://localhost:9222/":   "http://localhost:9222/",
		"ws://[::1]:9222/devtools": "ws://[::1]:9222/devtools",
	} {
		if got, err := resolveEndpoint(ctx, in); err != nil || got != want {
			t.Errorf("resolveEndpoint(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// A name is replaced by its address, keeping port and path.
	got, err := resolveEndpoint(ctx, "ws://ip6-localhost.invalid:9222/x")
	if err == nil {
		t.Errorf("unknown host resolved to %q", got)
	}
	got, err = resolveEndpoint(ctx, "http://localhost.:9222/json")
	if err != nil || strings.Contains(got, "localhost") || !strings.HasSuffix(got, ":9222/json") {
		t.Errorf("resolveEndpoint(localhost.) = %q, %v", got, err)
	}
}
