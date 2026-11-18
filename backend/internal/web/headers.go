package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// openDirectives allow what the app loads from origins that users and
// operators choose: station streams, map tiles, favicons, preview images and
// provider embeds. Spelling them out keeps default-src 'self' from closing
// them. Inline style attributes are used throughout the app.
const openDirectives = "img-src * data: blob:; media-src * data: blob:; connect-src *; frame-src *; " +
	"font-src * data:; style-src * 'unsafe-inline'"

// fixedDirectives forbid plugins, a foreign <base>, form posts to other
// sites and framing by other sites. MapLibre runs its worker from a
// same-origin URL or a blob.
const fixedDirectives = "worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'"

// Policy returns the Content-Security-Policy of the web app for the served
// HTML document doc: scripts may come from Boardly's origin, be one of doc's
// inline scripts (by hash, also those the consent manager activates later),
// come from the origin of one of doc's script tags, or from extra.
func Policy(doc []byte, extra []string) string {
	sources := []string{"'self'"}
	for _, s := range append(scriptSources(doc), extra...) {
		if !slices.Contains(sources, s) {
			sources = append(sources, s)
		}
	}
	return "default-src 'self'; script-src " + strings.Join(sources, " ") + "; " + fixedDirectives + "; " + openDirectives
}

// scriptSources returns the sha256 hash source of the text of every inline
// <script> in doc and the origin of every absolute script src, in document
// order. Scripts of any type count, because the consent manager turns
// type="text/plain" scripts into running ones with the same text.
func scriptSources(doc []byte) []string {
	root, err := html.Parse(bytes.NewReader(doc))
	if err != nil {
		return nil
	}
	var out []string
	for n := range root.Descendants() {
		if n.Type != html.ElementNode || n.DataAtom != atom.Script {
			continue
		}
		if src, ok := attr(n, "src"); ok {
			if u, err := url.Parse(strings.TrimSpace(src)); err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
				out = append(out, u.Scheme+"://"+u.Host)
			}
			continue
		}
		var text strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				text.WriteString(c.Data)
			}
		}
		sum := sha256.Sum256([]byte(text.String()))
		out = append(out, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return out
}

func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

// setSecurityHeaders sets the headers every response of the web app carries.
func setSecurityHeaders(h http.Header, policy string) {
	h.Set("Content-Security-Policy", policy)
	h.Set("X-Frame-Options", "SAMEORIGIN")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
}
