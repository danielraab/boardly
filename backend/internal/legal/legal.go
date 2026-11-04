// Package legal loads the instance's legal pages, the imprint and the privacy
// policy, from the operator's Markdown files or external URLs at startup.
package legal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"boardly/internal/notes"
)

// The known pages.
const (
	Imprint = "imprint"
	Privacy = "privacy"
)

// pageList is every page, in a fixed order.
var pageList = []string{Imprint, Privacy}

// Languages are the supported content languages, in fallback order.
var Languages = []string{"en", "de"}

// MaxFileSize is the largest Markdown file a page accepts.
const MaxFileSize = 1 << 20

var fileNameRe = regexp.MustCompile(`^(imprint|privacy)(?:\.(en|de))?\.md$`)

// page is one legal page: an external URL or documents by language ("" is
// the fallback file).
type page struct {
	url   string
	docs  map[string]notes.Doc
	files []string // the file names, for error messages
}

// Pages holds the configured legal pages. The zero value and nil have none.
type Pages struct {
	pages map[string]*page
}

// Sources names where the pages come from; empty fields are unset.
type Sources struct {
	Dir        string
	ImprintURL string
	PrivacyURL string
}

// Load reads and checks the legal pages. Errors name the variable or file at
// fault.
func Load(src Sources) (*Pages, error) {
	p := &Pages{pages: map[string]*page{}}
	get := func(name string) *page {
		if p.pages[name] == nil {
			p.pages[name] = &page{docs: map[string]notes.Doc{}}
		}
		return p.pages[name]
	}

	if src.Dir != "" {
		entries, err := os.ReadDir(src.Dir)
		if err != nil {
			return nil, fmt.Errorf("BOARDLY_LEGAL_DIR: %w", err)
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || e.IsDir() || !strings.HasSuffix(name, ".md") {
				continue
			}
			m := fileNameRe.FindStringSubmatch(name)
			if m == nil {
				return nil, fmt.Errorf("BOARDLY_LEGAL_DIR: unknown file %q: expected imprint.md, privacy.md or <page>.<language>.md with a language of %s",
					name, strings.Join(Languages, ", "))
			}
			doc, err := readDoc(filepath.Join(src.Dir, name))
			if err != nil {
				return nil, fmt.Errorf("BOARDLY_LEGAL_DIR: %s: %w", name, err)
			}
			pg := get(m[1])
			pg.docs[m[2]] = doc
			pg.files = append(pg.files, name)
		}
	}

	for _, u := range []struct{ key, page, url string }{
		{"BOARDLY_IMPRINT_URL", Imprint, src.ImprintURL},
		{"BOARDLY_PRIVACY_URL", Privacy, src.PrivacyURL},
	} {
		if u.url == "" {
			continue
		}
		pg := get(u.page)
		if len(pg.files) > 0 {
			return nil, fmt.Errorf("%s: the %s has both a URL and files in BOARDLY_LEGAL_DIR (%s); use one of them",
				u.key, u.page, strings.Join(pg.files, ", "))
		}
		pg.url = u.url
	}
	return p, nil
}

// readDoc reads one Markdown file and converts it to a note document.
func readDoc(path string) (notes.Doc, error) {
	f, err := os.Open(path)
	if err != nil {
		return notes.Doc{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return notes.Doc{}, err
	}
	if !info.Mode().IsRegular() {
		return notes.Doc{}, errors.New("not a regular file")
	}
	if info.Size() > MaxFileSize {
		return notes.Doc{}, fmt.Errorf("larger than %d bytes", MaxFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return notes.Doc{}, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return notes.Doc{}, errors.New("is empty")
	}
	return notes.FromMarkdown(string(data), nil), nil
}

// Lookup returns a file page's document for the requested language, and the
// language of the file it came from ("" for the fallback <page>.md). The order
// is the requested language, the fallback, then the other languages. ok is
// false for an unknown page, a page that isn't configured and an external page.
func (p *Pages) Lookup(name, lang string) (doc notes.Doc, docLang string, ok bool) {
	pg := p.page(name)
	if pg == nil || pg.url != "" {
		return notes.Doc{}, "", false
	}
	order := []string{""}
	if slices.Contains(Languages, lang) {
		order = []string{lang, ""}
	}
	order = append(order, Languages...)
	for _, l := range order {
		if d, found := pg.docs[l]; found {
			return d, l, true
		}
	}
	return notes.Doc{}, "", false
}

// Href is where links to the page point: its path in the web app for a file
// page, its URL for an external page, or "" when it isn't configured.
func (p *Pages) Href(name string) string {
	pg := p.page(name)
	switch {
	case pg == nil:
		return ""
	case pg.url != "":
		return pg.url
	case len(pg.docs) > 0:
		return "/" + name
	}
	return ""
}

func (p *Pages) page(name string) *page {
	if p == nil {
		return nil
	}
	return p.pages[name]
}

// Names returns the known page names.
func Names() []string { return slices.Clone(pageList) }
