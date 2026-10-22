package karakeep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"famodash/internal/dbtest"
	"famodash/internal/favourites"
	"famodash/internal/users"
)

type fakeFavicons struct{ queued []string }

func (f *fakeFavicons) Lookup(_ context.Context, origins []string) (map[string]string, error) {
	out := map[string]string{}
	for _, o := range origins {
		if o == "https://go.dev" {
			out[o] = "icon-go"
		}
	}
	return out, nil
}

func (f *fakeFavicons) Queue(rawURL string) { f.queued = append(f.queued, rawURL) }

type fixture struct {
	t        *testing.T
	ctx      context.Context
	s        *Service
	fav      *favourites.Store
	icons    *fakeFavicons
	user     string
	now      time.Time
	sc       favourites.Scope
	topLevel string // the user's root folder
}

// newFixture returns a service whose scope is the folder "Reading" in Ann's
// tree. Every node is created one minute after the previous one.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	ctx := context.Background()
	ann, err := users.NewStore(database).EnsureByEmail(ctx, database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, ctx: ctx, user: ann.ID, now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC), icons: &fakeFavicons{}}
	f.fav = favourites.NewStore(database)
	f.fav.Now = func() time.Time { f.now = f.now.Add(time.Minute); return f.now }
	f.s = &Service{Favourites: f.fav, Favicons: f.icons, Origin: "https://famodash.example", Now: func() time.Time { return f.now }}
	if f.topLevel, err = f.fav.EnsureRoot(ctx, database, ann.ID); err != nil {
		t.Fatal(err)
	}
	f.sc = favourites.Scope{UserID: ann.ID, RootID: f.folder(f.topLevel, "Reading")}
	return f
}

func (f *fixture) folder(parentID, name string) string {
	f.t.Helper()
	n, err := f.fav.Create(f.ctx, f.user, favourites.NewNode{ParentID: parentID, Kind: favourites.KindFolder, Title: &name})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

func (f *fixture) link(parentID, title, url string, tags ...string) string {
	f.t.Helper()
	n, err := f.fav.Create(f.ctx, f.user, favourites.NewNode{ParentID: parentID, Kind: favourites.KindLink, Title: &title, URL: &url, Tags: tags})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

// children returns the titles of a folder's children in order.
func (f *fixture) children(id string) []string {
	f.t.Helper()
	nodes, err := f.fav.SubtreeNodes(f.ctx, favourites.Scope{UserID: f.user, RootID: f.topLevel})
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, n := range nodes {
		if n.ParentID == id && n.ID != f.topLevel {
			out = append(out, *n.Title)
		}
	}
	return out
}

func titles(bs []Bookmark) []string {
	out := []string{}
	for _, b := range bs {
		out = append(out, *b.Title)
	}
	return out
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBookmarkShape(t *testing.T) {
	f := newFixture(t)
	id := f.link(f.sc.RootID, "Go", "https://go.dev/tour", "dev")
	b, err := f.s.GetBookmark(f.ctx, f.sc, id)
	if err != nil {
		t.Fatal(err)
	}
	tagID := b.Tags[0].ID
	want := fmt.Sprintf(`{"id":%q,"createdAt":"2026-03-04T12:03:00.000Z","modifiedAt":"2026-03-04T12:03:00.000Z","title":"Go",`+
		`"archived":false,"favourited":false,"taggingStatus":null,"summarizationStatus":null,"embeddingStatus":null,`+
		`"note":null,"summary":null,"source":null,"userId":%q,"tags":[{"id":%q,"name":"dev","attachedBy":"human"}],`+
		`"content":{"type":"link","url":"https://go.dev/tour","title":"Go","favicon":"https://famodash.example/api/images/icon-go",`+
		`"crawlStatus":"success","crawledAt":"2026-03-04T12:03:00.000Z"},"assets":[]}`, id, f.user, tagID)
	if got := jsonOf(t, b); got != want {
		t.Errorf("bookmark =\n%s\nwant\n%s", got, want)
	}

	outside := f.link(f.topLevel, "Docs", "https://example.org")
	if _, err := f.s.GetBookmark(f.ctx, f.sc, outside); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("outside link: %v", err)
	}
	if _, err := f.s.GetBookmark(f.ctx, f.sc, f.sc.RootID); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("folder as bookmark: %v", err)
	}
}

func TestBookmarksPaging(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	for i := range 25 {
		parent := f.sc.RootID
		if i%5 == 0 {
			parent = later
		}
		f.link(parent, fmt.Sprintf("L%02d", i), fmt.Sprintf("https://example.org/%d", i))
	}
	f.link(f.topLevel, "Outside", "https://example.org/out")

	first, err := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Bookmarks) != 20 || *first.Bookmarks[0].Title != "L24" || first.NextCursor == nil {
		t.Fatalf("first page = %v, cursor %v", titles(first.Bookmarks), first.NextCursor)
	}
	in := BookmarksInput{}
	if err := json.Unmarshal([]byte(`{"cursor":`+jsonOf(t, first.NextCursor)+`}`), &in); err != nil {
		t.Fatal(err)
	}
	second, err := f.s.Bookmarks(f.ctx, f.sc, in)
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(second.Bookmarks); !reflect.DeepEqual(got, []string{"L04", "L03", "L02", "L01", "L00"}) || second.NextCursor != nil {
		t.Errorf("second page = %v, cursor %v", got, second.NextCursor)
	}

	asc, _ := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{SortOrder: "asc", Limit: 2})
	if got := titles(asc.Bookmarks); !reflect.DeepEqual(got, []string{"L00", "L01"}) {
		t.Errorf("asc = %v", got)
	}
	list, _ := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{ListID: later})
	if got := titles(list.Bookmarks); !reflect.DeepEqual(got, []string{"L20", "L15", "L10", "L05", "L00"}) {
		t.Errorf("list = %v", got)
	}
	yes := true
	if archived, _ := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{Archived: &yes}); len(archived.Bookmarks) != 0 || archived.NextCursor != nil {
		t.Errorf("archived = %v", titles(archived.Bookmarks))
	}
	if _, err := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{ListID: f.sc.RootID}); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("root as list: %v", err)
	}
	var inputErr *InputError
	if _, err := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{Limit: 101}); !errors.As(err, &inputErr) {
		t.Errorf("limit 101: %v", err)
	}
}

