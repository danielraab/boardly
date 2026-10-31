package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const index = `<html><head><title>Boardly</title><meta name="application-name" content="Boardly" /></head>app</html>`

var built = fstest.MapFS{
	"index.html":              {Data: []byte(index)},
	"assets/app-abc.js":       {Data: []byte("console.log(1)")},
	"favicon.ico":             {Data: []byte("ico")},
	"manifest.en.webmanifest": {Data: []byte(`{"name":"Boardly","short_name":"Boardly","lang":"en"}`)},
	"manifest.de.webmanifest": {Data: []byte(`{"name":"Boardly","short_name":"Boardly","lang":"de"}`)},
	"assets/nested/x.js":      {Data: []byte("x")},
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := get(NewHandler(built, "Boardly", "", ""), http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != index {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q", cc)
	}
}

func TestServesAssetWithLongCache(t *testing.T) {
	rec := get(NewHandler(built, "Boardly", "", ""), http.MethodGet, "/assets/app-abc.js")
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
	rec := get(NewHandler(built, "Boardly", "", ""), http.MethodGet, "/manifest.en.webmanifest")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q", cc)
	}
}

func TestIndexCarriesAppTitle(t *testing.T) {
	h := NewHandler(built, "Boardly Staging", "", "")
	want := `<html><head><title>Boardly Staging</title><meta name="application-name" content="Boardly Staging" /></head>app</html>`
	for _, p := range []string{"/", "/settings", "/index.html"} {
		rec := get(h, http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
}

func TestIndexEscapesAppTitle(t *testing.T) {
	rec := get(NewHandler(built, `A & B <test> "x"`, "", ""), http.MethodGet, "/")
	want := `<html><head><title>A &amp; B &lt;test&gt; &#34;x&#34;</title><meta name="application-name" content="A &amp; B &lt;test&gt; &#34;x&#34;" /></head>app</html>`
	if rec.Body.String() != want {
		t.Fatalf("got %q", rec.Body.String())
	}
}

func TestManifestsCarryAppTitle(t *testing.T) {
	h := NewHandler(built, "Boardly Staging", "", "")
	for _, lang := range []string{"en", "de"} {
		rec := get(h, http.MethodGet, "/manifest."+lang+".webmanifest")
		var manifest map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
			t.Fatalf("%s: %v in %q", lang, err, rec.Body.String())
		}
		want := map[string]string{"name": "Boardly Staging", "short_name": "Boardly Staging", "lang": lang}
		if len(manifest) != len(want) || manifest["name"] != want["name"] || manifest["short_name"] != want["short_name"] || manifest["lang"] != lang {
			t.Fatalf("%s: got %v, want %v", lang, manifest, want)
		}
	}
}

func TestIndexCarriesAnalyticsSnippet(t *testing.T) {
	snippet := `<script defer src="https://stats.example.org/script.js" data-website-id="abc"></script>`
	h := NewHandler(built, "Boardly", snippet, "")
	want := `<html><head><title>Boardly</title><meta name="application-name" content="Boardly" />` + snippet + `</head>app</html>`
	for _, p := range []string{"/", "/settings", "/index.html"} {
		rec := get(h, http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
	for _, p := range []string{"/assets/app-abc.js", "/manifest.en.webmanifest"} {
		if body := get(h, http.MethodGet, p).Body.String(); strings.Contains(body, "stats.example.org") {
			t.Fatalf("%s: got snippet in %q", p, body)
		}
	}
}

func TestIndexKeepsAnalyticsSnippetUnescaped(t *testing.T) {
	snippet := `<script>window.a = 1 && 2 < 3;</script>`
	rec := get(NewHandler(built, "Boardly", snippet, ""), http.MethodGet, "/")
	if body := rec.Body.String(); !strings.Contains(body, snippet+"</head>") {
		t.Fatalf("got %q", body)
	}
}

func TestDeepLinkFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/profile", "/assets", "/a/b/c", "/index.html"} {
		rec := get(NewHandler(built, "Boardly", "", ""), http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != index {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
}

func TestMissingBuild(t *testing.T) {
	rec := get(NewHandler(fstest.MapFS{".gitkeep": {}}, "Boardly", "", ""), http.MethodGet, "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRejectsNonGet(t *testing.T) {
	rec := get(NewHandler(built, "Boardly", "", ""), http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestEmbeddedHandlerBuilds(t *testing.T) {
	// Works with only dist/.gitkeep present (no frontend build).
	if Handler("Boardly", "", "") == nil {
		t.Fatal("nil handler")
	}
}

func TestEmbeddedIndexHasTitleMarkers(t *testing.T) {
	data, err := dist.ReadFile("dist/index.html")
	if err != nil {
		t.Skip("frontend not built")
	}
	for _, marker := range []string{titleMarker, metaMarker, headEnd} {
		if !strings.Contains(string(data), marker) {
			t.Fatalf("dist/index.html lacks %q", marker)
		}
	}
}

func TestIndexCarriesMapStyle(t *testing.T) {
	fsys := fstest.MapFS{"index.html": {Data: []byte("<head>" + mapStyleMarker + "</head>")}}
	rec := get(NewHandler(fsys, "Boardly", "", "https://maps.example.org/style.json?key=a&b"), http.MethodGet, "/")
	want := `<head><meta name="boardly-map-style" content="https://maps.example.org/style.json?key=a&amp;b" /></head>`
	if rec.Body.String() != want {
		t.Fatalf("got %q", rec.Body.String())
	}
	rec = get(NewHandler(fsys, "Boardly", "", ""), http.MethodGet, "/")
	if rec.Body.String() != "<head>"+mapStyleMarker+"</head>" {
		t.Fatalf("empty style URL changed the default: %q", rec.Body.String())
	}
}
