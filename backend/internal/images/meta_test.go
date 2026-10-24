package images

import (
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"boardly/internal/favourites"
)

func TestParseMeta(t *testing.T) {
	base, _ := url.Parse("https://example.org/docs/page")
	for name, tt := range map[string]struct {
		html string
		want PageMeta
	}{
		"open graph": {
			`<head><meta property="og:description" content="A tour of Go"><meta property="og:image" content="/img/card.png"></head>`,
			PageMeta{"A tour of Go", "https://example.org/img/card.png"},
		},
		"og wins over meta description and twitter": {
			`<meta name="description" content="Plain"><meta name="twitter:image" content="https://cdn.example/t.png">
			 <meta property="og:description" content="OG"><meta property="og:image" content="https://cdn.example/og.png">`,
			PageMeta{"OG", "https://cdn.example/og.png"},
		},
		"fallbacks": {
			`<meta name="description" content="Docs"><meta name="twitter:image" content="t.png">`,
			PageMeta{"Docs", "https://example.org/docs/t.png"},
		},
		"first tag counts": {
			`<meta property="og:image" content="/a.png"><meta property="og:image" content="/b.png">`,
			PageMeta{"", "https://example.org/a.png"},
		},
		"name attribute for og and upper case": {
			`<META NAME="OG:IMAGE" CONTENT="/a.png">`,
			PageMeta{"", "https://example.org/a.png"},
		},
		"whitespace and entities": {
			"<meta name=\"description\" content=\"  Tom &amp; Jerry\n\n  forever \">",
			PageMeta{"Tom & Jerry forever", ""},
		},
		"base href": {
			`<base href="https://static.example/"><meta property="og:image" content="card.jpg">`,
			PageMeta{"", "https://static.example/card.jpg"},
		},
		"protocol-relative": {
			`<meta property="og:image" content="//cdn.example/x.png">`,
			PageMeta{"", "https://cdn.example/x.png"},
		},
		"data and javascript URLs": {
			`<meta property="og:image" content="data:image/png;base64,AAAA"><meta name="twitter:image" content="javascript:alert(1)">`,
			PageMeta{},
		},
		"bad og image falls back to twitter": {
			`<meta property="og:image" content="data:image/png;base64,AAAA"><meta name="twitter:image" content="/t.png">`,
			PageMeta{"", "https://example.org/t.png"},
		},
		"empty og description falls back": {
			`<meta property="og:description" content="  "><meta name="description" content="Docs">`,
			PageMeta{"Docs", ""},
		},
		"only the head": {
			`<head><title>x</title></head><body><meta property="og:image" content="/late.png"></body>`,
			PageMeta{},
		},
		"nothing": {`<html><head><title>Plain page</title></head></html>`, PageMeta{}},
	} {
		if got := ParseMeta(strings.NewReader(tt.html), base); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", name, got, tt.want)
		}
	}
}

func TestParseMetaCutsLongDescriptions(t *testing.T) {
	base, _ := url.Parse("https://example.org/")
	long := strings.Repeat("ö", favourites.MaxDescriptionLength+50)
	got := ParseMeta(strings.NewReader(`<meta name="description" content="`+long+`">`), base)
	if n := utf8.RuneCountInString(got.Description); n != favourites.MaxDescriptionLength {
		t.Fatalf("description has %d characters, want %d", n, favourites.MaxDescriptionLength)
	}
}
