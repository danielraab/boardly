package karakeep

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"boardly/internal/favourites"
	"boardly/internal/notes"
	"boardly/internal/trpc"
)

// Paging limits, as in Karakeep.
const (
	defaultLimit = 20
	maxLimit     = 100
)

// BookmarksInput is the input of bookmarks.getBookmarks.
type BookmarksInput struct {
	Archived   *bool  `json:"archived"`
	Favourited *bool  `json:"favourited"`
	TagID      string `json:"tagId"`
	ListID     string `json:"listId"`
	Limit      int    `json:"limit"`
	Cursor     *struct {
		CreatedAt time.Time `json:"createdAt"`
		ID        string    `json:"id"`
	} `json:"cursor"`
	SortOrder string `json:"sortOrder"`
}

// Cursor points after the last bookmark of a page of getBookmarks.
type Cursor struct {
	CreatedAt trpc.Date `json:"createdAt"`
	ID        string    `json:"id"`
}

// BookmarkPage is a page of bookmarks.getBookmarks.
type BookmarkPage struct {
	Bookmarks  []Bookmark `json:"bookmarks"`
	NextCursor *Cursor    `json:"nextCursor"`
}

// SearchInput is the input of bookmarks.searchBookmarks.
type SearchInput struct {
	Text   string `json:"text"`
	Limit  int    `json:"limit"`
	Cursor *struct {
		Offset int `json:"offset"`
	} `json:"cursor"`
	SortOrder string `json:"sortOrder"`
}

// SearchCursor is the offset of the next page of searchBookmarks.
type SearchCursor struct {
	Ver    int `json:"ver"`
	Offset int `json:"offset"`
}

// SearchPage is a page of bookmarks.searchBookmarks.
type SearchPage struct {
	Bookmarks  []Bookmark    `json:"bookmarks"`
	NextCursor *SearchCursor `json:"nextCursor"`
}

func limit(n int) (int, error) {
	switch {
	case n == 0:
		return defaultLimit, nil
	case n < 1 || n > maxLimit:
		return 0, &InputError{"The limit must be between 1 and 100"}
	}
	return n, nil
}

