package importer

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"boardly/internal/favourites"
)

// ParseNetscape reads a Netscape bookmark file, as Karakeep, Linkwarden and
// web browsers export it: <H3> starts a folder whose content is the <DL>
// after it, <A> is a link, and a <DD> after a link is its description.
// Everything else, such as scripts a proxy added, is ignored. The file is
// read with the tokenizer, not the HTML parser, because these files are not
// well-formed and the parser would move their elements around.
func ParseNetscape(r io.Reader) (favourites.ImportTree, error) {
	b := &builder{}
	z := html.NewTokenizer(r)

	var (
		isNetscape bool
		stack      = []*node{&b.root}
		pending    *node // folder of the last <H3>, until its <DL>
		inH3, inA  bool
		text       strings.Builder
		current    entry // the <A> being read
		lastLink   *favourites.ImportLink
		inDD       bool
	)
	top := func() *node { return stack[len(stack)-1] }
	// endDD ends a <DD>: its text becomes the last link's description.
	endDD := func() {
		if inDD && lastLink != nil {
			lastLink.Description = b.describe(text.String(), "", "")
		}
		inDD = false
		lastLink = nil
	}

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if err := z.Err(); !errors.Is(err, io.EOF) {
				return favourites.ImportTree{}, err
			}
			break
		}
		switch tt {
		case html.DoctypeToken:
			if strings.EqualFold(strings.TrimSpace(string(z.Text())), "NETSCAPE-Bookmark-file-1") {
				isNetscape = true
			}
		case html.TextToken:
			if inH3 || inA || inDD {
				text.Write(z.Text())
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch atom.Lookup(name) {
			case atom.Dt:
				isNetscape = true
				endDD()
			case atom.Dl:
				endDD()
				if pending != nil {
					stack = append(stack, pending)
					pending = nil
				} else {
					stack = append(stack, top())
				}
			case atom.H3:
				endDD()
				inH3 = true
				text.Reset()
			case atom.A:
				endDD()
				inA = true
				text.Reset()
				current = entry{}
				for hasAttr {
					var key, val []byte
					key, val, hasAttr = z.TagAttr()
					switch string(key) {
					case "href":
						current.url = string(val)
					case "add_date":
						if n, err := strconv.ParseInt(strings.TrimSpace(string(val)), 10, 64); err == nil {
							current.createdAt = unixTime(n)
						}
					case "tags":
						current.tags = strings.Split(string(val), ",")
					}
				}
			case atom.Dd:
				if lastLink != nil {
					inDD = true
					text.Reset()
				}
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch atom.Lookup(name) {
			case atom.Dl:
				endDD()
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
				}
			case atom.H3:
				if inH3 {
					inH3 = false
					pending = &node{name: folderName(text.String())}
					top().addFolder(pending)
				}
			case atom.A:
				if inA {
					inA = false
					current.title = text.String()
					if l := b.link(current); l != nil {
						top().addLink(l)
						lastLink = l
					}
				}
			}
		}
	}
	endDD()
	if !isNetscape {
		return favourites.ImportTree{}, ErrInvalidFile
	}
	return b.tree(), nil
}
