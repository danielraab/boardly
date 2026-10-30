package karakeep

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"boardly/internal/attachments"
	"boardly/internal/favourites"
	"boardly/internal/notes"
	"boardly/internal/trpc"
)

// TextContent is the content of a text bookmark: a note as Markdown.
type TextContent struct {
	Type      string  `json:"type"`
	Text      string  `json:"text"`
	SourceURL *string `json:"sourceUrl"`
}

// AssetContent is the content of an asset bookmark: a note whose content is
// one image or PDF.
type AssetContent struct {
	Type      string  `json:"type"`
	AssetType string  `json:"assetType"`
	AssetID   string  `json:"assetId"`
	FileName  string  `json:"fileName"`
	Size      int64   `json:"size"`
	SourceURL *string `json:"sourceUrl"`
	Content   *string `json:"content"`
}

// Asset is a file of a bookmark.
type Asset struct {
	ID        string `json:"id"`
	AssetType string `json:"assetType"`
	FileName  string `json:"fileName"`
}

// noteAsset returns the attachment of a note whose content is one image or
// PDF, and its Karakeep asset type, "image" or "pdf".
func noteAsset(n notes.Note) (notes.NoteAttachment, string, bool) {
	if len(n.Content.Blocks) != 1 || n.Content.Blocks[0].Type != notes.BlockAttachment {
		return notes.NoteAttachment{}, "", false
	}
	for _, a := range n.Attachments {
		if a.ID != n.Content.Blocks[0].ID {
			continue
		}
		if a.ContentType == "application/pdf" {
			return a, "pdf", true
		}
		if t, ok := attachments.TypeOf(a.ContentType); ok && t.Group() == "image" {
			return a, "image", true
		}
	}
	return notes.NoteAttachment{}, "", false
}

// errNoteInList is returned for list changes of a note: notes aren't in lists.
var errNoteInList = &InputError{"Notes can't be in lists in Boardly"}

// noteBookmark is a note as a text bookmark, or as an asset bookmark when
// its content is one image or PDF.
func (s *Service) noteBookmark(userID string, n notes.Note) Bookmark {
	if a, assetType, ok := noteAsset(n); ok {
		b := s.textBookmark(userID, n)
		b.Content = AssetContent{Type: "asset", AssetType: assetType, AssetID: a.ID, FileName: a.Name, Size: a.Size}
		b.Assets = []Asset{{ID: a.ID, AssetType: "bookmarkAsset", FileName: a.Name}}
		return b
	}
	return s.textBookmark(userID, n)
}

// textBookmark is a note as a text bookmark.
func (s *Service) textBookmark(userID string, n notes.Note) Bookmark {
	return Bookmark{
		ID:         n.ID,
		CreatedAt:  trpc.Date(n.CreatedAt),
		ModifiedAt: trpc.Date(n.UpdatedAt),
		Title:      n.Title,
		UserID:     userID,
		Tags:       []Tag{},
		Content:    TextContent{Type: "text", Text: notes.ToMarkdown(n.Content, notes.NoteMarkdownAttachments(s.Origin, n))},
		Archived:   n.Archived,
		Favourited: n.Pinned,
		Assets:     []Asset{},
	}
}

// note returns the note with the given ID if it is in the scope, or
// favourites.ErrNotFound, like a link outside the subtree.
func (s *Service) note(ctx context.Context, sc Scope, id string) (notes.Note, error) {
	n, err := s.Notes.ScopedNote(ctx, sc.Notes, id)
	if errors.Is(err, notes.ErrNotFound) {
		return notes.Note{}, favourites.ErrNotFound
	}
	return n, err
}

// item is a bookmark of a page: a link or a note.
type item struct {
	id      string
	created time.Time
	link    *favourites.ScopedNode
	note    *notes.Note
}

func linkItems(links []*favourites.ScopedNode) []item {
	out := make([]item, len(links))
	for i, l := range links {
		out[i] = item{id: l.ID, created: l.CreatedAt, link: l}
	}
	return out
}

func noteItems(ns []notes.Note) []item {
	out := make([]item, len(ns))
	for i := range ns {
		out[i] = item{id: ns[i].ID, created: ns[i].CreatedAt, note: &ns[i]}
	}
	return out
}

// byCreated orders items oldest first, and by ID for equal times.
func byCreated(a, b item) int {
	return cmp.Or(a.created.Compare(b.created), cmp.Compare(a.id, b.id))
}

// sortItems sorts items newest first, or oldest first when asc is set.
func sortItems(items []item, asc bool) {
	slices.SortFunc(items, func(a, b item) int {
		if asc {
			return byCreated(a, b)
		}
		return byCreated(b, a)
	})
}

// render returns the items as bookmarks, in their order.
func (s *Service) render(ctx context.Context, sc Scope, items []item) ([]Bookmark, error) {
	var links []*favourites.ScopedNode
	for _, it := range items {
		if it.link != nil {
			links = append(links, it.link)
		}
	}
	rendered, err := s.bookmarks(ctx, sc.UserID, links)
	if err != nil {
		return nil, err
	}
	out := make([]Bookmark, len(items))
	next := 0
	for i, it := range items {
		if it.link != nil {
			out[i] = rendered[next]
			next++
		} else {
			out[i] = s.noteBookmark(sc.UserID, *it.note)
		}
	}
	return out, nil
}

// noteMatches reports whether a search term matches a note: its title or its
// text as Markdown contains the term. "#name" terms only match tags, which
// notes don't have.
func noteMatches(n notes.Note, markdown, term string) bool {
	if name, ok := strings.CutPrefix(term, "#"); ok && name != "" {
		return false
	}
	if n.Title != nil && strings.Contains(strings.ToLower(*n.Title), term) {
		return true
	}
	return strings.Contains(strings.ToLower(markdown), term)
}

