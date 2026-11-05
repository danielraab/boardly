package offline

import (
	"bytes"
	"html/template"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// MinArticleText is the least text, in characters with whitespace collapsed,
// that counts as an article.
const MinArticleText = 200

// maxTitleLength is the longest title kept, in characters.
const maxTitleLength = 1000

// policy keeps the safe parts of an article: text structure, tables, links
// and images with absolute http(s) URLs. Scripts, styles, frames, embeds,
// forms and event handlers are removed. Links open in a new tab.
var policy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(false)
	p.AllowURLSchemes("http", "https")
	p.AllowElements("article", "section", "header", "footer", "main", "aside", "div", "span", "p", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "code", "kbd", "samp", "var",
		"em", "strong", "b", "i", "u", "s", "del", "ins", "mark", "small", "sub", "sup", "abbr", "cite", "q",
		"time", "figure", "figcaption", "picture")
	p.AllowLists()
	p.AllowTables()
	p.AllowImages()
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").Globally()
	p.AllowAttrs("cite").OnElements("blockquote", "q", "del", "ins")
	p.AllowAttrs("datetime").OnElements("time", "del", "ins")
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}()

// page is the document an article is stored as.
var articleTemplate = template.Must(template.New("article").Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="referrer" content="no-referrer">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
:root { color-scheme: light dark; }
body { margin: 0 auto; max-width: 42rem; padding: 1.5rem 1rem 3rem; font: 1.0625rem/1.65 Georgia, "Times New Roman", serif; color: #1f2328; background: #fff; overflow-wrap: break-word; }
@media (prefers-color-scheme: dark) { body { color: #e6e6e6; background: #16181d; } a { color: #8ab4f8; } }
header { margin-bottom: 2rem; font-family: system-ui, sans-serif; }
header h1 { margin: 0 0 .25rem; font-size: 1.75rem; line-height: 1.25; }
header a { font-size: .875rem; opacity: .75; word-break: break-all; }
img, video, picture { max-width: 100%; height: auto; }
pre { overflow-x: auto; padding: .75rem; border-radius: .375rem; background: rgba(127,127,127,.12); }
code { font-size: .9em; }
blockquote { margin: 1rem 0; padding-left: 1rem; border-left: 3px solid rgba(127,127,127,.4); }
table { border-collapse: collapse; display: block; overflow-x: auto; }
td, th { border: 1px solid rgba(127,127,127,.35); padding: .25rem .5rem; }
figcaption { font-size: .875rem; opacity: .75; }
</style>
</head>
<body>
<header>
<h1>{{.Title}}</h1>
<a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a>
</header>
<main>
{{.Body}}
</main>
</body>
</html>
`))

// Article is a readable article ready to be stored.
type Article struct {
	// Title is the article's title, or the page's <title>; may be empty.
	Title string
	// HTML is the complete, sanitized document.
	HTML []byte
}

// ExtractArticle finds the readable article of the HTML page data at
// pageURL, keeps only its safe content and wraps it in the article document.
// ok is false when the page has less than MinArticleText of readable text.
func ExtractArticle(data []byte, pageURL *url.URL) (Article, bool) {
	parser := readability.NewParser()
	art, err := parser.Parse(bytes.NewReader(data), pageURL)
	if err != nil || art.Node == nil {
		return Article{}, false
	}
	var raw bytes.Buffer
	if err := art.RenderHTML(&raw); err != nil {
		return Article{}, false
	}
	title := truncate(collapse(art.Title()), maxTitleLength)
	body, ok := sanitize(raw.String(), MinArticleText)
	if !ok {
		return Article{}, false
	}
	doc, err := render(title, pageURL.String(), art.Language(), body)
	if err != nil {
		return Article{}, false
	}
	return Article{Title: title, HTML: doc}, true
}

// SanitizeDocument cleans an article document that comes from elsewhere,
// such as an import: only its <main> (or <body>) content is kept, sanitized
// like a captured article, and put into a new article document. ok is false
// when no text is left.
func SanitizeDocument(data []byte, title, pageURL string) ([]byte, bool) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	content := find(doc, atom.Main)
	if content == nil {
		content = find(doc, atom.Body)
	}
	if content == nil {
		return nil, false
	}
	var inner bytes.Buffer
	for c := content.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&inner, c); err != nil {
			return nil, false
		}
	}
	body, ok := sanitize(inner.String(), 1)
	if !ok {
		return nil, false
	}
	if u, err := url.Parse(pageURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		pageURL = ""
	}
	out, err := render(truncate(collapse(title), maxTitleLength), pageURL, "", body)
	return out, err == nil
}

// sanitize applies the policy to fragment, marks images to load lazily and
// without referrer, and reports whether at least minText characters of text
// are left.
func sanitize(fragment string, minText int) (string, bool) {
	clean := policy.Sanitize(fragment)
	ctx := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(clean), ctx)
	if err != nil {
		return "", false
	}
	var text strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			text.WriteString(n.Data)
			text.WriteByte(' ')
		case n.Type == html.ElementNode && n.DataAtom == atom.Img:
			setAttr(n, "referrerpolicy", "no-referrer")
			setAttr(n, "loading", "lazy")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	var out bytes.Buffer
	for _, n := range nodes {
		walk(n)
		if err := html.Render(&out, n); err != nil {
			return "", false
		}
	}
	if utf8.RuneCountInString(collapse(text.String())) < minText {
		return "", false
	}
	return out.String(), true
}

func render(title, pageURL, lang, body string) ([]byte, error) {
	if lang == "" || strings.ContainsAny(lang, "\"'<> ") {
		lang = "und"
	}
	var out bytes.Buffer
	err := articleTemplate.Execute(&out, struct {
		Title, Lang string
		URL         template.URL
		Body        template.HTML
	}{
		Title: title,
		Lang:  lang,
		URL:   template.URL(pageURL), // checked to be http(s) by the caller
		// The body went through the sanitizer just before.
		Body: template.HTML(body), //nolint:gosec
	})
	return out.Bytes(), err
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func find(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := find(c, a); f != nil {
			return f
		}
	}
	return nil
}

// collapse joins runs of whitespace into single spaces and trims.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// readAll reads r up to limit bytes; tooLarge reports more.
func readAll(r io.Reader, limit int64) (data []byte, tooLarge bool, err error) {
	data, err = io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > limit {
		return nil, true, nil
	}
	return data, false, nil
}

func htmlUnescape(s string) string { return html.UnescapeString(s) }
