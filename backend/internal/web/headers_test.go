package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

func hashSource(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// scriptSrc returns the sources of the script-src directive of policy.
func scriptSrc(t *testing.T, policy string) []string {
	t.Helper()
	for _, d := range strings.Split(policy, ";") {
		if fields := strings.Fields(d); len(fields) > 0 && fields[0] == "script-src" {
			return fields[1:]
		}
	}
	t.Fatalf("no script-src in %q", policy)
	return nil
}

func checkSecurityHeaders(t *testing.T, h http.Header) {
	t.Helper()
	policy := h.Get("Content-Security-Policy")
	for _, d := range []string{"default-src 'self'", "object-src 'none'", "base-uri 'self'", "form-action 'self'",
		"frame-ancestors 'self'", "worker-src 'self' blob:", "img-src *", "media-src *", "connect-src *", "frame-src *"} {
		if !strings.Contains(policy, d) {
			t.Errorf("policy %q lacks %q", policy, d)
		}
	}
	if src := scriptSrc(t, policy); len(src) == 0 || src[0] != "'self'" {
		t.Errorf("script-src = %q, want 'self' first", src)
	}
	for k, want := range map[string]string{
		"X-Frame-Options":        "SAMEORIGIN",
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	h := NewHandler(built, Options{AppTitle: "Boardly"})
	for _, path := range []string{"/", "/settings", "/assets/app-abc.js", "/manifest.en.webmanifest", "/favicon.ico"} {
		t.Run(path, func(t *testing.T) {
			rec := get(h, http.MethodGet, path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			checkSecurityHeaders(t, rec.Header())
		})
	}
	t.Run("rejected method", func(t *testing.T) {
		checkSecurityHeaders(t, get(h, http.MethodPost, "/").Header())
	})
	t.Run("frontend not built", func(t *testing.T) {
		checkSecurityHeaders(t, get(NewHandler(fstest.MapFS{".gitkeep": {}}, Options{AppTitle: "Boardly"}), http.MethodGet, "/").Header())
	})
}

func TestPolicyWithoutInlineScripts(t *testing.T) {
	src := scriptSrc(t, get(NewHandler(built, Options{AppTitle: "Boardly"}), http.MethodGet, "/").Header().Get("Content-Security-Policy"))
	if len(src) != 1 {
		t.Fatalf("script-src = %q, want only 'self'", src)
	}
}

// The real index.html has an inline theme script; its hash must be allowed,
// or the theme would not apply before first paint.
func TestPolicyAllowsThemeScript(t *testing.T) {
	data, err := os.ReadFile("../../../frontend/index.html")
	if err != nil {
		t.Skip("frontend sources not available")
	}
	m := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindSubmatch(data)
	if m == nil {
		t.Fatal("frontend/index.html has no inline script")
	}
	h := NewHandler(fstest.MapFS{"index.html": {Data: data}}, Options{AppTitle: "Boardly"})
	src := scriptSrc(t, get(h, http.MethodGet, "/").Header().Get("Content-Security-Policy"))
	if want := hashSource(string(m[1])); !strings.Contains(strings.Join(src, " "), want) {
		t.Fatalf("script-src %q lacks the theme script's hash %s", src, want)
	}
}

func TestPolicyAllowsAnalyticsSnippet(t *testing.T) {
	snippet := `<script defer src="https://stats.example.org/script.js" data-website-id="abc"></script>` +
		`<script>window.a = 1 && 2;</script>` +
		`<script type="text/plain" data-category="analytics">track();</script>` +
		`<script src="/local.js"></script>`
	h := NewHandler(built, Options{AppTitle: "Boardly", AnalyticsSnippet: snippet, CSPScriptSrc: []string{"https://cdn.example.org"}})
	src := strings.Join(scriptSrc(t, get(h, http.MethodGet, "/").Header().Get("Content-Security-Policy")), " ")
	for _, want := range []string{"https://stats.example.org", hashSource("window.a = 1 && 2;"), hashSource("track();"), "https://cdn.example.org"} {
		if !strings.Contains(src, want) {
			t.Errorf("script-src %q lacks %s", src, want)
		}
	}
	if strings.Contains(src, "/local.js") {
		t.Errorf("script-src %q has a relative source; 'self' covers it", src)
	}
}

func TestPolicyListsSourcesOnce(t *testing.T) {
	doc := []byte(`<head><script src="https://a.example/1.js"></script><script src="https://a.example/2.js"></script><script>x()</script><script>x()</script></head>`)
	src := scriptSrc(t, Policy(doc, []string{"https://a.example"}))
	if want := []string{"'self'", "https://a.example", hashSource("x()")}; strings.Join(src, " ") != strings.Join(want, " ") {
		t.Fatalf("script-src = %q, want %q", src, want)
	}
}
