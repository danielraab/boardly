package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"boardly/internal/favourites"
	"boardly/internal/images"
)

var (
	testPNG = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	testICO = append([]byte{0, 0, 1, 0, 1, 0}, make([]byte, 32)...)
	testSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
)

// upload posts data as the raw request body.
func (s *server) upload(data []byte, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/images", bytes.NewReader(data))
	r.RemoteAddr = "192.0.2.1:4000"
	r.Header.Set("Content-Type", "application/octet-stream")
	r.Header.Set("Origin", testOrigin)
	for _, o := range opts {
		o(r)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)
	return rec
}

// allowPrivateFetches lets the server's favicon fetcher reach test servers
// on the loopback address.
func (s *server) allowPrivateFetches() {
	s.favicons.Client = images.NewFavicons(s.images, true).Client
}

// iconSite serves /favicon.ico and counts its requests.
func iconSite(t *testing.T, icon []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		if r.URL.Path == "/favicon.ico" && icon != nil {
			_, _ = w.Write(icon)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func TestUploadAndServeImages(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")

	rec := s.upload(testPNG, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	id := decodeJSONBody[map[string]string](t, rec)["id"]

	// Anyone can load it, without a session.
	rec = s.req("GET", "/api/images/"+id, "")
	h := rec.Header()
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), testPNG) || h.Get("Content-Type") != "image/png" {
		t.Fatalf("get: %d %q", rec.Code, h.Get("Content-Type"))
	}
	if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Content-Security-Policy") != "sandbox" ||
		h.Get("Cache-Control") != uploadCacheControl {
		t.Fatalf("headers = %v", h)
	}

	if rec := s.req("GET", "/api/images/doesnotexist", ""); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Fatalf("unknown: %d %s", rec.Code, rec.Body)
	}

	for name, r := range map[string]struct {
		data   []byte
		cookie string
		status int
		code   string
	}{
		"svg":        {testSVG, ann, http.StatusBadRequest, "unsupported_image"},
		"too large":  {append(bytes.Clone(testPNG), make([]byte, images.MaxSize)...), ann, http.StatusRequestEntityTooLarge, "image_too_large"},
		"no session": {testPNG, "", http.StatusUnauthorized, "unauthenticated"},
		"form type":  {testPNG, ann, http.StatusForbidden, "forbidden_origin"},
	} {
		var opts []func(*http.Request)
		if r.cookie != "" {
			opts = append(opts, withCookie(r.cookie))
		}
		if name == "form type" {
			// A cross-site form could send this type without a preflight.
			opts = append(opts, func(r *http.Request) { r.Header.Set("Content-Type", "multipart/form-data; boundary=x") })
		}
		rec := s.upload(r.data, opts...)
		if rec.Code != r.status || errorCode(t, rec) != r.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, r.status, r.code)
		}
	}
}

