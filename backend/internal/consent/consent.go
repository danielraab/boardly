// Package consent checks the analytics consent settings at startup and
// derives the consent revision the web app asks again on.
package consent

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"boardly/internal/config"
)

// Category is the consent category the snippet's scripts must name.
const Category = "analytics"

// CheckSnippet reports an error unless snippet holds only comments,
// whitespace and <script type="text/plain" data-category="analytics">
// elements, which the browser doesn't run before the consent manager
// activates them. Anything else would load without consent.
func CheckSnippet(snippet string) error {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(snippet), body)
	if err != nil {
		return fmt.Errorf("BOARDLY_ANALYTICS_SNIPPET: %w", err)
	}
	for _, n := range nodes {
		switch {
		case n.Type == html.CommentNode:
		case n.Type == html.TextNode && strings.TrimSpace(n.Data) == "":
		case n.Type == html.ElementNode && n.DataAtom == atom.Script:
			if !strings.EqualFold(attr(n, "type"), "text/plain") || !strings.EqualFold(attr(n, "data-category"), Category) {
				return fmt.Errorf(`BOARDLY_ANALYTICS_SNIPPET: %s needs type="text/plain" and data-category="analytics", so it only runs after consent`, describe(n))
			}
		case n.Type == html.ElementNode:
			return fmt.Errorf("BOARDLY_ANALYTICS_SNIPPET: <%s> would load without consent; with BOARDLY_CONSENT_ANALYTICS the snippet may only hold <script> elements", n.Data)
		default:
			return fmt.Errorf("BOARDLY_ANALYTICS_SNIPPET: text %q outside a <script>; with BOARDLY_CONSENT_ANALYTICS the snippet may only hold <script> elements", strings.TrimSpace(n.Data))
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

// describe names a script for an error message.
func describe(n *html.Node) string {
	if src := attr(n, "src"); src != "" {
		return fmt.Sprintf("<script src=%q>", src)
	}
	return "an inline <script>"
}

// Revision derives the consent revision from the settings that a visitor
// agrees to. It is the same for the same settings and between 1 and 2^31.
// The consent manager treats 0 as "no revisions", so it is never 0.
func Revision(snippet, service string, cookies []string) int {
	h := sha256.Sum256([]byte(snippet + "\x00" + service + "\x00" + strings.Join(cookies, "\x00")))
	return int(binary.BigEndian.Uint32(h[:4])&0x7fffffff) + 1
}

// Check reports an error when consent is on but the snippet is missing or
// would run without consent, or when no privacy policy is configured.
// privacyHref is the privacy policy's link target, empty when it has none.
func Check(c config.Consent, snippet, privacyHref string) error {
	if c.Service == "" {
		return nil
	}
	if snippet == "" {
		return errors.New("BOARDLY_ANALYTICS_SNIPPET: required with BOARDLY_CONSENT_ANALYTICS, set it to the analytics service's scripts")
	}
	if err := CheckSnippet(snippet); err != nil {
		return err
	}
	if privacyHref == "" {
		return errors.New("BOARDLY_CONSENT_ANALYTICS: a privacy policy is required for the consent banner to link to; add privacy.md to BOARDLY_LEGAL_DIR or set BOARDLY_PRIVACY_URL")
	}
	return nil
}