func TestSearchAndTagFilter(t *testing.T) {
	f := newFixture(t)
	f.link(f.sc.RootID, "Go tour", "https://go.dev/tour", "dev")
	f.link(f.sc.RootID, "Rust book", "https://doc.rust-lang.org/book", "dev", "reading")
	f.link(f.sc.RootID, "News", "https://news.example", "daily")
	f.link(f.topLevel, "Go outside", "https://go.dev/outside", "dev")

	for text, want := range map[string][]string{
		"go":          {"Go tour"},
		"#dev":        {"Rust book", "Go tour"},
		"#DEV book":   {"Rust book"},
		"rust-lang":   {"Rust book"},
		"read":        {"Rust book"},
		"":            {"News", "Rust book", "Go tour"},
		"is:fav news": {},
	} {
		page, err := f.s.Search(f.ctx, f.sc, SearchInput{Text: text})
		if err != nil {
			t.Fatal(err)
		}
		if got := titles(page.Bookmarks); !reflect.DeepEqual(got, want) {
			t.Errorf("search %q = %v, want %v", text, got, want)
		}
	}
	page, _ := f.s.Search(f.ctx, f.sc, SearchInput{Limit: 2})
	if page.NextCursor == nil || page.NextCursor.Offset != 2 {
		t.Fatalf("cursor = %+v", page.NextCursor)
	}
	in := SearchInput{}
	_ = json.Unmarshal([]byte(`{"limit":2,"cursor":{"ver":1,"offset":2}}`), &in)
	if next, _ := f.s.Search(f.ctx, f.sc, in); !reflect.DeepEqual(titles(next.Bookmarks), []string{"Go tour"}) || next.NextCursor != nil {
		t.Errorf("second search page = %v", titles(next.Bookmarks))
	}

	tags, _ := f.s.Tags(f.ctx, f.sc, TagsInput{NameContains: "dev"})
	byTag, err := f.s.Bookmarks(f.ctx, f.sc, BookmarksInput{TagID: tags.Tags[0].ID})
	if err != nil || len(byTag.Bookmarks) != 2 {
		t.Errorf("by tag = %v, %v", titles(byTag.Bookmarks), err)
	}
}