// linkOrNote returns the link of the subtree or the note of the scope with
// the given ID, or favourites.ErrNotFound.
func (s *Service) linkOrNote(ctx context.Context, sc Scope, t *tree, id string) (*favourites.ScopedNode, *notes.Note, error) {
	if l, err := t.link(id); err == nil {
		return l, nil, nil
	}
	n, err := s.note(ctx, sc, id)
	if err != nil {
		return nil, nil, err
	}
	return nil, &n, nil
}

// createNote is bookmarks.createBookmark for a text bookmark: a note in the
// scope's root category, or uncategorized for all notes.
func (s *Service) createNote(ctx context.Context, sc Scope, in CreateBookmarkInput) (CreatedBookmark, error) {
	var category *string
	switch sc.Notes.Mode {
	case notes.ScopeNone:
		return CreatedBookmark{}, &InputError{"This connected app can't save notes"}
	case notes.ScopeCategory:
		category = &sc.Notes.CategoryID
	}
	if in.Text == nil {
		return CreatedBookmark{}, &InputError{"A text bookmark needs a text"}
	}
	att, err := s.Notes.UserMarkdownAttachments(ctx, sc.UserID, s.Origin)
	if err != nil {
		return CreatedBookmark{}, err
	}
	content, err := json.Marshal(notes.FromMarkdown(*in.Text, att))
	if err != nil {
		return CreatedBookmark{}, err
	}
	n, err := s.Notes.CreateNote(ctx, sc.UserID, notes.NewNote{
		Title: in.Title, CategoryID: category, Content: content,
		Archived: isTrue(in.Archived), Pinned: isTrue(in.Favourited),
	})
	if err != nil {
		return CreatedBookmark{}, err
	}
	return CreatedBookmark{Bookmark: s.noteBookmark(sc.UserID, n)}, nil
}

// createAsset is bookmarks.createBookmark for an asset bookmark: a note
// with only the uploaded file, in the scope's root category, or
// uncategorized for all notes. Its title is the given one or the file name.
func (s *Service) createAsset(ctx context.Context, sc Scope, in CreateBookmarkInput) (CreatedBookmark, error) {
	var category *string
	switch sc.Notes.Mode {
	case notes.ScopeNone:
		return CreatedBookmark{}, &InputError{"This connected app can't save notes"}
	case notes.ScopeCategory:
		category = &sc.Notes.CategoryID
	}
	att, err := s.Notes.UserMarkdownAttachments(ctx, sc.UserID, s.Origin)
	if err != nil {
		return CreatedBookmark{}, err
	}
	a, ok := att.ByID[in.AssetID]
	if !ok {
		return CreatedBookmark{}, &InputError{"Unknown file"}
	}
	title := in.Title
	if title == nil || strings.TrimSpace(*title) == "" {
		name := []rune(a.Name)
		name = name[:min(len(name), notes.MaxNoteTitle)]
		title = new(string)
		*title = strings.TrimSpace(string(name))
	}
	content, err := json.Marshal(notes.Doc{V: notes.DocVersion, Blocks: []notes.Block{{Type: notes.BlockAttachment, ID: a.ID}}})
	if err != nil {
		return CreatedBookmark{}, err
	}
	n, err := s.Notes.CreateNote(ctx, sc.UserID, notes.NewNote{
		Title: title, CategoryID: category, Content: content,
		Archived: isTrue(in.Archived), Pinned: isTrue(in.Favourited),
	})
	if err != nil {
		return CreatedBookmark{}, err
	}
	return CreatedBookmark{Bookmark: s.noteBookmark(sc.UserID, n)}, nil
}

// updateNote is bookmarks.updateBookmark for a note: its title, its text,
// whose tasks keep their IDs where their text is the same, and whether it is
// archived and favourited (pinned).
func (s *Service) updateNote(ctx context.Context, sc Scope, n notes.Note, in UpdateBookmarkInput) (Bookmark, error) {
	if in.URL != nil || in.Description != nil {
		return Bookmark{}, &InputError{"Text bookmarks have no address or description in Boardly"}
	}
	var patch notes.NotePatch
	title, err := clearable(in.Title, "Invalid title")
	if err != nil {
		return Bookmark{}, err
	}
	if title != nil {
		var t *string
		if *title != "" {
			t = title
		}
		patch.Title = &t
	}
	if in.Text != nil && *in.Text != notes.ToMarkdown(n.Content, notes.NoteMarkdownAttachments(s.Origin, n)) {
		att, err := s.Notes.UserMarkdownAttachments(ctx, sc.UserID, s.Origin)
		if err != nil {
			return Bookmark{}, err
		}
		if patch.Content, err = json.Marshal(notes.ReplaceContentFromMarkdown(n.Content, *in.Text, att)); err != nil {
			return Bookmark{}, err
		}
	}
	if in.Archived != nil && *in.Archived != n.Archived {
		patch.Archived = in.Archived
	}
	if in.Favourited != nil && *in.Favourited != n.Pinned {
		patch.Pinned = in.Favourited
	}
	if patch.Title == nil && patch.Content == nil && patch.Archived == nil && patch.Pinned == nil {
		return s.noteBookmark(sc.UserID, n), nil
	}
	updated, err := s.Notes.UpdateNote(ctx, sc.UserID, n.ID, n.Version, patch)
	if err != nil {
		return Bookmark{}, err
	}
	return s.noteBookmark(sc.UserID, updated), nil
}
