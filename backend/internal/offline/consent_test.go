package offline

import (
	"regexp"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
)

func TestHostMatcher(t *testing.T) {
	match := hostMatcher(parseHostList(`
# a comment
cdn.cookielaw.org   # trailing comment

Consent.Cookiebot.com
`))
	for host, want := range map[string]bool{
		"cdn.cookielaw.org":             true,
		"CDN.cookielaw.org.":            true,
		"eu.cdn.cookielaw.org":          true,
		"consent.cookiebot.com":         true,
		"cookielaw.org":                 false,
		"evilcdn.cookielaw.org.example": false,
		"xcdn.cookielaw.org":            false,
		"example.org":                   false,
		"":                              false,
		"# a comment":                   false,
	} {
		if got := match(host); got != want {
			t.Errorf("match(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestConsentHostList(t *testing.T) {
	hosts := parseHostList(consentHostList)
	if len(hosts) < 20 {
		t.Fatalf("only %d consent hosts", len(hosts))
	}
	valid := regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+$`)
	for _, h := range hosts {
		if !valid.MatchString(h) {
			t.Errorf("invalid host %q", h)
		}
	}
	for _, h := range []string{"cdn.cookielaw.org", "consent.cookiebot.com", "app.usercentrics.eu"} {
		if !isConsentHost(h) {
			t.Errorf("%s is not blocked", h)
		}
	}
}

func TestBlocksConsent(t *testing.T) {
	b := NewBrowser("ws://unused", nil)
	const main = cdp.FrameID("MAIN")
	paused := func(u string, kind network.ResourceType, frame cdp.FrameID) fetch.EventRequestPaused {
		return fetch.EventRequestPaused{Request: &network.Request{URL: u}, ResourceType: kind, FrameID: frame}
	}
	for name, tt := range map[string]struct {
		ev   fetch.EventRequestPaused
		want bool
	}{
		"provider script":  {paused("https://cdn.cookielaw.org/scripttemplates/otSDKStub.js", network.ResourceTypeScript, main), true},
		"provider iframe":  {paused("https://cdn.privacy-mgmt.com/index.html", network.ResourceTypeDocument, "CHILD"), true},
		"page on provider": {paused("https://consent.cookiebot.com/", network.ResourceTypeDocument, main), false},
		"other script":     {paused("https://example.org/app.js", network.ResourceTypeScript, main), false},
		"invalid URL":      {paused("http://%zz", network.ResourceTypeScript, main), false},
	} {
		if got := b.blocksConsent(tt.ev, main); got != tt.want {
			t.Errorf("%s: blocksConsent = %v, want %v", name, got, tt.want)
		}
	}
}

func TestConsentStyles(t *testing.T) {
	css := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(consentStyles, "")
	if !strings.Contains(css, "display: none !important") || !strings.Contains(css, "overflow: auto !important") {
		t.Fatal("consent.css lacks its rules")
	}
	for _, rule := range strings.Split(css, "}") {
		selectors, _, ok := strings.Cut(rule, "{")
		if !ok {
			continue
		}
		for sel := range strings.SplitSeq(selectors, ",") {
			sel = strings.TrimSpace(sel)
			switch sel {
			case "", "*", "div", ".modal", ".overlay", ".popup", ".banner", "dialog":
				t.Errorf("generic selector %q in consent.css", sel)
			}
		}
	}
}
