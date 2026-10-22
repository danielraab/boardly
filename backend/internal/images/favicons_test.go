package images

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsBlocked(t *testing.T) {
	for _, ip := range []string{
		"127.0.0.1", "10.0.0.5", "172.16.3.4", "192.168.1.10", "169.254.169.254",
		"100.64.0.1", "0.0.0.0", "224.0.0.1", "255.255.255.255", "198.18.0.1",
		"::1", "::", "fc00::1", "fd12::1", "fe80::1", "ff02::1",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "64:ff9b::a00:1",
	} {
		if !IsBlocked(netip.MustParseAddr(ip)) {
			t.Errorf("%s is not blocked", ip)
		}
	}
	for _, ip := range []string{"93.184.215.14", "1.1.1.1", "2606:4700:4700::1111", "::ffff:8.8.8.8"} {
		if IsBlocked(netip.MustParseAddr(ip)) {
			t.Errorf("%s is blocked", ip)
		}
	}
}

func TestOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"https://GitHub.com/settings?x=1": "https://github.com",
		"https://example.org:443/a":       "https://example.org",
		"http://example.org:80":           "http://example.org",
		"http://192.168.1.10:8096/web":    "http://192.168.1.10:8096",
		"http://[::1]:8080/":              "http://[::1]:8080",
	} {
		if got, err := Origin(in); err != nil || got != want {
			t.Errorf("Origin(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"example.org", "javascript:alert(1)", "ftp://example.org", "https://"} {
		if _, err := Origin(in); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("Origin(%q): err = %v", in, err)
		}
	}
}

func TestParseIcons(t *testing.T) {
	base, _ := url.Parse("https://example.org/docs/page")
	page := `<html><head>
		<link rel="apple-touch-icon" href="/touch.png" sizes="180x180">
		<link rel="shortcut icon" href="/old.ico">
		<link rel="icon" href="small.png" sizes="16x16">
		<link rel="icon" href="/big.png" sizes="32x32 48x48">
		<link rel="icon" href="/logo.svg">
		<link rel="icon" type="image/svg+xml" href="/logo">
		<link rel="icon" href="data:image/png;base64,AAAA">
		<link rel="stylesheet" href="/style.css">
		</head><body><link rel="icon" href="/late.png"></body></html>`
	got := ParseIcons(strings.NewReader(page), base)
	want := []string{
		"https://example.org/big.png",
		"https://example.org/docs/small.png",
		"https://example.org/old.ico",
		"https://example.org/touch.png",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("icons = %v, want %v", got, want)
	}

	withBase := `<head><base href="https://cdn.example.net/assets/"><link rel="icon" href="i.png"></head>`
	if got := ParseIcons(strings.NewReader(withBase), base); len(got) != 1 || got[0] != "https://cdn.example.net/assets/i.png" {
		t.Fatalf("with base = %v", got)
	}
}

// site is a test web site that counts its requests.
type site struct {
	*httptest.Server
	requests atomic.Int32
	routes   map[string]func(w http.ResponseWriter, r *http.Request)
}

func newSite(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) *site {
	t.Helper()
	s := &site{routes: routes}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		if h, ok := s.routes[r.URL.Path]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func htmlPage(body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func file(data []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }
}

// fakeClock is a settable clock.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func newFavicons(t *testing.T, allowPrivate bool) (*Favicons, *fakeClock) {
	t.Helper()
	s, _ := newStore(t)
	c := &fakeClock{now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	s.Now = c.Now
	f := NewFavicons(s, allowPrivate)
	f.Now = c.Now
	return f, c
}

func (f *Favicons) mustRead(t *testing.T, id string) []byte {
	t.Helper()
	_, file, err := f.Images.Open(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := ReadLimited(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestFetchDeclaredIconAndFallback(t *testing.T) {
	ctx := context.Background()
	f, _ := newFavicons(t, true)

	declared := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/":                htmlPage(`<link rel="icon" href="/static/logo.png">`),
		"/static/logo.png": file(pngData),
		"/favicon.ico":     file(icoData),
	})
	id, err := f.Get(ctx, declared.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.mustRead(t, id), pngData) {
		t.Fatal("declared icon not used")
	}

	fallback := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/":            htmlPage(`<title>No icon</title>`),
		"/favicon.ico": file(icoData),
	})
	id, err = f.Get(ctx, fallback.URL+"/some/page")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.mustRead(t, id), icoData) {
		t.Fatal("fallback not used")
	}
}

func TestFetchRefusesBadIcons(t *testing.T) {
	ctx := context.Background()
	f, _ := newFavicons(t, true)
	for name, routes := range map[string]map[string]func(http.ResponseWriter, *http.Request){
		"svg only": {
			"/":         htmlPage(`<link rel="icon" href="/logo.svg">`),
			"/logo.svg": file(svgData),
		},
		"svg as ico": {"/favicon.ico": file(svgData)},
		"oversize":   {"/favicon.ico": file(append(bytes.Clone(pngData), make([]byte, MaxSize)...))},
		"nothing":    {},
	} {
		s := newSite(t, routes)
		if _, err := f.Get(ctx, s.URL); !errors.Is(err, ErrNoFavicon) {
			t.Errorf("%s: err = %v, want ErrNoFavicon", name, err)
		}
	}

	// A redirect loop on the page still allows the /favicon.ico fallback.
	loop := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/":            func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/", http.StatusFound) },
		"/favicon.ico": file(icoData),
	})
	if _, err := f.Get(ctx, loop.URL); err != nil {
		t.Fatalf("redirect loop: %v", err)
	}
	if n := loop.requests.Load(); n != MaxRedirects+2 {
		t.Fatalf("redirect loop made %d requests, want %d", n, MaxRedirects+2)
	}
}

