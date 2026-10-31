package linkwarden

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"boardly/internal/favourites"
)

func ptr[T any](v T) *T { return &v }

func TestCreateLink(t *testing.T) {
	f := newFixture(t)
	f.link(f.sc.RootID, "Old", "https://example.org/old")
	later := f.folder(f.sc.RootID, "Later")

	// From the extension with its default collection: first in the root.
	link, err := f.s.CreateLink(f.ctx, f.sc, CreateLinkInput{
		URL: ptr("https://example.org"), Name: ptr("Example"), Description: ptr(""), Image: ptr("png"),
		Collection: &CollectionRef{Name: ptr("Unorganized")}, Tags: []TagRef{{Name: "dev"}, {Name: "Dev"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if link.Name != "Example" || link.CollectionID != f.num(f.sc.RootID) || len(link.Tags) != 1 || link.Tags[0].Name != "dev" {
		t.Errorf("link = %s", jsonOf(t, link))
	}
	if got := f.children(f.sc.RootID); !reflect.DeepEqual(got, []string{"Example", "Old", "Later"}) {
		t.Errorf("root = %v", got)
	}
	if got := f.queuedFavicons(); len(got) != 1 || got[0] != "https://example.org" {
		t.Errorf("queued favicons = %v", got)
	}

	// Into a collection by ID, and again with the same URL: saved twice.
	for range 2 {
		if _, err := f.s.CreateLink(f.ctx, f.sc, CreateLinkInput{URL: ptr("https://example.org/x"), Collection: &CollectionRef{ID: ptr(f.num(later))}}); err != nil {
			t.Fatal(err)
		}
	}
	nodes := f.nodes()
	var inLater int
	for _, n := range nodes {
		if n.ParentID == later {
			inLater++
			if n.Title != nil {
				t.Errorf("link without name has title %q", *n.Title)
			}
		}
	}
	if inLater != 2 {
		t.Errorf("links in Later = %d, want 2", inLater)
	}

	// By name: an existing folder in the root (ignoring case), or a new one at its end.
	if l, err := f.s.CreateLink(f.ctx, f.sc, CreateLinkInput{URL: ptr("https://example.org/y"), Collection: &CollectionRef{Name: ptr("later")}}); err != nil || l.CollectionID != f.num(later) {
		t.Errorf("by existing name: %v, err %v", l.CollectionID, err)
	}
	l, err := f.s.CreateLink(f.ctx, f.sc, CreateLinkInput{URL: ptr("https://example.org/z"), Collection: &CollectionRef{Name: ptr("Recipes")}})
	if err != nil || l.Collection.Name != "Recipes" {
		t.Fatalf("by new name: %+v, err %v", l.Collection, err)
	}
	if got := f.children(f.sc.RootID); got[len(got)-1] != "Recipes" {
		t.Errorf("root = %v, want Recipes last", got)
	}
}

func TestCreateLinkRejections(t *testing.T) {
	f := newFixture(t)
	work := f.folder(f.topLevel, "Work")
	before := len(f.nodes())
	for name, in := range map[string]CreateLinkInput{
		"pdf":              {Type: ptr("pdf"), URL: ptr("https://example.org/a.pdf")},
		"no url":           {Name: ptr("Note")},
		"invalid url":      {URL: ptr("javascript:alert(1)")},
		"long description": {URL: ptr("https://example.org"), Description: ptr(strings.Repeat("x", 2049))},
		"outside":          {URL: ptr("https://example.org"), Collection: &CollectionRef{ID: ptr(f.num(work))}},
		"bad tag":          {URL: ptr("https://example.org"), Tags: []TagRef{{Name: " "}}, Collection: &CollectionRef{Name: ptr("New")}},
		"unknown tag ID":   {URL: ptr("https://example.org"), Tags: []TagRef{{ID: ptr(int64(99))}}},
		"bad title, named": {URL: ptr("https://example.org"), Name: ptr(string(make([]byte, 1001))), Collection: &CollectionRef{Name: ptr("New")}},
	} {
		_, err := f.s.CreateLink(f.ctx, f.sc, in)
		if err == nil {
			t.Errorf("%s: no error", name)
		}
		if !isInput(err) && !errors.Is(err, favourites.ErrInvalidURL) && !errors.Is(err, favourites.ErrInvalidTag) &&
			!errors.Is(err, favourites.ErrInvalidTitle) && !errors.Is(err, ErrTagNotFound) &&
			!errors.Is(err, favourites.ErrInvalidDescription) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if after := len(f.nodes()); after != before {
		t.Errorf("nodes = %d, want %d: a rejected link created something", after, before)
	}
}

func TestUpdateLink(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	recipes := f.folder(f.sc.RootID, "Recipes")
	f.link(recipes, "First", "https://example.org/first")
	id := f.link(later, "Go", "https://go.dev", "dev", "old")
	other := f.link(later, "Rust", "https://rust-lang.org", "old")
	num := f.num(id)

	// The app sends the whole link back with its changes.
	link, err := f.s.Link(f.ctx, f.sc, num)
	if err != nil {
		t.Fatal(err)
	}
	var in UpdateLinkInput
	body := jsonOf(t, link)
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		t.Fatal(err)
	}
	in.Name = json.RawMessage(`"Docs"`)
	in.Tags = &[]TagRef{{Name: "dev"}, {Name: "go"}}
	in.Collection = &CollectionRef{ID: ptr(f.num(recipes))}
	got, err := f.s.UpdateLink(f.ctx, f.sc, num, in)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, tag := range got.Tags {
		tags = append(tags, tag.Name)
	}
	if got.Name != "Docs" || !reflect.DeepEqual(tags, []string{"dev", "go"}) || got.CollectionID != f.num(recipes) {
		t.Errorf("updated = %s", jsonOf(t, got))
	}
	if kids := f.children(recipes); !reflect.DeepEqual(kids, []string{"Docs", "First"}) {
		t.Errorf("Recipes = %v", kids)
	}
	if n := f.nodes()[other]; len(n.Tags) != 1 || n.Tags[0].Name != "old" {
		t.Errorf("other link's tags = %v", n.Tags)
	}

	// null removes the title; a missing name keeps it.
	got, err = f.s.UpdateLink(f.ctx, f.sc, num, UpdateLinkInput{Name: json.RawMessage(`null`), Tags: &[]TagRef{}, Collection: &CollectionRef{ID: ptr(f.num(recipes))}})
	if err != nil || got.Name != "" {
		t.Errorf("null name: %q, err %v", got.Name, err)
	}
	if _, err := f.s.UpdateLink(f.ctx, f.sc, num, UpdateLinkInput{URL: ptr("https://go.dev/doc"), Tags: &[]TagRef{}, Collection: &CollectionRef{ID: ptr(f.num(recipes))}}); err != nil {
		t.Fatal(err)
	}
	if n := f.nodes()[id]; n.Title != nil || *n.URL != "https://go.dev/doc" {
		t.Errorf("after URL change: %+v", n)
	}
}

func TestUpdateLinkRejections(t *testing.T) {
	f := newFixture(t)
	work := f.folder(f.topLevel, "Work")
	id := f.link(f.sc.RootID, "Go", "https://go.dev", "dev")
	num, root := f.num(id), f.num(f.sc.RootID)
	valid := func() UpdateLinkInput {
		return UpdateLinkInput{Name: json.RawMessage(`"Changed"`), Tags: &[]TagRef{{Name: "new"}}, Collection: &CollectionRef{ID: &root}}
	}
	for name, change := range map[string]func(in *UpdateLinkInput){
		"pin":             func(in *UpdateLinkInput) { in.PinnedBy = []*PinRef{{ID: ptr(int64(1))}} },
		"description":     func(in *UpdateLinkInput) { in.Description = json.RawMessage(strconv.Quote(strings.Repeat("x", 2049))) },
		"bad description": func(in *UpdateLinkInput) { in.Description = json.RawMessage(`42`) },
		"color":           func(in *UpdateLinkInput) { in.Color = ptr("#fff") },
		"icon":            func(in *UpdateLinkInput) { in.Icon = ptr("star") },
		"no tags":         func(in *UpdateLinkInput) { in.Tags = nil },
		"no collection":   func(in *UpdateLinkInput) { in.Collection = nil },
		"outside":         func(in *UpdateLinkInput) { in.Collection = &CollectionRef{ID: ptr(f.num(work))} },
		"invalid url":     func(in *UpdateLinkInput) { in.URL = ptr("example") },
		"bad tag":         func(in *UpdateLinkInput) { in.Tags = &[]TagRef{{Name: ""}} },
	} {
		in := valid()
		change(&in)
		if _, err := f.s.UpdateLink(f.ctx, f.sc, num, in); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if n := f.nodes()[id]; *n.Title != "Go" || len(n.Tags) != 1 || n.Tags[0].Name != "dev" || n.ParentID != f.sc.RootID {
		t.Errorf("link changed by a rejected update: %+v", n)
	}
	// Empty pin entries and fields the API returns as null or "" pass.
	in := valid()
	in.PinnedBy, in.Description, in.Icon = []*PinRef{}, json.RawMessage(`""`), nil
	if _, err := f.s.UpdateLink(f.ctx, f.sc, num, in); err != nil {
		t.Errorf("round trip: %v", err)
	}
}

func TestLinkDescriptions(t *testing.T) {
	f := newFixture(t)
	created, err := f.s.CreateLink(f.ctx, f.sc, CreateLinkInput{URL: ptr("https://example.org"), Description: ptr(" Read on the weekend ")})
	if err != nil || created.Description != "Read on the weekend" {
		t.Fatalf("create: %q, %v", created.Description, err)
	}
	root := f.num(f.sc.RootID)
	update := func(raw string) Link {
		t.Helper()
		in := UpdateLinkInput{Tags: &[]TagRef{}, Collection: &CollectionRef{ID: &root}}
		if raw != "" {
			in.Description = json.RawMessage(raw)
		}
		got, err := f.s.UpdateLink(f.ctx, f.sc, created.ID, in)
		if err != nil {
			t.Fatalf("update %s: %v", raw, err)
		}
		return got
	}
	for _, tt := range []struct{ raw, want string }{
		{`"Updated notes"`, "Updated notes"},
		{"", "Updated notes"}, // missing: kept
		{`null`, ""},
		{`"Again"`, "Again"},
		{`""`, ""},
	} {
		if got := update(tt.raw); got.Description != tt.want {
			t.Errorf("update %s: description %q, want %q", tt.raw, got.Description, tt.want)
		}
	}
	if d := f.nodes()[f.linkID(created.ID)].Description; d != nil {
		t.Errorf("stored description = %q, want none", *d)
	}
}

// linkID returns the node ID of the link with an integer ID.
func (f *fixture) linkID(num int64) string {
	f.t.Helper()
	for id, n := range f.nodes() {
		if n.Num == num {
			return id
		}
	}
	f.t.Fatalf("no link %d", num)
	return ""
}

func TestDeleteLink(t *testing.T) {
	f := newFixture(t)
	id := f.link(f.sc.RootID, "Go", "https://go.dev", "dev")
	link, err := f.s.DeleteLink(f.ctx, f.sc, f.num(id))
	if err != nil || link.Name != "Go" {
		t.Fatalf("deleted = %+v, err %v", link, err)
	}
	if _, ok := f.nodes()[id]; ok {
		t.Error("link still exists")
	}
	if tags, _ := f.s.Tags(f.ctx, f.sc, TagsInput{}); len(tags.Tags) != 0 {
		t.Errorf("tags = %v, want the unused tag removed", tags.Tags)
	}
}

func TestCollections(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	videos := f.folder(later, "Videos")
	f.folder(f.sc.RootID, "Recipes")
	for _, title := range []string{"A", "B", "C"} {
		f.link(later, title, "https://example.org/"+title)
	}
	f.link(videos, "V1", "https://example.org/v1")
	f.link(videos, "V2", "https://example.org/v2")

	list, err := f.s.Collections(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range list {
		parent := ""
		if c.Parent != nil {
			parent = c.Parent.Name
		}
		got = append(got, c.Name+"<"+parent+":"+string(rune('0'+c.Count.Links)))
	}
	if want := []string{"Reading<:0", "Later<Reading:3", "Videos<Later:2", "Recipes<Reading:0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("collections = %v, want %v", got, want)
	}

	// Create at the end of the root or of a parent.
	c, err := f.s.CreateCollection(f.ctx, f.sc, CreateCollectionInput{Name: "Books", Color: ptr("#f00"), Icon: ptr("book")})
	if err != nil || c.ParentID == nil || *c.ParentID != f.num(f.sc.RootID) {
		t.Fatalf("created = %+v, err %v", c, err)
	}
	if kids := f.children(f.sc.RootID); kids[len(kids)-1] != "Books" {
		t.Errorf("root = %v", kids)
	}
	if c, err := f.s.CreateCollection(f.ctx, f.sc, CreateCollectionInput{Name: "Talks", ParentID: ptr(f.num(videos))}); err != nil || c.Parent.Name != "Videos" {
		t.Errorf("nested = %+v, err %v", c, err)
	}
	work := f.folder(f.topLevel, "Work")
	for name, in := range map[string]CreateCollectionInput{
		"empty name":  {Name: " "},
		"description": {Name: "X", Description: ptr("About")},
		"outside":     {Name: "X", ParentID: ptr(f.num(work))},
	} {
		if _, err := f.s.CreateCollection(f.ctx, f.sc, in); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	// Delete: the folder goes with its links and subfolders, as in Linkwarden.
	before := len(f.nodes())
	deleted, err := f.s.DeleteCollection(f.ctx, f.sc, f.num(later))
	if err != nil || deleted.Name != "Later" || deleted.Count.Links != 3 {
		t.Fatalf("deleted = %+v, err %v", deleted, err)
	}
	if kids := f.children(f.sc.RootID); !reflect.DeepEqual(kids, []string{"Recipes", "Books"}) {
		t.Errorf("root after delete = %v", kids)
	}
	// Later, Videos, Talks, A, B, C, V1 and V2 are gone.
	if after := len(f.nodes()); after != before-8 {
		t.Errorf("nodes after delete = %d, want %d", after, before-8)
	}
	if _, err := f.s.DeleteCollection(f.ctx, f.sc, f.num(f.sc.RootID)); !errors.Is(err, ErrRootCollection) {
		t.Errorf("delete root: err = %v", err)
	}
}

func TestGetAndEditCollection(t *testing.T) {
	f := newFixture(t)
	later := f.folder(f.sc.RootID, "Later")
	videos := f.folder(later, "Videos")
	recipes := f.folder(f.sc.RootID, "Recipes")
	work := f.folder(f.topLevel, "Work")

	c, err := f.s.Collection(f.ctx, f.sc, f.num(later))
	if err != nil || c.Name != "Later" || c.Parent == nil || c.Parent.Name != "Reading" {
		t.Fatalf("get = %+v, err %v", c, err)
	}
	if _, err := f.s.Collection(f.ctx, f.sc, f.num(work)); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("get outside: err = %v", err)
	}

	// Floccus sends back what GET returned, with a new name and parent.
	echo := func(id string, change map[string]any) EditCollectionInput {
		t.Helper()
		got, err := f.s.Collection(f.ctx, f.sc, f.num(id))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(jsonOf(t, got)), &m); err != nil {
			t.Fatal(err)
		}
		for k, v := range change {
			m[k] = v
		}
		var in EditCollectionInput
		if err := json.Unmarshal([]byte(jsonOf(t, m)), &in); err != nil {
			t.Fatal(err)
		}
		return in
	}

	renamed, err := f.s.EditCollection(f.ctx, f.sc, f.num(later), echo(later, map[string]any{"name": "Read later"}))
	if err != nil || renamed.Name != "Read later" {
		t.Fatalf("rename = %+v, err %v", renamed, err)
	}
	moved, err := f.s.EditCollection(f.ctx, f.sc, f.num(videos), echo(videos, map[string]any{"parentId": f.num(recipes)}))
	if err != nil || moved.Parent == nil || moved.Parent.Name != "Recipes" {
		t.Fatalf("move = %+v, err %v", moved, err)
	}
	if _, err := f.s.EditCollection(f.ctx, f.sc, f.num(videos), echo(videos, map[string]any{"parentId": nil})); err != nil {
		t.Fatal(err)
	}
	if kids := f.children(f.sc.RootID); kids[len(kids)-1] != "Videos" {
		t.Errorf("root after move to null = %v", kids)
	}
	// Unchanged root: no-op.
	if _, err := f.s.EditCollection(f.ctx, f.sc, f.num(f.sc.RootID), echo(f.sc.RootID, nil)); err != nil {
		t.Errorf("unchanged root: %v", err)
	}

	sub := f.folder(later, "Sub")
	before := f.nodes()
	for name, tc := range map[string]struct {
		id     string
		change map[string]any
		want   error
	}{
		"into itself":    {later, map[string]any{"parentId": f.num(later)}, nil},
		"into its child": {later, map[string]any{"parentId": f.num(sub)}, nil},
		"outside parent": {later, map[string]any{"parentId": f.num(work)}, errNotAccessible},
		"description":    {later, map[string]any{"description": "About"}, nil},
		"public":         {later, map[string]any{"isPublic": true}, nil},
		"members":        {later, map[string]any{"members": []any{map[string]any{"userId": 2}}}, nil},
		"rename root":    {f.sc.RootID, map[string]any{"name": "Other"}, ErrRootCollectionEdit},
		"move root":      {f.sc.RootID, map[string]any{"parentId": f.num(later)}, ErrRootCollectionEdit},
	} {
		_, err := f.s.EditCollection(f.ctx, f.sc, f.num(tc.id), echo(tc.id, tc.change))
		if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if !reflect.DeepEqual(f.nodes(), before) {
		t.Error("a rejected edit changed the tree")
	}
}

func TestTags(t *testing.T) {
	f := newFixture(t)
	work := f.folder(f.topLevel, "Work")
	a := f.link(f.sc.RootID, "A", "https://example.org/a", "dev", "design")
	f.link(f.sc.RootID, "B", "https://example.org/b", "dev")
	outside := f.link(work, "W", "https://example.org/w", "dev", "work")

	list := func(in TagsInput) []string {
		t.Helper()
		page, err := f.s.Tags(f.ctx, f.sc, in)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, tag := range page.Tags {
			out = append(out, tag.Name+":"+string(rune('0'+tag.Count.Links)))
		}
		return out
	}
	if got := list(TagsInput{Sort: TagSortNameAZ}); !reflect.DeepEqual(got, []string{"design:1", "dev:2"}) {
		t.Errorf("A–Z = %v", got)
	}
	if got := list(TagsInput{Sort: TagSortMostLinks}); got[0] != "dev:2" {
		t.Errorf("most links = %v", got)
	}
	if got := list(TagsInput{Sort: TagSortNameAZ, Search: "DE"}); !reflect.DeepEqual(got, []string{"design:1", "dev:2"}) {
		t.Errorf("search = %v", got)
	}
	if got := list(TagsInput{Search: "dev"}); !reflect.DeepEqual(got, []string{"dev:2"}) {
		t.Errorf("search dev = %v", got)
	}
	if _, err := f.s.Tag(f.ctx, f.sc, f.tagNum(outside, "work")); !errors.Is(err, ErrTagNotFound) {
		t.Errorf("tag outside: err = %v", err)
	}

	// Delete: links inside lose it, links outside keep it.
	dev := f.tagNum(a, "dev")
	if _, err := f.s.DeleteTag(f.ctx, f.sc, dev); err != nil {
		t.Fatal(err)
	}
	if got := list(TagsInput{}); !reflect.DeepEqual(got, []string{"design:1"}) {
		t.Errorf("after delete = %v", got)
	}
	if n := f.nodes()[outside]; len(n.Tags) != 2 {
		t.Errorf("outside tags = %v", n.Tags)
	}
	if err := f.s.CreateTag(); !errors.Is(err, ErrTagCreate) {
		t.Errorf("create tag: err = %v", err)
	}
}

func TestTagPages(t *testing.T) {
	f := newFixture(t)
	var tags []string
	for i := range 55 {
		tags = append(tags, "t"+string(rune('A'+i/26))+string(rune('a'+i%26)))
	}
	f.link(f.sc.RootID, "A", "https://example.org/a", tags...)
	first, err := f.s.Tags(f.ctx, f.sc, TagsInput{Sort: TagSortNameAZ})
	if err != nil || len(first.Tags) != 50 || first.NextCursor == nil {
		t.Fatalf("first page: %d tags, cursor %v, err %v", len(first.Tags), first.NextCursor, err)
	}
	second, err := f.s.Tags(f.ctx, f.sc, TagsInput{Sort: TagSortNameAZ, Cursor: *first.NextCursor})
	if err != nil || len(second.Tags) != 5 || second.NextCursor != nil || second.Tags[4].Name != "tCc" {
		t.Fatalf("second page: %v, cursor %v, err %v", second.Tags, second.NextCursor, err)
	}
}

func TestMeConfigAndDashboard(t *testing.T) {
	f := newFixture(t)
	f.user.Name = ptr("Ann")
	me := jsonOf(t, Me(f.user))
	for _, want := range []string{
		`"id":1,"uuid":"` + f.user.ID + `","name":"Ann","username":"` + f.user.Slug + `","email":"ann@example.org"`,
		`"linksRouteTo":"ORIGINAL"`, `"hasPassword":false`, `"subscription":{"active":false,"quantity":0}`,
		`"parentSubscription":{"active":false,"user":{}}`, `"type":"STATS"`, `"type":"RECENT_LINKS"`, `"collectionOrder":[]`,
	} {
		if !contains(me, want) {
			t.Errorf("me = %s\nmissing %s", me, want)
		}
	}
	config := jsonOf(t, Config())
	for _, want := range []string{`"INSTANCE_VERSION":"v2.16.3"`, `"DISABLE_REGISTRATION":true`, `"STRIPE_ENABLED":null`, `"REQUIRE_CC":null`} {
		if !contains(config, want) {
			t.Errorf("config = %s\nmissing %s", config, want)
		}
	}

	for i := range 20 {
		f.link(f.sc.RootID, "L"+string(rune('a'+i)), "https://example.org/"+string(rune('a'+i)), "t"+string(rune('a'+i%4)))
	}
	d, err := f.s.Dashboard(f.ctx, f.sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Links) != 16 || d.Links[0].Name != "Lt" || d.NumberOfTags != 4 || d.NumberOfPinnedLinks != 0 || jsonOf(t, d.CollectionLinks) != "{}" {
		t.Errorf("dashboard = %d links, first %s, %d tags", len(d.Links), d.Links[0].Name, d.NumberOfTags)
	}
}

func TestPreview(t *testing.T) {
	f := newFixture(t)
	png := f.link(f.sc.RootID, "Go", "https://go.dev/tour")
	ico := f.link(f.sc.RootID, "Ico", "https://ico.example/")
	img, err := f.s.Preview(f.ctx, f.sc, f.num(png))
	if err != nil || img.ID != "icon-go" || img.ContentType != "image/png" {
		t.Errorf("preview = %+v, err %v", img, err)
	}
	if _, err := f.s.Preview(f.ctx, f.sc, f.num(ico)); !errors.Is(err, ErrPreviewNotFound) {
		t.Errorf("ico preview: err = %v", err)
	}
	if _, err := f.s.Preview(f.ctx, f.sc, 999); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("unknown link: err = %v", err)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s[:len(sub)] == sub || contains(s[1:], sub)))
}
