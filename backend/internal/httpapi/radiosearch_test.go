package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeRadioBrowser points the server's directory at a test server whose
// search returns a station with the logo logoURL (or none), and counts
// searches.
func (s *server) fakeRadioBrowser(t *testing.T, status int, logoURL string) *atomic.Int32 {
	t.Helper()
	var searches atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json/servers":
			_, _ = w.Write([]byte(`[{"name":"at1.api.radio-browser.info"}]`))
		case "/json/stations/search":
			searches.Add(1)
			if status != 0 {
				w.WriteHeader(status)
				return
			}
			_, _ = w.Write([]byte(`[{"stationuuid":"a","name":"FM4","url_resolved":"https://fm4.example/live","favicon":` +
				mustJSON(logoURL) + `,"country":"Austria","tags":"indie","bitrate":128}]`))
		}
	}))
	t.Cleanup(srv.Close)
	s.directory.Client = srv.Client()
	s.directory.ServersURL = srv.URL + "/json/servers"
	s.directory.MirrorURL = func(string) string { return srv.URL }
	return &searches
}

type searchResultJSON struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	LogoURL  *string `json:"logoUrl"`
	Logo     *string `json:"logo"`
	Playable bool    `json:"playable"`
}

func TestSearchStations(t *testing.T) {
	s := newServer(t)
	s.fakeRadioBrowser(t, 0, "https://fm4.example/logo.png")
	ann := s.signIn(t, "ann@example.org")
	rec := s.req("GET", "/api/media/search?q=+fm4+", "", withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("search: %d %s", rec.Code, rec.Body)
	}
	list := decodeJSONBody[[]searchResultJSON](t, rec)
	if len(list) != 1 {
		t.Fatalf("results = %+v", list)
	}
	fm4 := list[0]
	if fm4.Name != "FM4" || fm4.URL != "https://fm4.example/live" || !fm4.Playable ||
		fm4.LogoURL == nil || *fm4.LogoURL != "https://fm4.example/logo.png" || fm4.Logo == nil ||
		!strings.HasPrefix(*fm4.Logo, "/api/media/search/logo?") {
		t.Fatalf("fm4 = %+v", fm4)
	}

	s.fakeRadioBrowser(t, 0, "")
	rec = s.req("GET", "/api/media/search?q=fm4", "", withCookie(ann))
	if list := decodeJSONBody[[]searchResultJSON](t, rec); list[0].Logo != nil || list[0].LogoURL != nil {
		t.Fatalf("without logo = %+v", list[0])
	}
}

func TestSearchStationsErrors(t *testing.T) {
	s := newServer(t)
	searches := s.fakeRadioBrowser(t, 0, "")
	if rec := s.req("GET", "/api/media/search?q=fm4", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
	ann := s.signIn(t, "ann@example.org")
	for _, q := range []string{"", "f", "  f  ", strings.Repeat("x", 101)} {
		rec := s.req("GET", "/api/media/search?"+url.Values{"q": {q}}.Encode(), "", withCookie(ann))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_query" {
			t.Fatalf("q=%q: %d %s", q, rec.Code, rec.Body)
		}
	}
	if searches.Load() != 0 {
		t.Fatalf("%d searches sent", searches.Load())
	}

	s.fakeRadioBrowser(t, http.StatusServiceUnavailable, "")
	rec := s.req("GET", "/api/media/search?q=fm4", "", withCookie(ann))
	if rec.Code != http.StatusBadGateway || errorCode(t, rec) != "directory_unreachable" {
		t.Fatalf("directory down: %d %s", rec.Code, rec.Body)
	}
}

func TestSearchStationsRateLimited(t *testing.T) {
	s := newServer(t)
	s.fakeRadioBrowser(t, 0, "")
	ann := s.signIn(t, "ann@example.org")
	var last *httptest.ResponseRecorder
	for range searchBurst + 1 {
		last = s.req("GET", "/api/media/search?q=fm4", "", withCookie(ann))
	}
	if last.Code != http.StatusTooManyRequests || errorCode(t, last) != "rate_limited" || last.Header().Get("Retry-After") == "" {
		t.Fatalf("after burst: %d %s", last.Code, last.Body)
	}
	bob := s.signIn(t, "bob@example.org")
	if rec := s.req("GET", "/api/media/search?q=fm4", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Fatalf("other user: %d", rec.Code)
	}
}

func TestSearchLogo(t *testing.T) {
	s := newServer(t)
	s.allowPrivateFetches()
	var fetches atomic.Int32
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		switch r.URL.Path {
		case "/logo.png":
			_, _ = w.Write(testPNG)
		case "/page":
			_, _ = w.Write([]byte("<html></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(site.Close)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	logoLink := func(logoURL string) string {
		s.fakeRadioBrowser(t, 0, logoURL)
		rec := s.req("GET", "/api/media/search?q=fm4", "", withCookie(ann))
		return *decodeJSONBody[[]searchResultJSON](t, rec)[0].Logo
	}

	link := logoLink(site.URL + "/logo.png")
	rec := s.req("GET", link, "", withCookie(ann))
	if rec.Code != http.StatusOK || rec.Body.String() != string(testPNG) {
		t.Fatalf("logo: %d", rec.Code)
	}
	h := rec.Header()
	if h.Get("Content-Type") != "image/png" || h.Get("X-Content-Type-Options") != "nosniff" ||
		h.Get("Content-Security-Policy") != "sandbox" || h.Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("headers = %v", h)
	}

	before := fetches.Load()
	tampered := strings.Replace(link, url.QueryEscape("/logo.png"), url.QueryEscape("/other.png"), 1)
	for name, c := range map[string]struct{ link, cookie string }{
		"other user": {link, bob},
		"tampered":   {tampered, ann},
		"unsigned":   {"/api/media/search/logo?u=" + url.QueryEscape(site.URL+"/logo.png"), ann},
	} {
		if rec := s.req("GET", c.link, "", withCookie(c.cookie)); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d", name, rec.Code)
		}
	}
	if fetches.Load() != before {
		t.Fatal("fetched without a valid signature")
	}
	if rec := s.req("GET", link, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}

	if rec := s.req("GET", logoLink(site.URL+"/page"), "", withCookie(ann)); rec.Code != http.StatusNotFound {
		t.Fatalf("not an image: %d", rec.Code)
	}
}