func TestFaviconCache(t *testing.T) {
	ctx := context.Background()
	f, clock := newFavicons(t, true)
	icon := pngData
	s := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/favicon.ico": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(icon) },
	})

	id, err := f.Get(ctx, s.URL+"/a")
	if err != nil {
		t.Fatal(err)
	}
	fetches := s.requests.Load()

	// Same origin, another page: served from the cache.
	clock.now = clock.now.Add(29 * 24 * time.Hour)
	if again, err := f.Get(ctx, s.URL+"/b"); err != nil || again != id || s.requests.Load() != fetches {
		t.Fatalf("cached: %q, %v, %d requests", again, err, s.requests.Load())
	}
	if got, err := f.Lookup(ctx, []string{s.URL, "https://unknown.example"}); err != nil || len(got) != 1 || got[s.URL] != id {
		t.Fatalf("Lookup = %v, %v", got, err)
	}

	// After 30 days it is fetched again under the same ID.
	icon = gifData
	clock.now = clock.now.Add(2 * 24 * time.Hour)
	if refreshed, err := f.Get(ctx, s.URL); err != nil || refreshed != id {
		t.Fatalf("refresh: %q, %v", refreshed, err)
	}
	if !bytes.Equal(f.mustRead(t, id), gifData) {
		t.Fatal("refresh didn't replace the bytes")
	}

	// A failed refresh keeps the old image.
	delete(s.routes, "/favicon.ico")
	clock.now = clock.now.Add(31 * 24 * time.Hour)
	if kept, err := f.Get(ctx, s.URL); err != nil || kept != id {
		t.Fatalf("failed refresh: %q, %v", kept, err)
	}
}

func TestFaviconBackOff(t *testing.T) {
	ctx := context.Background()
	f, clock := newFavicons(t, true)
	s := newSite(t, nil)

	if _, err := f.Get(ctx, s.URL); !errors.Is(err, ErrNoFavicon) {
		t.Fatalf("first: %v", err)
	}
	n := s.requests.Load()
	clock.now = clock.now.Add(time.Hour)
	if _, err := f.Get(ctx, s.URL); !errors.Is(err, ErrNoFavicon) || s.requests.Load() != n {
		t.Fatalf("within a day: %v, %d requests (was %d)", err, s.requests.Load(), n)
	}
	s.routes = map[string]func(http.ResponseWriter, *http.Request){"/favicon.ico": file(icoData)}
	clock.now = clock.now.Add(24 * time.Hour)
	if _, err := f.Get(ctx, s.URL); err != nil {
		t.Fatalf("after a day: %v", err)
	}
}

func TestFaviconBlocksPrivateAddresses(t *testing.T) {
	ctx := context.Background()
	f, _ := newFavicons(t, false)
	s := newSite(t, map[string]func(http.ResponseWriter, *http.Request){"/favicon.ico": file(icoData)})

	for range 2 {
		if _, err := f.Get(ctx, s.URL); !errors.Is(err, ErrBlocked) {
			t.Fatalf("err = %v, want ErrBlocked", err)
		}
	}
	if n := s.requests.Load(); n != 0 {
		t.Fatalf("server got %d requests", n)
	}

	// A public page that redirects to a private address isn't followed. The
	// redirect target is checked like the first address.
	redirector := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/": func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, s.URL+"/", http.StatusFound) },
	})
	f2, _ := newFavicons(t, false)
	f2.Client.Transport.(*http.Transport).DialContext = (&fakePublic{
		target: redirector.Listener.Addr().String(),
		guard:  guardedDialer(false),
	}).DialContext
	if _, err := f2.Get(ctx, "http://public.example/"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("redirect: err = %v, want ErrBlocked", err)
	}
	if redirector.requests.Load() != 1 || s.requests.Load() != 0 {
		t.Fatalf("requests: redirector %d, private %d", redirector.requests.Load(), s.requests.Load())
	}
}

// fakePublic connects public.example to a local server without the guard,
// as if it were a public site, and guards every other address.
type fakePublic struct {
	target string
	guard  *net.Dialer
}

func (d *fakePublic) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if addr == "public.example:80" {
		return (&net.Dialer{}).DialContext(ctx, network, d.target)
	}
	return d.guard.DialContext(ctx, network, addr)
}

func TestLookupImages(t *testing.T) {
	ctx := context.Background()
	f, _ := newFavicons(t, true)
	pngSite := newSite(t, map[string]func(http.ResponseWriter, *http.Request){"/favicon.ico": file(pngData)})
	icoSite := newSite(t, map[string]func(http.ResponseWriter, *http.Request){"/favicon.ico": file(icoData)})
	pngID, err := f.Get(ctx, pngSite.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	icoID, err := f.Get(ctx, icoSite.URL+"/")
	if err != nil {
		t.Fatal(err)
	}

	got, err := f.LookupImages(ctx, []string{pngSite.URL, icoSite.URL, "https://unknown.example"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]FaviconImage{
		pngSite.URL: {ID: pngID, ContentType: "image/png"},
		icoSite.URL: {ID: icoID, ContentType: "image/x-icon"},
	}
	if len(got) != len(want) || got[pngSite.URL] != want[pngSite.URL] || got[icoSite.URL] != want[icoSite.URL] {
		t.Fatalf("LookupImages = %v, want %v", got, want)
	}
}