func TestCheckURL(t *testing.T) {
	f := newFixture(t)
	id := f.link(f.sc.RootID, "Docs", "https://example.org/docs/")
	f.link(f.topLevel, "Elsewhere", "https://example.org/elsewhere")
	for raw, want := range map[string]*string{
		"https://example.org/docs#intro": &id,
		"https://EXAMPLE.org/docs/":      &id,
		"https://example.org/docs/x":     nil,
		"https://example.org/elsewhere":  nil,
	} {
		got, err := f.s.CheckURL(f.ctx, f.sc, raw)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("CheckURL(%s) = %v, %v", raw, got, err)
		}
	}
}

func TestCreateBookmark(t *testing.T) {
	f := newFixture(t)
	f.link(f.sc.RootID, "A", "https://example.org/a")
	title := "Example"
	created, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "link", URL: "https://example.org", Title: &title})
	if err != nil || created.AlreadyExists || *created.Title != "Example" {
		t.Fatalf("create = %+v, %v", created, err)
	}
	if got := f.children(f.sc.RootID); !reflect.DeepEqual(got, []string{"Example", "A"}) {
		t.Errorf("root = %v", got)
	}
	if !reflect.DeepEqual(f.icons.queued, []string{"https://example.org"}) {
		t.Errorf("favicons queued = %v", f.icons.queued)
	}

	again, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "link", URL: "https://example.org/#top"})
	if err != nil || !again.AlreadyExists || again.ID != created.ID {
		t.Errorf("duplicate = %+v, %v", again, err)
	}
	if got := jsonOf(t, again); !strings.HasSuffix(got, `,"assets":[],"alreadyExists":true}`) {
		t.Errorf("JSON = %s", got)
	}

	note, yes := "remember", true
	for name, in := range map[string]CreateBookmarkInput{
		"text":     {Type: "text"},
		"note":     {Type: "link", URL: "https://example.org/n", Note: &note},
		"archived": {Type: "link", URL: "https://example.org/n", Archived: &yes},
	} {
		var inputErr *InputError
		if _, err := f.s.CreateBookmark(f.ctx, f.sc, in); !errors.As(err, &inputErr) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := f.s.CreateBookmark(f.ctx, f.sc, CreateBookmarkInput{Type: "link", URL: "javascript:alert(1)"}); !errors.Is(err, favourites.ErrInvalidURL) {
		t.Errorf("invalid URL: %v", err)
	}
	if got := f.children(f.sc.RootID); len(got) != 2 {
		t.Errorf("rejected creates added links: %v", got)
	}
}

func TestUpdateAndDeleteBookmark(t *testing.T) {
	f := newFixture(t)
	id := f.link(f.sc.RootID, "Old", "https://example.org/old")
	in := UpdateBookmarkInput{}
	_ = json.Unmarshal([]byte(`{"bookmarkId":"`+id+`","title":"New","url":"https://example.org/new","archived":false,"note":""}`), &in)
	b, err := f.s.UpdateBookmark(f.ctx, f.sc, in)
	if err != nil || *b.Title != "New" || b.Content.URL != "https://example.org/new" {
		t.Fatalf("update = %+v, %v", b, err)
	}

	_ = json.Unmarshal([]byte(`{"bookmarkId":"`+id+`","title":null}`), &in)
	in.URL = nil
	if b, err := f.s.UpdateBookmark(f.ctx, f.sc, in); err != nil || b.Title != nil {
		t.Errorf("clear title = %+v, %v", b.Title, err)
	}

	for name, raw := range map[string]string{
		"archived":      `{"archived":true}`,
		"note":          `{"note":"x"}`,
		"datePublished": `{"datePublished":"2026-01-01T00:00:00.000Z"}`,
	} {
		in := UpdateBookmarkInput{}
		_ = json.Unmarshal([]byte(raw), &in)
		in.BookmarkID = id
		var inputErr *InputError
		if _, err := f.s.UpdateBookmark(f.ctx, f.sc, in); !errors.As(err, &inputErr) {
			t.Errorf("%s: %v", name, err)
		}
	}

	if err := f.s.DeleteBookmark(f.ctx, f.sc, id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.GetBookmark(f.ctx, f.sc, id); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("deleted bookmark: %v", err)
	}
	if err := f.s.DeleteBookmark(f.ctx, f.sc, f.sc.RootID); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("delete root: %v", err)
	}
}

