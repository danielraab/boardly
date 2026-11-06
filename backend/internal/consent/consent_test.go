package consent

import (
	"strings"
	"testing"

	"boardly/internal/config"
)

const marked = `<!-- Google tag -->
<script type="text/plain" data-category="analytics" async src="https://www.googletagmanager.com/gtag/js?id=G-1"></script>
<script type="text/plain" data-category="analytics">
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date()); gtag('config', 'G-1');
</script>`

func TestCheckSnippet(t *testing.T) {
	tests := []struct {
		name    string
		snippet string
		wantErr string
	}{
		{"marked scripts with comments", marked, ""},
		{"upper case values", `<SCRIPT TYPE="TEXT/PLAIN" DATA-CATEGORY="Analytics" src="/a.js"></SCRIPT>`, ""},
		{"unmarked script", `<script async src="https://www.googletagmanager.com/gtag/js?id=G-1"></script>`, "googletagmanager"},
		{"only the type", `<script type="text/plain" src="/a.js"></script>`, "data-category"},
		{"only the category", `<script data-category="analytics">x()</script>`, "inline <script>"},
		{"other category", `<script type="text/plain" data-category="marketing">x()</script>`, "data-category"},
		{"tracking pixel", `<noscript><img src="https://stats.example.org/matomo.php?idsite=1"></noscript>`, "<noscript>"},
		{"image", `<img src="https://stats.example.org/p.gif">`, "<img>"},
		{"link", `<link rel="preconnect" href="https://stats.example.org">`, "<link>"},
		{"iframe", `<iframe src="https://stats.example.org"></iframe>`, "<iframe>"},
		{"text", `hello`, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSnippet(tt.snippet)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), "BOARDLY_ANALYTICS_SNIPPET") || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q should name BOARDLY_ANALYTICS_SNIPPET and %q", err, tt.wantErr)
			}
		})
	}
}

func TestRevision(t *testing.T) {
	base := Revision(marked, "Google Analytics", []string{"_ga", "_ga_*"})
	if base != Revision(marked, "Google Analytics", []string{"_ga", "_ga_*"}) {
		t.Fatal("same settings give different revisions")
	}
	for name, r := range map[string]int{
		"snippet": Revision(strings.ReplaceAll(marked, "G-1", "G-2"), "Google Analytics", []string{"_ga", "_ga_*"}),
		"service": Revision(marked, "Matomo", []string{"_ga", "_ga_*"}),
		"cookies": Revision(marked, "Google Analytics", []string{"_ga"}),
		"split":   Revision(marked, "Google Analytics", []string{"_ga,_ga_*"}),
	} {
		if r == base {
			t.Errorf("changed %s keeps the revision %d", name, r)
		}
	}
	for _, s := range []string{"", "a", marked, strings.Repeat("x", 1000)} {
		if r := Revision(s, "", nil); r < 1 || r > 1<<31 {
			t.Errorf("Revision(%q) = %d, out of range", s, r)
		}
	}
}

func TestCheck(t *testing.T) {
	on := config.Consent{Service: "Google Analytics"}
	tests := []struct {
		name    string
		consent config.Consent
		snippet string
		privacy string
		wantErr string
	}{
		{"off", config.Consent{}, `<script>x()</script>`, "", ""},
		{"valid", on, marked, "/privacy", ""},
		{"no snippet", on, "", "/privacy", "BOARDLY_ANALYTICS_SNIPPET"},
		{"unmarked snippet", on, `<script>x()</script>`, "/privacy", "BOARDLY_ANALYTICS_SNIPPET"},
		{"no privacy policy", on, marked, "", "BOARDLY_CONSENT_ANALYTICS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.consent, tt.snippet, tt.privacy)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %v should name %s", err, tt.wantErr)
			}
		})
	}
}