func TestFaviconLookup(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	site, requests := iconSite(t, testICO)
	body := `{"url":"` + site.URL + `/some/page"}`

	// The test site is on a loopback address, which is blocked by default.
	rec := s.req("POST", "/api/favicons", body, withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "blocked_address" || requests.Load() != 0 {
		t.Fatalf("blocked: %d %s, %d requests", rec.Code, rec.Body, requests.Load())
	}
	for name, r := range map[string]struct {
		body, cookie string
		status       int
		code         string
	}{
		"invalid url": {`{"url":"javascript:alert(1)"}`, ann, http.StatusBadRequest, "invalid_url"},
		"no session":  {body, "", http.StatusUnauthorized, "unauthenticated"},
	} {
		var opts []func(*http.Request)
		if r.cookie != "" {
			opts = append(opts, withCookie(r.cookie))
		}
		if rec := s.req("POST", "/api/favicons", r.body, opts...); rec.Code != r.status || errorCode(t, rec) != r.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, r.status, r.code)
		}
	}

	s.allowPrivateFetches()
	rec = s.req("POST", "/api/favicons", body, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("allowed: %d %s", rec.Code, rec.Body)
	}
	id := decodeJSONBody[map[string]string](t, rec)["id"]
	fetched := requests.Load()
	rec = s.req("POST", "/api/favicons", `{"url":"`+site.URL+`/other"}`, withCookie(ann))
	if rec.Code != http.StatusOK || decodeJSONBody[map[string]string](t, rec)["id"] != id || requests.Load() != fetched {
		t.Fatalf("cached: %d %s, %d requests", rec.Code, rec.Body, requests.Load())
	}
	if rec := s.req("GET", "/api/images/"+id, ""); rec.Header().Get("Cache-Control") != faviconCacheControl {
		t.Fatalf("favicon Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}

	empty, _ := iconSite(t, nil)
	rec = s.req("POST", "/api/favicons", `{"url":"`+empty.URL+`"}`, withCookie(ann))
	if rec.Code != http.StatusNotFound || errorCode(t, rec) != "favicon_not_found" {
		t.Fatalf("no favicon: %d %s", rec.Code, rec.Body)
	}
}

func TestFavouriteLinksGetFavicons(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	s.allowPrivateFetches()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.favicons.Run(ctx)
	site, _ := iconSite(t, testICO)

	link := s.create(t, ann, map[string]any{"kind": "link", "url": site.URL + "/docs", "title": "Docs"})
	if link.IconImageID != nil {
		t.Fatalf("icon before the fetch: %v", *link.IconImageID)
	}

	// The worker fetches in the background; the tree shows it afterwards.
	deadline := time.Now().Add(5 * time.Second)
	for {
		root := decodeJSONBody[favourites.Node](t, s.req("GET", "/api/favourites", "", withCookie(ann)))
		if n := root.Children[0]; n.IconImageID != nil {
			rec := s.req("GET", "/api/images/"+*n.IconImageID, "")
			if !bytes.Equal(rec.Body.Bytes(), testICO) {
				t.Fatalf("icon = %q", rec.Body)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the link never got a favicon")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// A folder never has one.
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	if folder.IconImageID != nil {
		t.Fatal("folder has an icon")
	}
}

func TestServePreviewImage(t *testing.T) {
	s := newServer(t)
	s.signIn(t, "ann@example.org")
	var userID string
	if err := s.images.DB.QueryRow(`SELECT id FROM users WHERE email = 'ann@example.org'`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 32)...)
	img, err := s.images.Save(context.Background(), images.SourcePreview, userID, jpeg)
	if err != nil {
		t.Fatal(err)
	}
	rec := s.req("GET", "/api/images/"+img.ID, "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" ||
		rec.Header().Get("Cache-Control") != uploadCacheControl {
		t.Fatalf("preview: %d %v", rec.Code, rec.Header())
	}
}

func TestRefreshFavicon(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	site, _ := iconSite(t, nil)
	body := `{"url":"` + site.URL + `/docs"}`

	// Private addresses stay blocked for refreshes too.
	if rec := s.req("POST", "/api/favicons/refresh", body, withCookie(ann)); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "blocked_address" {
		t.Fatalf("blocked: %d %s", rec.Code, rec.Body)
	}
	s.allowPrivateFetches()

	// The site has no favicon yet: the fetch fails and starts the back-off.
	if rec := s.req("POST", "/api/favicons", body, withCookie(ann)); rec.Code != http.StatusNotFound {
		t.Fatalf("lookup: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("POST", "/api/favicons/refresh", body, withCookie(ann)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "favicon_not_found" {
		t.Fatalf("no favicon: %d %s", rec.Code, rec.Body)
	}

	// Now it has one: a refresh fetches it despite the back-off.
	withIcon, _ := iconSite(t, testICO)
	site.Config.Handler = withIcon.Config.Handler
	rec := s.req("POST", "/api/favicons/refresh", body, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rec.Code, rec.Body)
	}
	id := decodeJSONBody[map[string]string](t, rec)["id"]
	if got := s.req("GET", "/api/images/"+id, ""); !bytes.Equal(got.Body.Bytes(), testICO) {
		t.Fatalf("refreshed icon = %q", got.Body)
	}

	for name, r := range map[string]struct {
		body, cookie string
		status       int
		code         string
	}{
		"invalid url": {`{"url":"javascript:alert(1)"}`, ann, http.StatusBadRequest, "invalid_url"},
		"no session":  {body, "", http.StatusUnauthorized, "unauthenticated"},
	} {
		var opts []func(*http.Request)
		if r.cookie != "" {
			opts = append(opts, withCookie(r.cookie))
		}
		if rec := s.req("POST", "/api/favicons/refresh", r.body, opts...); rec.Code != r.status || errorCode(t, rec) != r.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, r.status, r.code)
		}
	}
}

func TestRefreshFaviconRateLimit(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.allowPrivateFetches()
	site, requests := iconSite(t, testICO)
	body := `{"url":"` + site.URL + `"}`
	for i := range refreshBurst {
		if rec := s.req("POST", "/api/favicons/refresh", body, withCookie(ann)); rec.Code != http.StatusOK {
			t.Fatalf("refresh %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	before := requests.Load()
	rec := s.req("POST", "/api/favicons/refresh", body, withCookie(ann))
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "rate_limited" || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over the limit: %d %s %v", rec.Code, rec.Body, rec.Header())
	}
	if requests.Load() != before {
		t.Fatal("a limited refresh made a request")
	}
	// The limit is per user.
	if rec := s.req("POST", "/api/favicons/refresh", body, withCookie(bob)); rec.Code != http.StatusOK {
		t.Fatalf("other user: %d %s", rec.Code, rec.Body)
	}
}

func TestFavouritesJSONHasNoPreviewFields(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/", "title": "Docs"})
	if _, err := s.db.Exec(`UPDATE favourite_nodes SET description = 'Mine' WHERE id = ?`, link.ID); err != nil {
		t.Fatal(err)
	}
	// The Boardly API ignores a description in requests and never returns one.
	rec := s.req("PATCH", "/api/favourites/"+link.ID, `{"title":"Docs 2","description":"Other"}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	tree := s.req("GET", "/api/favourites", "", withCookie(ann)).Body.String()
	for _, body := range []string{rec.Body.String(), tree} {
		for _, field := range []string{"description", "Mine", "Other", "preview"} {
			if strings.Contains(body, field) {
				t.Fatalf("response %s contains %q", body, field)
			}
		}
	}
	var d string
	if err := s.db.QueryRow(`SELECT description FROM favourite_nodes WHERE id = ?`, link.ID).Scan(&d); err != nil || d != "Mine" {
		t.Fatalf("description = %q, %v; want Mine", d, err)
	}
}
