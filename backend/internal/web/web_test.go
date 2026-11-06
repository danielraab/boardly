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

// notice is the cookie notice variant every document carries while consent
// is off and no analytics snippet is set.
const notice = `<meta name="boardly-cookie-notice" content="necessary" />`

// served is index as the handler serves it without further options.
const served = `<html><head><title>Boardly</title><meta name="application-name" content="Boardly" />` + notice + `</head>app</html>`

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
	rec := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != served {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q", cc)
	}
}

func TestServesAssetWithLongCache(t *testing.T) {
	rec := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, "/assets/app-abc.js")
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
	rec := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, "/manifest.en.webmanifest")
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
	h := NewHandler(built, Options{AppTitle: "Boardly Staging"})
	want := `<html><head><title>Boardly Staging</title><meta name="application-name" content="Boardly Staging" />` + notice + `</head>app</html>`
	for _, p := range []string{"/", "/settings", "/index.html"} {
		rec := get(h, http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
}

func TestIndexEscapesAppTitle(t *testing.T) {
	rec := get(NewHandler(built, Options{AppTitle: `A & B <test> "x"`}), http.MethodGet, "/")
	want := `<html><head><title>A &amp; B &lt;test&gt; &#34;x&#34;</title><meta name="application-name" content="A &amp; B &lt;test&gt; &#34;x&#34;" />` + notice + `</head>app</html>`
	if rec.Body.String() != want {
		t.Fatalf("got %q", rec.Body.String())
	}
}

func TestManifestsCarryAppTitle(t *testing.T) {
	h := NewHandler(built, Options{AppTitle: "Boardly Staging"})
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
	h := NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: snippet})
	want := `<html><head><title>Boardly</title><meta name="application-name" content="Boardly" />` +
		`<meta name="boardly-cookie-notice" content="cookieless-analytics" />` + snippet + `</head>app</html>`
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
	rec := get(NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: snippet}), http.MethodGet, "/")
	if body := rec.Body.String(); !strings.Contains(body, snippet+"</head>") {
		t.Fatalf("got %q", body)
	}
}

func TestDeepLinkFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/profile", "/assets", "/a/b/c", "/index.html"} {
		rec := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != served {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
}

func TestMissingBuild(t *testing.T) {
	rec := get(NewHandler(fstest.MapFS{".gitkeep": {}}, Options{AppTitle: "Boardly"}), http.MethodGet, "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRejectsNonGet(t *testing.T) {
	rec := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestEmbeddedHandlerBuilds(t *testing.T) {
	// Works with only dist/.gitkeep present (no frontend build).
	if Handler(Options{AppTitle: "Boardly"}) == nil {
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
	rec := get(NewHandler(fsys, Options{AppTitle: "Boardly", MapStyleURL: "https://maps.example.org/style.json?key=a&b"}), http.MethodGet, "/")
	want := `<head><meta name="boardly-map-style" content="https://maps.example.org/style.json?key=a&amp;b" />` + notice + `</head>`
	if rec.Body.String() != want {
		t.Fatalf("got %q", rec.Body.String())
	}
	rec = get(NewHandler(fsys, Options{AppTitle: "Boardly"}), http.MethodGet, "/")
	if rec.Body.String() != "<head>"+mapStyleMarker+notice+"</head>" {
		t.Fatalf("empty style URL changed the default: %q", rec.Body.String())
	}
}

func TestIndexCarriesLegalLinks(t *testing.T) {
	h := NewHandler(built, Options{
		AppTitle:    "Boardly",
		ImprintHref: "/imprint",
		PrivacyHref: "https://example.at/datenschutz?a=1&b=2",
	})
	want := `<html><head><title>Boardly</title><meta name="application-name" content="Boardly" />` +
		`<meta name="boardly-imprint" content="/imprint" />` +
		`<meta name="boardly-privacy" content="https://example.at/datenschutz?a=1&amp;b=2" />` +
		notice + `</head>app</html>`
	for _, p := range []string{"/", "/settings", "/imprint"} {
		rec := get(h, http.MethodGet, p)
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}
	for _, p := range []string{"/assets/app-abc.js", "/manifest.en.webmanifest"} {
		if body := get(h, http.MethodGet, p).Body.String(); strings.Contains(body, "boardly-imprint") {
			t.Fatalf("%s: got legal meta tag in %q", p, body)
		}
	}
}

func TestIndexLegalLinksBeforeAnalyticsSnippet(t *testing.T) {
	snippet := `<script src="https://stats.example.org/s.js"></script>`
	rec := get(NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: snippet, ImprintHref: "/imprint"}), http.MethodGet, "/")
	if body := rec.Body.String(); !strings.Contains(body, `<meta name="boardly-imprint" content="/imprint" />`+
		`<meta name="boardly-cookie-notice" content="cookieless-analytics" />`+snippet+"</head>") {
		t.Fatalf("got %q", body)
	}
}

func TestIndexWithoutLegalLinks(t *testing.T) {
	body := get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, "/").Body.String()
	if strings.Contains(body, "boardly-imprint") || strings.Contains(body, "boardly-privacy") {
		t.Fatalf("got legal meta tags in %q", body)
	}
}

func TestIndexCarriesConsentSettings(t *testing.T) {
	h := NewHandler(built, Options{
		AppTitle: "Boardly", AnalyticsSnippet: "<script></script>",
		ConsentService: `A & "B"`, ConsentCookies: []string{"_ga", "_ga_*"}, ConsentRevision: 42,
	})
	want := `<meta name="boardly-consent-analytics" content="A &amp; &#34;B&#34;" />` +
		`<meta name="boardly-consent-cookies" content="_ga,_ga_*" />` +
		`<meta name="boardly-consent-revision" content="42" />` +
		`<script></script></head>`
	for _, p := range []string{"/", "/settings"} {
		body := get(h, http.MethodGet, p).Body.String()
		if !strings.Contains(body, want) || strings.Contains(body, "boardly-cookie-notice") {
			t.Fatalf("%s: got %q", p, body)
		}
	}
	if body := get(h, http.MethodGet, "/assets/app-abc.js").Body.String(); strings.Contains(body, "boardly-consent") {
		t.Fatalf("got consent tags in a bundle: %q", body)
	}

	body := get(NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: "<script></script>", ConsentService: "Matomo", ConsentRevision: 1}),
		http.MethodGet, "/").Body.String()
	if strings.Contains(body, "boardly-consent-cookies") || !strings.Contains(body, `content="Matomo"`) {
		t.Fatalf("without cookies: got %q", body)
	}
}

func TestIndexCarriesCookieNoticeVariant(t *testing.T) {
	for _, tt := range []struct {
		name, snippet, want string
	}{
		{"no snippet", "", "necessary"},
		{"snippet without consent", "<script></script>", "cookieless-analytics"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := get(NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: tt.snippet}), http.MethodGet, "/settings").Body.String()
			if !strings.Contains(body, `<meta name="boardly-cookie-notice" content="`+tt.want+`" />`) || strings.Contains(body, "boardly-consent-") {
				t.Fatalf("got %q", body)
			}
		})
	}
}
