package karakeep

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"boardly/internal/favourites"
	"boardly/internal/notes"
)

// withNotes gives the fixture's scope all notes, or those of a category.
func (f *fixture) withNotes(categoryID string) {
	f.sc.Notes = notes.Scope{UserID: f.user, Mode: notes.ScopeAll}
	if categoryID != "" {
		f.sc.Notes = notes.Scope{UserID: f.user, Mode: notes.ScopeCategory, CategoryID: categoryID}
	}
}

func (f *fixture) note(title, md string, category *string) notes.Note {
	f.t.Helper()
	content, _ := json.Marshal(notes.FromMarkdown(md))
	n, err := f.notes.CreateNote(f.ctx, f.user, notes.NewNote{Title: &title, CategoryID: category, Content: content})
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) category(title string) string {
	f.t.Helper()
	c, err := f.notes.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: title})
	if err != nil {
		f.t.Fatal(err)
	}
	return c.ID
}

func titlesOf(bs []Bookmark) []string {
	var out []string
	for _, b := range bs {
		out = append(out, *b.Title)
	}
	return out
}

func TestNotesInBookmarks(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	later := f.folder(f.sc.RootID, "Later")
	f.link(f.sc.RootID, "Old link", "https://example.org/old", "dev")
	f.note("Middle note", "text", nil)
	f.link(later, "New link", "https://example.org/new")
	f.note("Newest note", "more", nil)

	page, err := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := titlesOf(page.Bookmarks), []string{"Newest note", "New link", "Middle note"}; !slices.Equal(got, want) {
		t.Fatalf("first page = %v, want %v", got, want)
	}
	next := page.NextCursor
	in := BookmarksInput{Limit: 3}
	in.Cursor = &struct {
		CreatedAt time.Time `json:"createdAt"`
		ID        string    `json:"id"`
	}{time.Time(next.CreatedAt), next.ID}
	page, err = f.s.Bookmarks(f.ctx, f.sc, in)
	if err != nil || !slices.Equal(titlesOf(page.Bookmarks), []string{"Old link"}) || page.NextCursor != nil {
		t.Fatalf("second page = %v, %v", titlesOf(page.Bookmarks), err)
	}
	if c, ok := page.Bookmarks[0].Content.(LinkContent); !ok || c.Type != "link" {
		t.Errorf("link content = %#v", page.Bookmarks[0].Content)
	}

	for name, in := range map[string]BookmarksInput{"list": {ListID: later}, "tag": {TagID: tagID(t, f, "dev")}} {
		page, err := f.s.Bookmarks(f.ctx, f.sc, in)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range page.Bookmarks {
			if _, ok := b.Content.(TextContent); ok {
				t.Errorf("%s filter returned the note %q", name, *b.Title)
			}
		}
	}
}

func tagID(t *testing.T, f *fixture, name string) string {
	t.Helper()
	tr, err := f.s.load(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	for id, tag := range tr.tags() {
		if tag.Name == name {
			return id
		}
	}
	t.Fatalf("no tag %q", name)
	return ""
}

func TestSearchNotes(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	f.note("Plumbing", "call the **plumber**\n\n#dev", nil)
	f.link(f.sc.RootID, "Go", "https://go.dev", "dev")
	for text, want := range map[string][]string{
		"plumber":  {"Plumbing"},
		"plumbing": {"Plumbing"},
		"#dev":     {"Go"},
		"call go":  nil,
	} {
		page, err := f.s.Search(f.ctx, f.sc, SearchInput{Text: text})
		if err != nil {
			t.Fatal(err)
		}
		if got := titlesOf(page.Bookmarks); !slices.Equal(got, want) {
			t.Errorf("search %q = %v, want %v", text, got, want)
		}
	}
}

func TestCreateTextBookmark(t *testing.T) {
	f := newFixture(t)
	text := "- [ ] buy milk"
	title := "Shopping"

	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Text: &text}); !isInputError(err) {
		t.Errorf("without notes: err = %v, want an input error", err)
	}

	f.withNotes("")
	created, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Title: &title, Text: &text})
	if err != nil || created.AlreadyExists || *created.Title != "Shopping" || created.Content.(TextContent).Text != text {
		t.Fatalf("all notes: %+v, %v", created, err)
	}
	n, _ := f.notes.Note(f.ctx, f.user, created.ID)
	if n.CategoryID != nil || n.Content.Blocks[0].Style != notes.ListTask {
		t.Errorf("note = %+v, want an uncategorized task list", n)
	}

	inbox := f.category("Inbox")
	f.withNotes(inbox)
	created, err = f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Text: &text})
	if n, _ := f.notes.Note(f.ctx, f.user, created.ID); err != nil || n.CategoryID == nil || *n.CategoryID != inbox {
		t.Fatalf("root category: %+v, %v", n, err)
	}

	big := strings.Repeat("x", notes.MaxDocSize)
	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Text: &big}); !errors.Is(err, notes.ErrInvalidDocument) {
		t.Errorf("too large: err = %v", err)
	}
	long := strings.Repeat("t", 201)
	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Title: &long, Text: &text}); !errors.Is(err, notes.ErrInvalidTitle) {
		t.Errorf("long title: err = %v", err)
	}
	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "asset"}); !isInputError(err) {
		t.Errorf("asset: err = %v", err)
	}
}

