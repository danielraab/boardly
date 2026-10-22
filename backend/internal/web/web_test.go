package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

var built = fstest.MapFS{
	"index.html":              {Data: []byte("<html>app</html>")},
	"assets/app-abc.js":       {Data: []byte("console.log(1)")},
	"favicon.ico":             {Data: []byte("ico")},
	"manifest.en.webmanifest": {Data: []byte(`{"name":"Famodash"}`)},
	"assets/nested/x.js":      {Data: []byte("x")},
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := get(NewHandler(built), http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>app</html>" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q", cc)
	}
}

func TestServesAssetWithLongCache(t *testing.T) {
	rec := get(NewHandler(built), http.MethodGet, "/assets/app-abc.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", cc)
	}
}

func TestServesManifestAsManifestJSON(t *testing.T) {
	rec := get(NewHandler(built), http.MethodGet, "/manifest.en.webmanifest")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"name":"Famodash"}` {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestDeepLinkFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/profile", "/assets", "/a/b/c", "/index.html"} {
		rec := get(NewHandler(built), http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != "<html>app</html>" {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
}

func TestMissingBuild(t *testing.T) {
	rec := get(NewHandler(fstest.MapFS{".gitkeep": {}}), http.MethodGet, "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRejectsNonGet(t *testing.T) {
	rec := get(NewHandler(built), http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestEmbeddedHandlerBuilds(t *testing.T) {
	// Works with only dist/.gitkeep present (no frontend build).
	if Handler() == nil {
		t.Fatal("nil handler")
	}
}
