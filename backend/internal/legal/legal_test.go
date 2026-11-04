package legal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dir writes files (name → content) into a new directory and returns it.
func dir(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(d, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

// docText returns the JSON of the page's document for lang, and its language.
func docText(t *testing.T, p *Pages, page, lang string) (string, string) {
	t.Helper()
	doc, docLang, ok := p.Lookup(page, lang)
	if !ok {
		t.Fatalf("Lookup(%q, %q) found nothing", page, lang)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(data), docLang
}

func TestLoadFiles(t *testing.T) {
	d := dir(t, map[string]string{
		"imprint.md":    "# Imprint fallback",
		"imprint.en.md": "# Imprint English",
		"privacy.md":    "# Privacy",
	})
	p, err := Load(Sources{Dir: d})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Href(Imprint); got != "/imprint" {
		t.Errorf("Href(imprint) = %q", got)
	}
	if got := p.Href(Privacy); got != "/privacy" {
		t.Errorf("Href(privacy) = %q", got)
	}
	if text, lang := docText(t, p, Imprint, "en"); !strings.Contains(text, "Imprint English") || lang != "en" {
		t.Errorf("imprint en = %s (%q)", text, lang)
	}
}

func TestLoadNothing(t *testing.T) {
	p, err := Load(Sources{})
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range Names() {
		if p.Href(page) != "" {
			t.Errorf("Href(%s) = %q, want empty", page, p.Href(page))
		}
		if _, _, ok := p.Lookup(page, "en"); ok {
			t.Errorf("Lookup(%s) found a page", page)
		}
	}
	var nilPages *Pages
	if nilPages.Href(Imprint) != "" {
		t.Error("nil Pages has an imprint")
	}
}

func TestLoadOnlyOnePage(t *testing.T) {
	p, err := Load(Sources{Dir: dir(t, map[string]string{"imprint.de.md": "Impressum"})})
	if err != nil {
		t.Fatal(err)
	}
	if p.Href(Imprint) != "/imprint" || p.Href(Privacy) != "" {
		t.Fatalf("Href = %q, %q", p.Href(Imprint), p.Href(Privacy))
	}
}

func TestLoadExternalAndFile(t *testing.T) {
	p, err := Load(Sources{
		Dir:        dir(t, map[string]string{"privacy.md": "Privacy"}),
		ImprintURL: "https://example.at/impressum",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Href(Imprint); got != "https://example.at/impressum" {
		t.Errorf("Href(imprint) = %q", got)
	}
	if _, _, ok := p.Lookup(Imprint, "de"); ok {
		t.Error("external imprint has content")
	}
	if got := p.Href(Privacy); got != "/privacy" {
		t.Errorf("Href(privacy) = %q", got)
	}
}

func TestLoadIgnoresOtherFiles(t *testing.T) {
	d := dir(t, map[string]string{
		"imprint.md": "Imprint",
		"README.txt": "notes",
		".DS_Store":  "x",
		".hidden.md": "x",
	})
	if err := os.Mkdir(filepath.Join(d, "old"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "old", "impressum.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := Load(Sources{Dir: d})
	if err != nil {
		t.Fatal(err)
	}
	if p.Href(Imprint) != "/imprint" {
		t.Fatal("imprint missing")
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name  string
		src   func(t *testing.T) Sources
		names []string // the error must contain each
	}{
		{"unknown file", func(t *testing.T) Sources {
			return Sources{Dir: dir(t, map[string]string{"impressum.md": "x"})}
		}, []string{"impressum.md"}},
		{"unsupported language", func(t *testing.T) Sources {
			return Sources{Dir: dir(t, map[string]string{"imprint.fr.md": "x"})}
		}, []string{"imprint.fr.md"}},
		{"file and URL", func(t *testing.T) Sources {
			return Sources{Dir: dir(t, map[string]string{"imprint.de.md": "x"}), ImprintURL: "https://example.at/impressum"}
		}, []string{"BOARDLY_IMPRINT_URL", "imprint.de.md"}},
		{"missing directory", func(t *testing.T) Sources {
			return Sources{Dir: filepath.Join(t.TempDir(), "legal")}
		}, []string{"BOARDLY_LEGAL_DIR"}},
		{"directory is a file", func(t *testing.T) Sources {
			d := dir(t, map[string]string{"legal": "x"})
			return Sources{Dir: filepath.Join(d, "legal")}
		}, []string{"BOARDLY_LEGAL_DIR"}},
		{"empty file", func(t *testing.T) Sources {
			return Sources{Dir: dir(t, map[string]string{"privacy.md": "\n  \n\t\n"})}
		}, []string{"privacy.md", "empty"}},
		{"too large", func(t *testing.T) Sources {
			return Sources{Dir: dir(t, map[string]string{"privacy.md": strings.Repeat("a", MaxFileSize+1)})}
		}, []string{"privacy.md"}},
		{"subdirectory named like a page", func(t *testing.T) Sources {
			d := t.TempDir()
			if err := os.Mkdir(filepath.Join(d, "imprint.md"), 0o700); err != nil {
				t.Fatal(err)
			}
			// A subdirectory is skipped even when its name ends in .md.
			return Sources{Dir: d}
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(tt.src(t))
			if tt.names == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, n := range tt.names {
				if !strings.Contains(err.Error(), n) {
					t.Errorf("error %q does not name %s", err, n)
				}
			}
		})
	}
}

func TestLookupLanguage(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		lang     string
		want     string
		wantLang string
	}{
		{"requested language", map[string]string{"imprint.md": "fallback", "imprint.de.md": "german"}, "de", "german", "de"},
		{"fallback file", map[string]string{"imprint.md": "fallback", "imprint.de.md": "german"}, "en", "fallback", ""},
		{"only another language", map[string]string{"imprint.de.md": "german"}, "en", "german", "de"},
		{"English before others", map[string]string{"imprint.en.md": "english", "imprint.de.md": "german"}, "fr", "english", "en"},
		{"no language", map[string]string{"imprint.md": "fallback", "imprint.de.md": "german"}, "", "fallback", ""},
		{"unsupported without fallback", map[string]string{"imprint.de.md": "german"}, "xx", "german", "de"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Load(Sources{Dir: dir(t, tt.files)})
			if err != nil {
				t.Fatal(err)
			}
			text, lang := docText(t, p, Imprint, tt.lang)
			if !strings.Contains(text, `"`+tt.want+`"`) || lang != tt.wantLang {
				t.Fatalf("got %s (%q), want %q (%q)", text, lang, tt.want, tt.wantLang)
			}
		})
	}
	if _, _, ok := (&Pages{}).Lookup("terms", "en"); ok {
		t.Error("unknown page found")
	}
}

func TestRawHTMLStaysText(t *testing.T) {
	p, err := Load(Sources{Dir: dir(t, map[string]string{
		"privacy.md": "Before <b>bold</b> after\n\n<script>alert(1)</script>\n",
	})})
	if err != nil {
		t.Fatal(err)
	}
	text, _ := docText(t, p, Privacy, "en")
	// The tags are kept as characters inside text runs, never as markup.
	for _, want := range []string{`<b>`, `<script>alert(1)</script>`} {
		if !strings.Contains(text, jsonEscape(want)) {
			t.Errorf("document %s does not contain %q as text", text, want)
		}
	}
	if strings.Contains(text, `"type":"html"`) {
		t.Errorf("document has an HTML block: %s", text)
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}

// TestExampleDirectoryLoads keeps the templates in docs/legal-example usable
// as BOARDLY_LEGAL_DIR.
func TestExampleDirectoryLoads(t *testing.T) {
	p, err := Load(Sources{Dir: filepath.Join("..", "..", "..", "docs", "legal-example")})
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range Names() {
		if p.Href(page) != "/"+page {
			t.Errorf("Href(%s) = %q", page, p.Href(page))
		}
	}
}