func isInputError(err error) bool {
	var input *InputError
	return errors.As(err, &input)
}

func TestUpdateNoteBookmark(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	n := f.note("Shopping", "- [x] milk\n- [ ] eggs", nil)
	md := notes.ToMarkdown(n.Content)

	// The same text changes nothing.
	b, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: n.ID, Text: &md})
	if got, _ := f.notes.Note(f.ctx, f.user, n.ID); err != nil || got.Version != 1 || b.ID != n.ID {
		t.Fatalf("same text: version %d, %v", got.Version, err)
	}

	text := "- [x] milk\n- [ ] eggs\n- [ ] bread"
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: n.ID, Title: json.RawMessage(`null`), Text: &text}); err != nil {
		t.Fatal(err)
	}
	got, _ := f.notes.Note(f.ctx, f.user, n.ID)
	items := *got.Content.Blocks[0].Items
	if got.Title != nil || got.Version != 2 || len(items) != 3 || items[0].ID != (*n.Content.Blocks[0].Items)[0].ID {
		t.Errorf("after the edit: %+v", got)
	}

	url := "https://example.org"
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: n.ID, URL: &url}); !isInputError(err) {
		t.Errorf("url on a note: err = %v", err)
	}
	link := f.link(f.sc.RootID, "Go", "https://go.dev")
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: link, Text: &text}); !isInputError(err) {
		t.Errorf("text on a link: err = %v", err)
	}

	// A change made elsewhere in the meantime is a conflict.
	stale := got
	if _, err := f.notes.UpdateNote(f.ctx, f.user, n.ID, got.Version, notes.NotePatch{Title: ptrTo(ptrTo("Web"))}); err != nil {
		t.Fatal(err)
	}
	other := "- [ ] other"
	if _, err := f.s.updateNote(f.ctx, f.sc, stale, UpdateBookmarkInput{BookmarkID: n.ID, Text: &other}); !errors.Is(err, notes.ErrVersionConflict) {
		t.Errorf("stale note: err = %v", err)
	}
}

func ptrTo[T any](v T) *T { return &v }

func TestNotesTagsListsAndDelete(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	later := f.folder(f.sc.RootID, "Later")
	n := f.note("Note", "text", nil)

	if res, err := f.s.UpdateTags(f.ctx, f.sc, UpdateTagsInput{BookmarkID: n.ID}); err != nil || len(res.Attached) != 0 {
		t.Errorf("no tag changes: %+v, %v", res, err)
	}
	if _, err := f.s.UpdateTags(f.ctx, f.sc, UpdateTagsInput{BookmarkID: n.ID, Attach: []TagRef{{TagName: "x"}}}); !isInputError(err) {
		t.Errorf("tag a note: err = %v", err)
	}
	if lists, err := f.s.ListsOfBookmark(f.ctx, f.sc, n.ID); err != nil || len(lists) != 0 {
		t.Errorf("lists of a note: %v, %v", lists, err)
	}
	if err := f.s.AddToList(f.ctx, f.sc, later, n.ID); !isInputError(err) {
		t.Errorf("add to list: err = %v", err)
	}
	if err := f.s.RemoveFromList(f.ctx, f.sc, later, n.ID); !isInputError(err) {
		t.Errorf("remove from list: err = %v", err)
	}

	hidden := f.note("Hidden", "x", ptrTo(f.category("Other")))
	f.withNotes(f.category("Inbox"))
	if err := f.s.DeleteBookmark(f.ctx, f.sc, hidden.ID); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("delete outside the scope: err = %v", err)
	}
	f.withNotes("")
	if err := f.s.DeleteBookmark(f.ctx, f.sc, n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.notes.Note(f.ctx, f.user, n.ID); !errors.Is(err, notes.ErrNotFound) {
		t.Errorf("deleted note still there: %v", err)
	}
}

