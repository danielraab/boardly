package linkwarden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/users"
)

type fakeFavicons struct{ queued []string }

func (f *fakeFavicons) LookupImages(_ context.Context, origins []string) (map[string]images.FaviconImage, error) {
	out := map[string]images.FaviconImage{}
	for _, o := range origins {
		switch o {
		case "https://go.dev":
			out[o] = images.FaviconImage{ID: "icon-go", ContentType: "image/png"}
		case "https://ico.example":
			out[o] = images.FaviconImage{ID: "icon-ico", ContentType: "image/x-icon"}
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
	user     users.User
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
	if ann, err = users.NewStore(database).Get(ctx, ann.ID); err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, ctx: ctx, user: ann, now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC), icons: &fakeFavicons{}}
	f.fav = favourites.NewStore(database)
	f.fav.Now = func() time.Time { f.now = f.now.Add(time.Minute); return f.now }
	f.s = &Service{Favourites: f.fav, Favicons: f.icons}
	if f.topLevel, err = f.fav.EnsureRoot(ctx, database, ann.ID); err != nil {
		t.Fatal(err)
	}
	f.sc = favourites.Scope{UserID: ann.ID, RootID: f.folder(f.topLevel, "Reading")}
	return f
}

func (f *fixture) folder(parentID, name string) string {
	f.t.Helper()
	n, err := f.fav.Create(f.ctx, f.user.ID, favourites.NewNode{ParentID: parentID, Kind: favourites.KindFolder, Title: &name})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

func (f *fixture) link(parentID, title, url string, tags ...string) string {
	f.t.Helper()
	n, err := f.fav.Create(f.ctx, f.user.ID, favourites.NewNode{ParentID: parentID, Kind: favourites.KindLink, Title: &title, URL: &url, Tags: tags})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

// nodes returns all nodes of Ann's tree by ID.
func (f *fixture) nodes() map[string]favourites.ScopedNode {
	f.t.Helper()
	nodes, err := f.fav.SubtreeNodes(f.ctx, favourites.Scope{UserID: f.user.ID, RootID: f.topLevel})
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[string]favourites.ScopedNode{}
	for _, n := range nodes {
		out[n.ID] = n
	}
	return out
}

// num returns the integer ID of a node.
func (f *fixture) num(id string) int64 {
	f.t.Helper()
	n, ok := f.nodes()[id]
	if !ok {
		f.t.Fatalf("no node %s", id)
	}
	return n.Num
}

// tagNum returns the integer ID of a tag of a link.
func (f *fixture) tagNum(linkID, name string) int64 {
	f.t.Helper()
	for _, tag := range f.nodes()[linkID].Tags {
		if tag.Name == name {
			return tag.Num
		}
	}
	f.t.Fatalf("link %s has no tag %s", linkID, name)
	return 0
}

// children returns the titles of a folder's children in order.
func (f *fixture) children(id string) []string {
	f.t.Helper()
	var kids []favourites.ScopedNode
	for _, n := range f.nodes() {
		if n.ParentID == id && n.ID != f.topLevel {
			kids = append(kids, n)
		}
	}
	out := make([]string, len(kids))
	for _, n := range kids {
		out[n.Position] = *n.Title
	}
	return out
}

func names(links []Link) []string {
	out := []string{}
	for _, l := range links {
		out = append(out, l.Name)
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

func isInput(err error) bool {
	var in *InputError
	return errors.As(err, &in)
}

func TestShapes(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	id := f.link(later, "Go", "https://go.dev/tour", "dev")

	link, err := f.s.Link(f.ctx, f.sc, f.num(id))
	if err != nil {
		t.Fatal(err)
	}
	const tag = `{"id":1,"name":"dev","ownerId":1,"archiveAsScreenshot":null,"archiveAsMonolith":null,"archiveAsPDF":null,` +
		`"archiveAsReadable":null,"archiveAsWaybackMachine":null,"aiTag":null,"aiGenerated":false,` +
		`"createdAt":"2026-03-04T12:05:00.000Z","updatedAt":"2026-03-04T12:05:00.000Z"`
	const short = `{"id":3,"name":"Later","description":"","icon":null,"iconWeight":null,"color":"#0ea5e9","parentId":2,` +
		`"isPublic":false,"ownerId":1,"createdById":1,"createdAt":"2026-03-04T12:03:00.000Z","updatedAt":"2026-03-04T12:03:00.000Z"}`
	want := `{"id":4,"name":"Go","type":"url","description":"","createdById":1,"collectionId":3,"icon":null,"iconWeight":null,` +
		`"color":null,"url":"https://go.dev/tour","textContent":null,"preview":"archives/preview/3/4.jpeg","image":"unavailable",` +
		`"pdf":"unavailable","readable":"unavailable","monolith":"unavailable","clientSide":false,"aiTagged":false,` +
		`"metaDescription":null,"indexVersion":null,"lastPreserved":"2026-03-04T12:04:00.000Z","importDate":null,` +
		`"createdAt":"2026-03-04T12:04:00.000Z","updatedAt":"2026-03-04T12:04:00.000Z","tags":[` + tag + `}],` +
		`"collection":` + short + `,"pinnedBy":[]}`
	if got := jsonOf(t, link); got != want {
		t.Errorf("link =\n%s\nwant\n%s", got, want)
	}

	collections, err := f.s.Collections(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	wantCollection := short[:len(short)-1] + `,"parent":{"id":2,"name":"Reading"},"members":[],"_count":{"links":1}}`
	if len(collections) != 2 || jsonOf(t, collections[1]) != wantCollection {
		t.Errorf("collections = %s, want Reading and\n%s", jsonOf(t, collections), wantCollection)
	}
	if c := collections[0]; c.Name != "Reading" || c.ParentID != nil || c.Parent != nil || c.Count.Links != 0 {
		t.Errorf("root collection = %s", jsonOf(t, c))
	}

	got, err := f.s.Tag(f.ctx, f.sc, 1)
	if err != nil || jsonOf(t, got) != tag+`,"_count":{"links":1}}` {
		t.Errorf("tag = %s, err %v", jsonOf(t, got), err)
	}
}

func TestPreviewValues(t *testing.T) {
	f := newFixture(t)
	ico := f.link(f.sc.RootID, "Ico", "https://ico.example/a")
	none := f.link(f.sc.RootID, "None", "https://none.example/a")
	for id, want := range map[string]string{ico: "unavailable", none: "unavailable"} {
		link, err := f.s.Link(f.ctx, f.sc, f.num(id))
		if err != nil || link.Preview != want {
			t.Errorf("preview = %q, err %v; want %q", link.Preview, err, want)
		}
	}
}

func TestTopLevelIsUnorganized(t *testing.T) {
	f := newFixture(t)
	sc := favourites.Scope{UserID: f.user.ID, RootID: f.topLevel}
	collections, err := f.s.Collections(f.ctx, sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(collections) != 2 || collections[0].Name != "Unorganized" || collections[1].Name != "Reading" ||
		collections[1].ParentID == nil || *collections[1].ParentID != collections[0].ID {
		t.Errorf("collections = %s", jsonOf(t, collections))
	}
}

func TestOutsideScope(t *testing.T) {
	f := newFixture(t)
	work := f.folder(f.topLevel, "Work")
	docs := f.link(work, "Docs", "https://example.org/docs")
	if _, err := f.s.Link(f.ctx, f.sc, f.num(docs)); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("link outside: err = %v", err)
	}
	if _, err := f.s.Link(f.ctx, f.sc, f.num(work)); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("folder as link: err = %v", err)
	}
	if _, err := f.s.DeleteCollection(f.ctx, f.sc, f.num(work)); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("delete collection outside: err = %v", err)
	}
	if page, err := f.s.Search(f.ctx, f.sc, SearchInput{CollectionID: f.num(work)}); err != nil || len(page.Links) != 0 {
		t.Errorf("search in collection outside = %v, %v", names(page.Links), err)
	}
}

func TestSearchPages(t *testing.T) {
	f := newFixture(t)
	for i := range 60 {
		f.link(f.sc.RootID, fmt.Sprintf("L%02d", i), fmt.Sprintf("https://example.org/%d", i))
	}
	first, err := f.s.Search(f.ctx, f.sc, SearchInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Links) != 50 || first.Links[0].Name != "L59" || first.NextCursor == nil || *first.NextCursor != first.Links[49].ID {
		t.Fatalf("first page: %d links, first %s, cursor %v", len(first.Links), first.Links[0].Name, first.NextCursor)
	}
	second, err := f.s.Search(f.ctx, f.sc, SearchInput{Cursor: *first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Links) != 10 || second.Links[0].Name != "L09" || second.NextCursor != nil {
		t.Fatalf("second page: %v, cursor %v", names(second.Links), second.NextCursor)
	}

	oldest, _ := f.s.Search(f.ctx, f.sc, SearchInput{Sort: SortOldest})
	za, _ := f.s.Search(f.ctx, f.sc, SearchInput{Sort: SortNameZA})
	if oldest.Links[0].Name != "L00" || za.Links[0].Name != "L59" || za.Links[1].Name != "L58" {
		t.Errorf("oldest first %s, Z–A %v", oldest.Links[0].Name, names(za.Links[:2]))
	}
	if page, _ := f.s.Search(f.ctx, f.sc, SearchInput{PinnedOnly: true}); len(page.Links) != 0 || page.NextCursor != nil {
		t.Errorf("pinned = %v", names(page.Links))
	}
}

func TestSearchFilters(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	videos := f.folder(later, "Videos")
	goTour := f.link(later, "Go tour", "https://go.dev/tour/", "dev")
	f.link(videos, "Talk", "https://example.org/talk", "dev")
	f.link(f.sc.RootID, "Rust book", "https://doc.rust-lang.org/book", "Reading list")

	for name, tc := range map[string]struct {
		in   SearchInput
		want []string
	}{
		"collection":         {SearchInput{CollectionID: f.num(later)}, []string{"Go tour"}},
		"tag":                {SearchInput{TagID: f.tagNum(goTour, "dev")}, []string{"Talk", "Go tour"}},
		"words":              {SearchInput{Query: "go"}, []string{"Go tour"}},
		"every term":         {SearchInput{Query: "go rust"}, []string{}},
		"url with fragment":  {SearchInput{Query: "url:https://go.dev/tour#intro"}, []string{"Go tour"}},
		"url is exact":       {SearchInput{Query: "url:https://go.dev"}, []string{}},
		"name":               {SearchInput{Query: "name:tour"}, []string{"Go tour"}},
		"tag qualifier":      {SearchInput{Query: `tag:"reading list"`}, []string{"Rust book"}},
		"collection name":    {SearchInput{Query: "collection:vid"}, []string{"Talk"}},
		"unknown qualifier":  {SearchInput{Query: "pinned:true"}, []string{}},
		"plain text in tags": {SearchInput{Query: "list"}, []string{"Rust book"}},
		"empty":              {SearchInput{Query: "  "}, []string{"Rust book", "Talk", "Go tour"}},
	} {
		page, err := f.s.Search(f.ctx, f.sc, tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := names(page.Links); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}
