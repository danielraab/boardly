package feeds

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var parseNow = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func parseOne(t *testing.T, item string) ParsedItem {
	t.Helper()
	p, err := Parse([]byte(`<?xml version="1.0"?><rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel><title>T</title>`+item+`</channel></rss>`), 100, parseNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 1 {
		t.Fatalf("%d items", len(p.Items))
	}
	return p.Items[0]
}

func TestSummaryIsPlainText(t *testing.T) {
	item := parseOne(t, `<item><title>A &amp; B</title><description><![CDATA[<p>Hello <b>world</b></p><script>x()</script>]]></description></item>`)
	if item.Title != "A & B" {
		t.Fatalf("title = %q", item.Title)
	}
	if item.Summary == nil || *item.Summary != "Hello world" {
		t.Fatalf("summary = %v", item.Summary)
	}
}

func TestDangerousLinkDropped(t *testing.T) {
	if item := parseOne(t, `<item><title>x</title><link>javascript:alert(1)</link></item>`); item.Link != nil {
		t.Fatalf("link = %s", *item.Link)
	}
}

func TestEpisodeAudio(t *testing.T) {
	item := parseOne(t, `<item><title>x</title><enclosure url="http://cdn.example.org/1.mp3" type="audio/mpeg" length="9"/><itunes:duration>01:02:03</itunes:duration></item>`)
	if item.Audio == nil || item.Audio.URL != "https://cdn.example.org/1.mp3" {
		t.Fatalf("%+v", item.Audio)
	}
	if *item.Audio.DurationSeconds != 3723 {
		t.Fatalf("duration = %d", *item.Audio.DurationSeconds)
	}
}

func TestGenericEnclosureWithAudioExtension(t *testing.T) {
	if item := parseOne(t, `<item><title>x</title><enclosure url="https://a.example/1.m4a" type="application/octet-stream"/></item>`); item.Audio == nil {
		t.Fatal("no audio")
	}
	if item := parseOne(t, `<item><title>x</title><enclosure url="https://a.example/1.jpg" type="image/jpeg"/></item>`); item.Audio != nil {
		t.Fatal("an image counted as audio")
	}
}

func TestFutureDateIsClamped(t *testing.T) {
	item := parseOne(t, `<item><title>x</title><pubDate>Wed, 08 Nov 2028 12:00:00 +0000</pubDate></item>`)
	if item.Time.After(parseNow) {
		t.Fatalf("time = %v", item.Time)
	}
}

func TestEntryWithoutTitle(t *testing.T) {
	item := parseOne(t, `<item><description>`+strings.Repeat("a", 150)+`</description></item>`)
	if len(item.Title) != untitledTitleLength {
		t.Fatalf("title has %d characters", len(item.Title))
	}
}

func TestEntryWithoutTitleAndSummaryIsLeftOut(t *testing.T) {
	p, err := Parse([]byte(`<rss version="2.0"><channel><title>T</title><item><guid>1</guid></item></channel></rss>`), 100, parseNow)
	if err != nil || len(p.Items) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestIdentityFallback(t *testing.T) {
	if got := parseOne(t, `<item><guid>g</guid><link>https://a.example/1</link><title>x</title></item>`).Identity; got != "g" {
		t.Fatal(got)
	}
	if got := parseOne(t, `<item><link>https://a.example/1</link><title>x</title></item>`).Identity; got != "https://a.example/1" {
		t.Fatal(got)
	}
	if got := parseOne(t, `<item><title>x</title></item>`).Identity; !strings.HasPrefix(got, "sha256:") {
		t.Fatal(got)
	}
}

func TestAtomAndJSON(t *testing.T) {
	atom := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>A</title><link rel="self" href="https://evil.example/other"/><link rel="alternate" href="https://a.example/"/><entry><id>e1</id><title>E</title><link href="https://a.example/1"/><updated>2026-01-01T00:00:00Z</updated></entry></feed>`
	p, err := Parse([]byte(atom), 100, parseNow)
	if err != nil || p.Title != "A" || len(p.Items) != 1 || p.Items[0].Identity != "e1" {
		t.Fatalf("%+v %v", p, err)
	}
	if p.SiteURL == nil || *p.SiteURL != "https://a.example/" {
		t.Fatalf("site = %v", p.SiteURL)
	}
	jsonFeed := `{"version":"https://jsonfeed.org/version/1.1","title":"J","items":[{"id":"j1","title":"X","url":"https://a.example/j","date_published":"2026-01-01T00:00:00Z"}]}`
	p, err = Parse([]byte(jsonFeed), 100, parseNow)
	if err != nil || p.Title != "J" || len(p.Items) != 1 || p.Items[0].Identity != "j1" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestNotAFeed(t *testing.T) {
	if _, err := Parse([]byte(`hello`), 100, parseNow); err == nil {
		t.Fatal("no error")
	}
}

func TestNormalizeURL(t *testing.T) {
	for in, want := range map[string]string{
		"HTTPS://Example.ORG:443/feed#top": "https://example.org/feed",
		"http://example.org:80":            "http://example.org/",
		"https://example.org/a?token=X":    "https://example.org/a?token=X",
		"https://example.org:8443/a":       "https://example.org:8443/a",
	} {
		if got, err := normalizeURL(in); err != nil || got != want {
			t.Errorf("%s: %s %v", in, got, err)
		}
	}
	for _, bad := range []string{"ftp://a.example/", "javascript:alert(1)", "/relative", ""} {
		if _, err := normalizeURL(bad); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

func TestDiscoverIgnoresOtherLinks(t *testing.T) {
	base, _ := url.Parse("https://a.example/blog/")
	got, ok := discover([]byte(`<html><head><link rel="alternate" hreflang="de" href="/de"><link rel="alternate" type="application/atom+xml" href="feed.atom"></head></html>`), base)
	if !ok || got != "https://a.example/blog/feed.atom" {
		t.Fatalf("%q %v", got, ok)
	}
	if _, ok := discover([]byte(`<html><body><link rel="alternate" type="application/rss+xml" href="/x"></body></html>`), base); ok {
		t.Fatal("a link in the body counted")
	}
}

func TestFetchLimits(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, 2048))
	}))
	defer big.Close()
	f := NewFetcher(true)
	f.MaxSize = 1024
	if _, err := f.Get(ctx, big.URL, Conditional{}); err == nil {
		t.Fatal("an oversized feed was accepted")
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer slow.Close()
	f = NewFetcher(true)
	f.Timeout = 50 * time.Millisecond
	if _, err := f.Get(ctx, slow.URL, Conditional{}); err == nil {
		t.Fatal("no timeout")
	}
}