func TestUpdateTags(t *testing.T) {
	f := newFixture(t)
	id := f.link(f.sc.RootID, "Go", "https://go.dev", "dev")
	f.link(f.topLevel, "Hidden", "https://example.org", "secret")
	b, _ := f.s.GetBookmark(f.ctx, f.sc, id)
	dev := b.Tags[0].ID

	res, err := f.s.UpdateTags(f.ctx, f.sc, UpdateTagsInput{
		BookmarkID: id,
		Attach:     []TagRef{{TagName: "later"}, {TagName: "DEV"}},
		Detach:     []TagRef{{TagID: dev}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ = f.s.GetBookmark(f.ctx, f.sc, id)
	if len(b.Tags) != 2 || len(res.Attached) != 2 || !reflect.DeepEqual(res.Detached, []string{dev}) {
		t.Errorf("tags = %+v, result %+v", b.Tags, res)
	}

	tags, _ := f.s.Tags(f.ctx, f.sc, TagsInput{})
	hidden := ""
	all, _ := f.fav.SubtreeNodes(f.ctx, favourites.Scope{UserID: f.user, RootID: f.topLevel})
	for _, n := range all {
		if len(n.Tags) > 0 && n.Tags[0].Name == "secret" {
			hidden = n.Tags[0].ID
		}
	}
	if len(tags.Tags) != 2 || hidden == "" {
		t.Fatalf("tags = %+v", tags.Tags)
	}
	if _, err := f.s.UpdateTags(f.ctx, f.sc, UpdateTagsInput{BookmarkID: id, Attach: []TagRef{{TagID: hidden}}}); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("attach hidden tag: %v", err)
	}
}

func TestLists(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	videos := f.folder(later, "Videos")
	a := f.link(later, "A", "https://example.org/a")
	f.link(later, "B", "https://example.org/b")
	f.link(videos, "V", "https://example.org/v")
	root := f.link(f.sc.RootID, "R", "https://example.org/r")
	f.folder(f.topLevel, "Work")

	lists, err := f.s.Lists(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`[{"id":%q,"name":"Later","description":null,"icon":"📁","parentId":null,"type":"manual","query":null,"public":false,"hasCollaborators":false,"userRole":"owner"},`+
		`{"id":%q,"name":"Videos","description":null,"icon":"📁","parentId":%q,"type":"manual","query":null,"public":false,"hasCollaborators":false,"userRole":"owner"}]`, later, videos, later)
	if got := jsonOf(t, lists); got != want {
		t.Errorf("lists =\n%s\nwant\n%s", got, want)
	}
	stats, _ := f.s.ListStats(f.ctx, f.sc)
	if got := jsonOf(t, stats); got != fmt.Sprintf(`[[%q,2],[%q,1]]`, later, videos) {
		t.Errorf("stats = %s", got)
	}
	if of, _ := f.s.ListsOfBookmark(f.ctx, f.sc, a); len(of) != 1 || of[0].ID != later {
		t.Errorf("lists of A = %+v", of)
	}
	if of, _ := f.s.ListsOfBookmark(f.ctx, f.sc, root); len(of) != 0 {
		t.Errorf("lists of R = %+v", of)
	}

	// Add moves the link first into the list; remove moves it first into the root.
	if err := f.s.AddToList(f.ctx, f.sc, videos, a); err != nil {
		t.Fatal(err)
	}
	if got := f.children(videos); !reflect.DeepEqual(got, []string{"A", "V"}) {
		t.Errorf("videos = %v", got)
	}
	var inputErr *InputError
	if err := f.s.RemoveFromList(f.ctx, f.sc, later, a); !errors.As(err, &inputErr) {
		t.Errorf("remove from wrong list: %v", err)
	}
	if err := f.s.RemoveFromList(f.ctx, f.sc, videos, a); err != nil {
		t.Fatal(err)
	}
	if got := f.children(f.sc.RootID); !reflect.DeepEqual(got, []string{"A", "Later", "R"}) {
		t.Errorf("root = %v", got)
	}

	created, err := f.s.CreateList(f.ctx, f.sc, CreateListInput{Name: "Recipes", Icon: "🍝"})
	if err != nil || created.ParentID != nil {
		t.Fatalf("create = %+v, %v", created, err)
	}
	sub, err := f.s.CreateList(f.ctx, f.sc, CreateListInput{Name: "Soups", ParentID: &created.ID})
	if err != nil || sub.ParentID == nil || *sub.ParentID != created.ID {
		t.Fatalf("create sub = %+v, %v", sub, err)
	}
	q := "#x"
	if _, err := f.s.CreateList(f.ctx, f.sc, CreateListInput{Name: "Smart", Type: "smart", Query: &q}); !errors.As(err, &inputErr) {
		t.Errorf("smart list: %v", err)
	}

	edit := EditListInput{}
	_ = json.Unmarshal([]byte(`{"listId":"`+created.ID+`","name":"Food","icon":"🍕","parentId":"`+later+`"}`), &edit)
	edited, err := f.s.EditList(f.ctx, f.sc, edit)
	if err != nil || edited.Name != "Food" || *edited.ParentID != later {
		t.Fatalf("edit = %+v, %v", edited, err)
	}
	_ = json.Unmarshal([]byte(`{"listId":"`+created.ID+`","parentId":"`+sub.ID+`"}`), &edit)
	edit.Name = nil
	if _, err := f.s.EditList(f.ctx, f.sc, edit); !errors.Is(err, favourites.ErrInvalidMove) {
		t.Errorf("move into own sublist: %v", err)
	}
	edit = EditListInput{}
	_ = json.Unmarshal([]byte(`{"listId":"`+created.ID+`","parentId":null}`), &edit)
	if moved, err := f.s.EditList(f.ctx, f.sc, edit); err != nil || moved.ParentID != nil {
		t.Errorf("move to root = %+v, %v", moved, err)
	}

	// Deleting "Later" moves its links and "Videos" to the root, in its place.
	if err := f.s.DeleteList(f.ctx, f.sc, later); err != nil {
		t.Fatal(err)
	}
	if got := f.children(f.sc.RootID); !reflect.DeepEqual(got, []string{"A", "Videos", "B", "R", "Food"}) {
		t.Errorf("root after delete = %v", got)
	}
	if err := f.s.DeleteList(f.ctx, f.sc, f.sc.RootID); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("delete root list: %v", err)
	}
}

func TestTags(t *testing.T) {
	f := newFixture(t)
	f.link(f.sc.RootID, "A", "https://example.org/a", "dev", "go")
	f.link(f.sc.RootID, "B", "https://example.org/b", "dev")
	f.link(f.sc.RootID, "C", "https://example.org/c", "Alpha")
	f.link(f.topLevel, "Out", "https://example.org/out", "go", "work")

	names := func(p TagPage) []string {
		var out []string
		for _, tag := range p.Tags {
			out = append(out, fmt.Sprintf("%s:%d", tag.Name, tag.NumBookmarks))
		}
		return out
	}
	usage, _ := f.s.Tags(f.ctx, f.sc, TagsInput{})
	if got := names(usage); !reflect.DeepEqual(got, []string{"dev:2", "Alpha:1", "go:1"}) {
		t.Errorf("by usage = %v", got)
	}
	byName, _ := f.s.Tags(f.ctx, f.sc, TagsInput{SortBy: "name", Limit: 2})
	if got := names(byName); !reflect.DeepEqual(got, []string{"Alpha:1", "dev:2"}) || byName.NextCursor == nil || byName.NextCursor.Page != 1 {
		t.Errorf("by name = %v, %v", got, byName.NextCursor)
	}
	in := TagsInput{}
	_ = json.Unmarshal([]byte(`{"sortBy":"name","limit":2,"cursor":{"page":1}}`), &in)
	if next, _ := f.s.Tags(f.ctx, f.sc, in); !reflect.DeepEqual(names(next), []string{"go:1"}) || next.NextCursor != nil {
		t.Errorf("page 2 = %v", names(next))
	}
	if ai, _ := f.s.Tags(f.ctx, f.sc, TagsInput{AttachedBy: "ai"}); len(ai.Tags) != 0 {
		t.Errorf("ai = %v", names(ai))
	}
	if got := jsonOf(t, usage.Tags[0]); got != fmt.Sprintf(`{"id":%q,"name":"dev","numBookmarks":2,"numBookmarksByAttachedType":{"ai":0,"human":2}}`, usage.Tags[0].ID) {
		t.Errorf("tag JSON = %s", got)
	}

	id := func(name string) string {
		for _, tag := range usage.Tags {
			if tag.Name == name {
				return tag.ID
			}
		}
		t.Fatalf("no tag %s", name)
		return ""
	}
	newName := "development"
	if tag, err := f.s.RenameTag(f.ctx, f.sc, id("dev"), &newName); err != nil || tag.Name != "development" {
		t.Errorf("rename = %+v, %v", tag, err)
	}
	if _, err := f.s.RenameTag(f.ctx, f.sc, id("go"), &newName); !errors.Is(err, ErrTagShared) {
		t.Errorf("rename shared tag: %v", err)
	}
	taken := "alpha"
	if _, err := f.s.RenameTag(f.ctx, f.sc, id("dev"), &taken); !errors.Is(err, favourites.ErrTagExists) {
		t.Errorf("rename to taken name: %v", err)
	}

	// Deleting "go" removes it in the subtree only.
	if err := f.s.DeleteTag(f.ctx, f.sc, id("go")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Tag(f.ctx, f.sc, id("go")); !errors.Is(err, favourites.ErrNotFound) {
		t.Errorf("deleted tag still visible: %v", err)
	}
	if n, err := f.fav.TagLinkCount(f.ctx, f.user, id("go")); err != nil || n != 1 {
		t.Errorf("go outside = %d, %v", n, err)
	}
	if err := f.s.CreateTag(); !errors.Is(err, ErrTagCreate) {
		t.Errorf("create tag: %v", err)
	}
}

func TestSettingsAndStats(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	f.folder(later, "Videos")
	f.link(f.sc.RootID, "A", "https://example.org/a", "dev")
	f.link(later, "B", "https://EXAMPLE.org/b", "dev", "go")
	f.link(later, "C", "https://example.org/c")
	f.link(later, "D", "https://go.dev")
	f.link(f.sc.RootID, "E", "https://news.example")
	f.link(f.topLevel, "Out", "https://example.org/out")

	if got := jsonOf(t, f.s.UserSettings()); got != `{"bookmarkClickAction":"open_original_link","archiveDisplayBehaviour":"show","timezone":"UTC",`+
		`"backupsEnabled":false,"backupsFrequency":"weekly","backupsRetentionDays":30,"readerFontSize":null,"readerLineHeight":null,`+
		`"readerFontFamily":null,"autoTaggingEnabled":null,"autoSummarizationEnabled":null,"tagStyle":"lowercase-hyphens",`+
		`"curatedTagIds":null,"inferredTagLang":null}` {
		t.Errorf("settings = %s", got)
	}

	// The links were created on Wednesday 4 March 2026 around 12:05 UTC; "now"
	// is a week later, so they count for this month and year, not this week.
	f.now = time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)
	st, err := f.s.UserStats(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	a := st.BookmarkingActivity
	if st.NumBookmarks != 5 || st.BookmarksByType.Link != 5 || st.NumLists != 2 || st.NumTags != 2 ||
		!reflect.DeepEqual(st.TopDomains[0], DomainCount{"example.org", 3}) || len(st.TopDomains) != 3 ||
		!reflect.DeepEqual(st.TagUsage, []Count{{"dev", 2}, {"go", 1}}) ||
		a.ThisWeek != 0 || a.ThisMonth != 5 || a.ThisYear != 5 || a.ByHour[12].Count != 5 || a.ByDayOfWeek[3].Count != 5 ||
		len(a.ByHour) != 24 || len(a.ByDayOfWeek) != 7 || len(st.BookmarksBySource) != 1 {
		t.Errorf("stats = %s", jsonOf(t, st))
	}
	f.now = time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC).Add(-time.Second) // Sunday before
	if st, _ := f.s.UserStats(f.ctx, f.sc); st.BookmarkingActivity.ThisWeek != 5 {
		t.Errorf("this week = %d", st.BookmarkingActivity.ThisWeek)
	}
}
