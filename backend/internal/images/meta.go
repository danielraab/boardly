package images

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"boardly/internal/favourites"
)

// PageMeta is what a page says about itself for link previews.
type PageMeta struct {
	// Description is empty when the page has none.
	Description string
	// ImageURL is an absolute http(s) URL, or empty.
	ImageURL string
}

// ParseMeta reads the page's description, from <meta property="og:description">
// or else <meta name="description">, with whitespace collapsed and cut to
// favourites.MaxDescriptionLength characters, and its preview image, from
// og:image or else twitter:image, resolved against base (or the page's
// <base href>). The first tag of each kind counts. Only the <head> is read.
func ParseMeta(r io.Reader, base *url.URL) PageMeta {
	// Values by key, first one wins.
	found := map[string]string{}
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		tag := string(name)
		if tag == "body" {
			break
		}
		if !hasAttr || (tag != "meta" && tag != "base") {
			continue
		}
		attrs := map[string]string{}
		for {
			k, v, more := z.TagAttr()
			attrs[string(k)] = string(v)
			if !more {
				break
			}
		}
		if tag == "base" {
			if b, err := base.Parse(attrs["href"]); err == nil && attrs["href"] != "" {
				base = b
			}
			continue
		}
		// Sites use property= and name= for both Open Graph and Twitter tags.
		key := strings.ToLower(strings.TrimSpace(attrs["property"]))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(attrs["name"]))
		}
		switch key {
		case "og:description", "description", "og:image", "twitter:image":
		default:
			continue
		}
		if _, ok := found[key]; ok {
			continue
		}
		v := attrs["content"]
		if strings.HasSuffix(key, "image") {
			v = imageURL(base, v)
		} else {
			v = collapse(v)
		}
		if v != "" {
			found[key] = v
		}
	}
	return PageMeta{
		Description: first(found["og:description"], found["description"]),
		ImageURL:    first(found["og:image"], found["twitter:image"]),
	}
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// collapse joins the words of s with single spaces and cuts the result to
// favourites.MaxDescriptionLength characters.
func collapse(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > favourites.MaxDescriptionLength {
		s = strings.TrimSpace(string(runes[:favourites.MaxDescriptionLength]))
	}
	return s
}

// imageURL resolves an image reference against base; anything but an http(s)
// URL is dropped.
func imageURL(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := base.Parse(ref)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.String()
}