// Bookmarks is bookmarks.getBookmarks: the links of the subtree and the
// notes of the scope, or only links when filtered by list or tag.
func (s *Service) Bookmarks(ctx context.Context, sc Scope, in BookmarksInput) (BookmarkPage, error) {
	n, err := limit(in.Limit)
	if err != nil {
		return BookmarkPage{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return BookmarkPage{}, err
	}
	links := t.links()
	if in.ListID != "" {
		if _, err := t.list(in.ListID); err != nil {
			return BookmarkPage{}, err
		}
		links = slices.DeleteFunc(links, func(l *favourites.ScopedNode) bool { return l.ParentID != in.ListID })
	}
	if in.TagID != "" {
		if _, ok := t.tags()[in.TagID]; !ok {
			return BookmarkPage{}, favourites.ErrNotFound
		}
		links = slices.DeleteFunc(links, func(l *favourites.ScopedNode) bool {
			return !slices.ContainsFunc(l.Tags, func(tag favourites.Tag) bool { return tag.ID == in.TagID })
		})
	}
	items := linkItems(links)
	// Notes are in no list and have no tags.
	if in.ListID == "" && in.TagID == "" {
		ns, err := s.Notes.ScopedNotes(ctx, sc.Notes)
		if err != nil {
			return BookmarkPage{}, err
		}
		items = append(items, noteItems(ns)...)
	}
	// Only notes can be archived or favourited (pinned).
	items = slices.DeleteFunc(items, func(it item) bool {
		archived := it.note != nil && it.note.Archived
		pinned := it.note != nil && it.note.Pinned
		return (in.Archived != nil && *in.Archived != archived) || (in.Favourited != nil && *in.Favourited != pinned)
	})
	asc := in.SortOrder == "asc"
	sortItems(items, asc)
	if c := in.Cursor; c != nil {
		after := item{id: c.ID, created: c.CreatedAt}
		items = slices.DeleteFunc(items, func(it item) bool {
			if asc {
				return byCreated(it, after) <= 0
			}
			return byCreated(it, after) >= 0
		})
	}

	page := BookmarkPage{}
	if len(items) > n {
		items = items[:n]
		last := items[n-1]
		page.NextCursor = &Cursor{CreatedAt: trpc.Date(last.created), ID: last.id}
	}
	page.Bookmarks, err = s.render(ctx, sc, items)
	return page, err
}

// Search is bookmarks.searchBookmarks: every whitespace-separated term must
// match, "#name" a tag of that name and any other term a part of the title,
// URL or a tag name, ignoring case, or of a note's title or Markdown.
func (s *Service) Search(ctx context.Context, sc Scope, in SearchInput) (SearchPage, error) {
	n, err := limit(in.Limit)
	if err != nil {
		return SearchPage{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return SearchPage{}, err
	}
	terms := strings.Fields(strings.ToLower(in.Text))
	links := slices.DeleteFunc(t.links(), func(l *favourites.ScopedNode) bool {
		for _, term := range terms {
			if !matches(l, term) {
				return true
			}
		}
		return false
	})
	ns, err := s.Notes.ScopedNotes(ctx, sc.Notes)
	if err != nil {
		return SearchPage{}, err
	}
	ns = slices.DeleteFunc(ns, func(n notes.Note) bool {
		md := notes.ToMarkdown(n.Content)
		for _, term := range terms {
			if !noteMatches(n, md, term) {
				return true
			}
		}
		return false
	})
	items := append(linkItems(links), noteItems(ns)...)
	sortItems(items, in.SortOrder == "asc")

	offset := 0
	if in.Cursor != nil {
		offset = min(max(in.Cursor.Offset, 0), len(items))
	}
	items = items[offset:]
	page := SearchPage{}
	if len(items) > n {
		items = items[:n]
		page.NextCursor = &SearchCursor{Ver: 1, Offset: offset + n}
	}
	page.Bookmarks, err = s.render(ctx, sc, items)
	return page, err
}

func matches(l *favourites.ScopedNode, term string) bool {
	if name, ok := strings.CutPrefix(term, "#"); ok && name != "" {
		return slices.ContainsFunc(l.Tags, func(t favourites.Tag) bool { return strings.ToLower(t.Name) == name })
	}
	if l.Title != nil && strings.Contains(strings.ToLower(*l.Title), term) {
		return true
	}
	if strings.Contains(strings.ToLower(*l.URL), term) {
		return true
	}
	return slices.ContainsFunc(l.Tags, func(t favourites.Tag) bool { return strings.Contains(strings.ToLower(t.Name), term) })
}

// findURL returns a link of the subtree with the same URL (see
// favourites.SameURL), or nil.
func (t *tree) findURL(raw string) *favourites.ScopedNode {
	items := linkItems(t.links())
	sortItems(items, false)
	for _, it := range items {
		if favourites.SameURL(*it.link.URL, raw) {
			return it.link
		}
	}
	return nil
}

// CheckURL is bookmarks.checkUrl: the ID of a bookmark with this URL, or nil.
func (s *Service) CheckURL(ctx context.Context, sc Scope, rawURL string) (*string, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return nil, err
	}
	if l := t.findURL(rawURL); l != nil {
		return &l.ID, nil
	}
	return nil, nil
}

// GetBookmark is bookmarks.getBookmark.
func (s *Service) GetBookmark(ctx context.Context, sc Scope, id string) (Bookmark, error) {
	return s.bookmark(ctx, sc, id)
}

// CreateBookmarkInput is the input of bookmarks.createBookmark. Fields that
// Boardly ignores (source, crawlPriority, createdAt, …) aren't decoded.
type CreateBookmarkInput struct {
	Type       string  `json:"type"`
	URL        string  `json:"url"`
	Text       *string `json:"text"`
	Title      *string `json:"title"`
	Archived   *bool   `json:"archived"`
	Favourited *bool   `json:"favourited"`
	Note       *string `json:"note"`
	Summary    *string `json:"summary"`
}

// CreatedBookmark is the result of bookmarks.createBookmark.
type CreatedBookmark struct {
	Bookmark
	AlreadyExists bool `json:"alreadyExists"`
}

// CreateBookmark is bookmarks.createBookmark: a new link first in the
// token's root folder, or the existing bookmark with the same URL, or a new
// note for a text bookmark.
func (s *Service) CreateBookmark(ctx context.Context, sc Scope, in CreateBookmarkInput) (CreatedBookmark, error) {
	if in.Type != "link" && in.Type != "text" {
		return CreatedBookmark{}, &InputError{"Boardly only stores links and notes, not " + in.Type + " bookmarks"}
	}
	link := in.Type == "link"
	if err := checkUnsupported(map[string]bool{
		"archiving links": link && isTrue(in.Archived),
		"favourite links": link && isTrue(in.Favourited),
		"notes":           nonEmpty(in.Note),
		"summaries":       nonEmpty(in.Summary),
	}); err != nil {
		return CreatedBookmark{}, err
	}
	if in.Type == "text" {
		return s.createNote(ctx, sc, in)
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return CreatedBookmark{}, err
	}
	if existing := t.findURL(in.URL); existing != nil {
		b, err := s.bookmarks(ctx, sc.UserID, []*favourites.ScopedNode{existing})
		if err != nil {
			return CreatedBookmark{}, err
		}
		return CreatedBookmark{Bookmark: b[0], AlreadyExists: true}, nil
	}
	node, err := s.Favourites.Create(ctx, sc.UserID, favourites.NewNode{
		ParentID: sc.RootID, Kind: favourites.KindLink, URL: &in.URL, Title: in.Title, First: true,
	})
	if err != nil {
		return CreatedBookmark{}, err
	}
	b, err := s.bookmark(ctx, sc, node.ID)
	return CreatedBookmark{Bookmark: b}, err
}

// UpdateBookmarkInput is the input of bookmarks.updateBookmark. Title and
// Description are raw JSON to tell a missing value from null.
type UpdateBookmarkInput struct {
	BookmarkID    string          `json:"bookmarkId"`
	Title         json.RawMessage `json:"title"`
	URL           *string         `json:"url"`
	Archived      *bool           `json:"archived"`
	Favourited    *bool           `json:"favourited"`
	Summary       *string         `json:"summary"`
	Note          *string         `json:"note"`
	Description   json.RawMessage `json:"description"`
	Author        *string         `json:"author"`
	Publisher     *string         `json:"publisher"`
	Text          *string         `json:"text"`
	CreatedAt     json.RawMessage `json:"createdAt"`
	DatePublished json.RawMessage `json:"datePublished"`
	DateModified  json.RawMessage `json:"dateModified"`
	AssetContent  json.RawMessage `json:"assetContent"`
}

// UpdateBookmark is bookmarks.updateBookmark: title, URL and description of
// a link, or title, text, archived and favourited of a note.
func (s *Service) UpdateBookmark(ctx context.Context, sc Scope, in UpdateBookmarkInput) (Bookmark, error) {
	if err := checkUnsupported(map[string]bool{
		"summaries":                  nonEmpty(in.Summary),
		"notes":                      nonEmpty(in.Note),
		"authors":                    nonEmpty(in.Author),
		"publishers":                 nonEmpty(in.Publisher),
		"changing the creation date": isSet(in.CreatedAt),
		"publication dates":          isSet(in.DatePublished),
		"modification dates":         isSet(in.DateModified),
		"file bookmarks":             isSet(in.AssetContent),
	}); err != nil {
		return Bookmark{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return Bookmark{}, err
	}
	_, note, err := s.linkOrNote(ctx, sc, t, in.BookmarkID)
	if err != nil {
		return Bookmark{}, err
	}
	if note != nil {
		return s.updateNote(ctx, sc, *note, in)
	}
	if err := checkUnsupported(map[string]bool{
		"archiving links": isTrue(in.Archived),
		"favourite links": isTrue(in.Favourited),
	}); err != nil {
		return Bookmark{}, err
	}
	if nonEmpty(in.Text) {
		return Bookmark{}, &InputError{"Only text bookmarks have a text"}
	}
	patch := favourites.Patch{URL: in.URL}
	if patch.Title, err = clearable(in.Title, "Invalid title"); err != nil {
		return Bookmark{}, err
	}
	if patch.Description, err = clearable(in.Description, "Invalid description"); err != nil {
		return Bookmark{}, err
	}
	if patch.Title != nil || patch.URL != nil || patch.Description != nil {
		if _, err := s.Favourites.Update(ctx, sc.UserID, in.BookmarkID, patch); err != nil {
			return Bookmark{}, err
		}
	}
	return s.bookmark(ctx, sc, in.BookmarkID)
}

// clearable decodes an optional string field: nil when it is missing, and an
// empty string, which clears the value, for null.
func clearable(raw json.RawMessage, invalid string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	var v *string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, &InputError{invalid}
	}
	if v == nil {
		v = new(string)
	}
	return v, nil
}

// DeleteBookmark is bookmarks.deleteBookmark: it deletes a link or a note.
func (s *Service) DeleteBookmark(ctx context.Context, sc Scope, id string) error {
	t, err := s.load(ctx, sc)
	if err != nil {
		return err
	}
	_, note, err := s.linkOrNote(ctx, sc, t, id)
	if err != nil {
		return err
	}
	if note != nil {
		return s.Notes.DeleteNote(ctx, sc.UserID, id)
	}
	return s.Favourites.Delete(ctx, sc.UserID, id, favourites.DeleteKeepNone)
}

// TagRef names a tag by ID or by name.
type TagRef struct {
	TagID   string `json:"tagId"`
	TagName string `json:"tagName"`
}

// UpdateTagsInput is the input of bookmarks.updateTags.
type UpdateTagsInput struct {
	BookmarkID string   `json:"bookmarkId"`
	Attach     []TagRef `json:"attach"`
	Detach     []TagRef `json:"detach"`
}

// UpdateTagsResult is the result of bookmarks.updateTags.
type UpdateTagsResult struct {
	Attached []string `json:"attached"`
	Detached []string `json:"detached"`
}

// UpdateTags is bookmarks.updateTags.
func (s *Service) UpdateTags(ctx context.Context, sc Scope, in UpdateTagsInput) (UpdateTagsResult, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return UpdateTagsResult{}, err
	}
	link, note, err := s.linkOrNote(ctx, sc, t, in.BookmarkID)
	if err != nil {
		return UpdateTagsResult{}, err
	}
	if note != nil {
		if len(in.Attach) > 0 || len(in.Detach) > 0 {
			return UpdateTagsResult{}, &InputError{"Notes have no tags in Boardly"}
		}
		return UpdateTagsResult{Attached: []string{}, Detached: []string{}}, nil
	}
	visible := t.tags()
	known := map[string]string{} // lower-case name → ID, of visible tags and the link's tags
	for _, tag := range visible {
		known[strings.ToLower(tag.Name)] = tag.ID
	}
	// name resolves a reference to a tag name.
	name := func(ref TagRef) (string, error) {
		if ref.TagID != "" {
			tag, ok := visible[ref.TagID]
			if !ok {
				return "", favourites.ErrNotFound
			}
			return tag.Name, nil
		}
		if strings.TrimSpace(ref.TagName) == "" {
			return "", &InputError{"A tag needs an ID or a name"}
		}
		return strings.TrimSpace(ref.TagName), nil
	}

	var names []string
	for _, tag := range link.Tags {
		names = append(names, tag.Name)
	}
	result := UpdateTagsResult{Attached: []string{}, Detached: []string{}}
	for _, ref := range in.Detach {
		n, err := name(ref)
		if err != nil {
			return UpdateTagsResult{}, err
		}
		names = slices.DeleteFunc(names, func(x string) bool { return strings.EqualFold(x, n) })
		if id, ok := known[strings.ToLower(n)]; ok {
			result.Detached = append(result.Detached, id)
		}
	}
	var attached []string
	for _, ref := range in.Attach {
		n, err := name(ref)
		if err != nil {
			return UpdateTagsResult{}, err
		}
		names = append(names, n)
		attached = append(attached, n)
	}
	if _, err := s.Favourites.Update(ctx, sc.UserID, link.ID, favourites.Patch{Tags: &names}); err != nil {
		return UpdateTagsResult{}, err
	}

	// New tags only have IDs now.
	t, err = s.load(ctx, sc)
	if err != nil {
		return UpdateTagsResult{}, err
	}
	link, err = t.link(in.BookmarkID)
	if err != nil {
		return UpdateTagsResult{}, err
	}
	for _, n := range attached {
		for _, tag := range link.Tags {
			if strings.EqualFold(tag.Name, strings.TrimSpace(n)) {
				result.Attached = append(result.Attached, tag.ID)
			}
		}
	}
	return result, nil
}

// tags returns the tags on the subtree's links, by ID.
func (t *tree) tags() map[string]favourites.Tag {
	out := map[string]favourites.Tag{}
	for _, l := range t.links() {
		for _, tag := range l.Tags {
			out[tag.ID] = tag
		}
	}
	return out
}

func isTrue(b *bool) bool          { return b != nil && *b }
func nonEmpty(s *string) bool      { return s != nil && strings.TrimSpace(*s) != "" }
func isSet(r json.RawMessage) bool { return r != nil && string(r) != "null" }

// checkUnsupported returns an InputError for the first set feature, in name order.
func checkUnsupported(features map[string]bool) error {
	names := make([]string, 0, len(features))
	for name, set := range features {
		if set {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	slices.Sort(names)
	return unsupported(names[0])
}
