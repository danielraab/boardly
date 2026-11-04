package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boardly/internal/legal"
)

func newLegalHandler(t *testing.T, files map[string]string, imprintURL string) http.Handler {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pages, err := legal.Load(legal.Sources{Dir: dir, ImprintURL: imprintURL})
	if err != nil {
		t.Fatal(err)
	}
	return New(Deps{DB: fakeDB{}, Web: fakeWeb, Legal: pages})
}

func TestGetLegalPage(t *testing.T) {
	h := newLegalHandler(t, map[string]string{
		"privacy.md":    "# Privacy fallback",
		"privacy.de.md": "# Datenschutz",
	}, "https://example.at/impressum")

	tests := []struct {
		path     string
		wantText string
		wantLang any
	}{
		{"/api/legal/privacy?lang=de", "Datenschutz", "de"},
		{"/api/legal/privacy?lang=en", "Privacy fallback", nil},
		{"/api/legal/privacy", "Privacy fallback", nil},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// No session cookie: the route is public.
			rec := do(t, h, http.MethodGet, tt.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["page"] != "privacy" || body["language"] != tt.wantLang {
				t.Fatalf("page/language = %v/%v", body["page"], body["language"])
			}
			if doc, _ := json.Marshal(body["doc"]); !strings.Contains(string(doc), tt.wantText) || !strings.Contains(string(doc), `"blocks"`) {
				t.Fatalf("doc = %s", doc)
			}
		})
	}
}

func TestGetLegalPageNotFound(t *testing.T) {
	h := newLegalHandler(t, map[string]string{"privacy.md": "Privacy"}, "https://example.at/impressum")
	for _, path := range []string{"/api/legal/imprint", "/api/legal/terms"} {
		rec := do(t, h, http.MethodGet, path)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
			t.Fatalf("%s: got %d %s", path, rec.Code, rec.Body)
		}
	}

	// Without any legal configuration.
	rec := do(t, New(Deps{DB: fakeDB{}, Web: fakeWeb}), http.MethodGet, "/api/legal/privacy")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unconfigured: got %d %s", rec.Code, rec.Body)
	}
}