func TestStatsWithNotes(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	for i := range 5 {
		f.link(f.sc.RootID, "L", "https://example.org/"+string(rune('a'+i)))
	}
	f.note("A", "a", nil)
	f.note("B", "b", nil)
	st, err := f.s.UserStats(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	if st.NumBookmarks != 7 || st.BookmarksByType.Link != 5 || st.BookmarksByType.Text != 2 ||
		len(st.TopDomains) != 1 || st.TopDomains[0].Count != 5 || st.BookmarkingActivity.ThisYear != 7 {
		t.Errorf("stats = %+v", st)
	}
}

// flag sets a note's pinned and archived flags.
func (f *fixture) flag(n notes.Note, pinned, archived bool) notes.Note {
	f.t.Helper()
	n, err := f.notes.UpdateNote(f.ctx, f.user, n.ID, n.Version, notes.NotePatch{Pinned: &pinned, Archived: &archived})
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

func TestNoteFlagsInBookmarks(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	f.link(f.sc.RootID, "Link", "https://example.org/a")
	f.flag(f.note("Old plan", "old", nil), false, true)
	f.flag(f.note("Groceries", "milk", nil), true, false)
	both := f.flag(f.note("Both", "x", nil), true, true)

	b, err := f.s.GetBookmark(f.ctx, f.sc, both.ID)
	if err != nil || !b.Archived || !b.Favourited {
		t.Errorf("pinned and archived note = %+v, %v; want archived and favourited", b, err)
	}
	yes, no := true, false
	for name, tc := range map[string]struct {
		in   BookmarksInput
		want []string
	}{
		"all":            {BookmarksInput{}, []string{"Both", "Groceries", "Old plan", "Link"}},
		"archived":       {BookmarksInput{Archived: &yes}, []string{"Both", "Old plan"}},
		"not archived":   {BookmarksInput{Archived: &no}, []string{"Groceries", "Link"}},
		"favourites":     {BookmarksInput{Favourited: &yes}, []string{"Both", "Groceries"}},
		"no favourites":  {BookmarksInput{Favourited: &no}, []string{"Old plan", "Link"}},
		"both filters":   {BookmarksInput{Archived: &no, Favourited: &yes}, []string{"Groceries"}},
		"archived pages": {BookmarksInput{Archived: &yes, Limit: 1}, []string{"Both"}},
	} {
		page, err := f.s.Bookmarks(f.ctx, f.sc, tc.in)
		if err != nil || !slices.Equal(titlesOf(page.Bookmarks), tc.want) {
			t.Errorf("%s = %v, %v; want %v", name, titlesOf(page.Bookmarks), err, tc.want)
		}
		if name == "archived pages" && page.NextCursor == nil {
			t.Errorf("archived pages: no cursor to the second archived note")
		}
	}

	st, err := f.s.UserStats(f.ctx, f.sc)
	if err != nil || st.NumFavorites != 2 || st.NumArchived != 2 {
		t.Errorf("stats favourites %d, archived %d, err %v; want 2 and 2", st.NumFavorites, st.NumArchived, err)
	}
}

func TestNoteFlagsOnCreateAndUpdate(t *testing.T) {
	f := newFixture(t)
	f.withNotes("")
	yes, no := true, false
	created, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "text", Text: ptrTo("milk"), Favourited: &yes})
	if err != nil || !created.Favourited || created.Archived {
		t.Fatalf("favourite note = %+v, %v", created.Bookmark, err)
	}
	if n, _ := f.notes.Note(f.ctx, f.user, created.ID); !n.Pinned || n.Archived {
		t.Errorf("created note pinned %v, archived %v; want pinned only", n.Pinned, n.Archived)
	}
	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "link", URL: "https://example.org/x", Archived: &yes}); !isInputError(err) {
		t.Errorf("archived link: %v, want an input error", err)
	}

	n := f.note("Old plan", "old", nil)
	updated, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: n.ID, Archived: &yes})
	if err != nil || !updated.Archived {
		t.Fatalf("archive note = %+v, %v", updated, err)
	}
	if got, _ := f.notes.Note(f.ctx, f.user, n.ID); !got.Archived || got.Version != n.Version+1 {
		t.Errorf("archived note = %+v, want archived at version %d", got, n.Version+1)
	}
	same, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: n.ID, Archived: &yes})
	if got, _ := f.notes.Note(f.ctx, f.user, n.ID); err != nil || !same.Archived || got.Version != n.Version+1 {
		t.Errorf("archiving again: version %d, err %v; want no change", got.Version, err)
	}
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: created.ID, Favourited: &no}); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.notes.Note(f.ctx, f.user, created.ID); got.Pinned {
		t.Error("unfavourited note is still pinned")
	}

	link := f.link(f.sc.RootID, "Link", "https://example.org/a")
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: link, Favourited: &yes}); !isInputError(err) {
		t.Errorf("favourite link: %v, want an input error", err)
	}
	if _, err := f.s.UpdateBookmark(f.ctx, f.sc, UpdateBookmarkInput{BookmarkID: link, Archived: &no}); err != nil {
		t.Errorf("archived false on a link: %v, want accepted", err)
	}
}
