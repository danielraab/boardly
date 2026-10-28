// Package importer reads bookmark export files of Karakeep, Linkwarden and
// web browsers into a favourites.ImportTree, and the notes of Karakeep files
// into a notes.ImportTree. It needs no database: duplicates and existing
// folders and categories are handled by the stores.
package importer

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"boardly/internal/favourites"
	"boardly/internal/notes"
)

// ErrInvalidFile is returned for a file that isn't an export of the format.
var ErrInvalidFile = errors.New("not a valid import file")

// Format is the format of an import file.
type Format string

// Formats.
const (
	FormatKarakeep   Format = "karakeep"
	FormatLinkwarden Format = "linkwarden"
	FormatNetscape   Format = "netscape"
)

// Result is the content of an import file: its links and its notes.
type Result struct {
	Links favourites.ImportTree
	Notes notes.ImportTree
}

// Parse reads the file in the format. ok is false for an unknown format.
func Parse(format Format, r io.Reader) (res Result, ok bool, err error) {
	switch format {
	case FormatKarakeep:
		res, err = ParseKarakeep(r)
	case FormatLinkwarden:
		res.Links, err = ParseLinkwarden(r)
	case FormatNetscape:
		res.Links, err = ParseNetscape(r)
	default:
		return Result{}, false, nil
	}
	return res, true, err
}

// node is a folder while a file is read: its items keep their order, and a
// node without a name is replaced by its items when the tree is built.
type node struct {
	name  string
	items []nodeItem
}

type nodeItem struct {
	folder *node
	link   *favourites.ImportLink
}

func (n *node) addFolder(f *node)                { n.items = append(n.items, nodeItem{folder: f}) }
func (n *node) addLink(l *favourites.ImportLink) { n.items = append(n.items, nodeItem{link: l}) }

// importItems converts the node's items, replacing unnamed folders by their items.
func (n *node) importItems() []favourites.ImportItem {
	var out []favourites.ImportItem
	for _, it := range n.items {
		switch {
		case it.link != nil:
			out = append(out, favourites.ImportItem{Link: it.link})
		case it.folder.name == "":
			out = append(out, it.folder.importItems()...)
		default:
			out = append(out, favourites.ImportItem{Folder: &favourites.ImportFolder{
				Name: it.folder.name, Items: it.folder.importItems(),
			}})
		}
	}
	return out
}

// builder collects a tree and its report.
type builder struct {
	root   node
	report favourites.ImportReport
}

func (b *builder) tree() favourites.ImportTree {
	return favourites.ImportTree{Root: favourites.ImportFolder{Items: b.root.importItems()}, Report: b.report}
}

// entry is a link as a file describes it.
type entry struct {
	url, title                 string
	tags                       []string
	description, note, summary string
	createdAt                  time.Time
}

// link returns the entry as an import link, or nil when its URL is invalid,
// which it counts.
func (b *builder) link(e entry) *favourites.ImportLink {
	u, err := favourites.NormalizeURL(e.url)
	if err != nil {
		b.report.Skipped.InvalidURLs++
		return nil
	}
	l := &favourites.ImportLink{URL: u, Tags: cleanTags(e.tags), CreatedAt: e.createdAt}
	if title := cut(strings.TrimSpace(e.title), favourites.MaxLinkTitleLength, ""); title != "" {
		l.Title = &title
	}
	l.Description = b.describe(e.description, e.note, e.summary)
	return l
}

// describe joins the description, the note and the summary, and cuts the
// result to the maximum description length, which it counts.
func (b *builder) describe(description, note, summary string) *string {
	var parts []string
	if s := strings.TrimSpace(description); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(note); s != "" {
		parts = append(parts, "Note: "+s)
	}
	if s := strings.TrimSpace(summary); s != "" {
		parts = append(parts, "Summary: "+s)
	}
	if len(parts) == 0 {
		return nil
	}
	d := strings.Join(parts, "\n\n")
	if utf8.RuneCountInString(d) > favourites.MaxDescriptionLength {
		d = cut(d, favourites.MaxDescriptionLength, "…")
		b.report.ShortenedDescriptions++
	}
	return &d
}

// folderName trims the name and cuts it to the maximum folder name length.
func folderName(name string) string {
	return cut(strings.TrimSpace(name), favourites.MaxFolderNameLength, "")
}

// cut shortens s to at most n characters, ending with suffix when it was
// shortened.
func cut(s string, n int, suffix string) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-utf8.RuneCountInString(suffix)]) + suffix
}

// cleanTags trims the names and leaves out empty and too long ones.
func cleanTags(names []string) []string {
	var out []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && utf8.RuneCountInString(name) <= favourites.MaxTagNameLength {
			out = append(out, name)
		}
	}
	return out
}

// unixTime reads seconds since the epoch, or milliseconds for values too
// large to be seconds. Zero and negative values give the zero time.
func unixTime(n int64) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	if n > 1e11 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}

// flexTime is a JSON time given as Unix seconds or milliseconds, or as an
// RFC 3339 string. Anything else is the zero time.
type flexTime time.Time

func (t flexTime) time() time.Time { return time.Time(t) }

func (t *flexTime) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*t = flexTime{}
	switch v := v.(type) {
	case float64:
		*t = flexTime(unixTime(int64(v)))
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			*t = flexTime(parsed.UTC())
		} else if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			*t = flexTime(unixTime(n))
		}
	}
	return nil
}

// flexID is a JSON ID given as a string or a number; null and missing are "".
type flexID string

func (id *flexID) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		*id = flexID(v)
	case float64:
		*id = flexID(strconv.FormatFloat(v, 'f', -1, 64))
	default:
		*id = ""
	}
	return nil
}

// tagNames are tags given as names or as {"name"} objects.
type tagNames []string

func (t *tagNames) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		*t = nil
		return nil
	}
	*t = nil
	for _, r := range raw {
		var name string
		if json.Unmarshal(r, &name) == nil {
			*t = append(*t, name)
			continue
		}
		var obj struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(r, &obj) == nil {
			*t = append(*t, obj.Name)
		}
	}
	return nil
}

// optString is a JSON string that may also be null or of another type,
// which both read as "".
type optString string

func (s *optString) UnmarshalJSON(data []byte) error {
	var v string
	if json.Unmarshal(data, &v) == nil {
		*s = optString(v)
	} else {
		*s = ""
	}
	return nil
}

// decodeJSON decodes r into v and returns ErrInvalidFile for invalid JSON.
func decodeJSON(r io.Reader, v any) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return ErrInvalidFile
	}
	return nil
}
