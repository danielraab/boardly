package favourites

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestCreateFirst(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	f.link(reading, "A")
	f.link(reading, "B")
	n, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: reading, Kind: KindLink, URL: ptr("https://example.org/new"), Title: ptr("New"), First: true})
	if err != nil {
		t.Fatal(err)
	}
	f.link(reading, "C")
	if got, want := f.titles(reading), []string{"New", "A", "B", "C"}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
	if n.Title == nil || *n.Title != "New" {
		t.Errorf("created = %+v", n)
	}
}

func TestSubtreeNodes(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	later := f.folder(reading, "Later")
	f.link(later, "Go", "dev", "Alpha")
	f.link(reading, "Rust")
	work := f.folder("", "Work")
	f.link(work, "Docs", "dev")
	bob, _ := f.s.Create(f.ctx, f.other, NewNode{Kind: KindLink, URL: ptr("https://example.org/bob")})

	nodes, err := f.s.SubtreeNodes(f.ctx, Scope{UserID: f.user, RootID: reading})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range nodes {
		got = append(got, f.name(n.ID)+"<"+f.name(n.ParentID))
		if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
			t.Errorf("%s has no times", f.name(n.ID))
		}
		if n.ID == bob.ID {
			t.Error("Bob's link is in Ann's subtree")
		}
		if title := f.name(n.ID); title == "Go" {
			if len(n.Tags) != 2 || n.Tags[0].Name != "Alpha" || n.Tags[1].Name != "dev" || n.URL == nil {
				t.Errorf("Go = %+v", n)
			}
		}
	}
	// The root comes first (it has no parent in the scope), then the children by position.
	want := []string{"Reading<", "Later<Reading", "Rust<Reading", "Go<Later"}
	if len(got) != len(want) {
		t.Fatalf("nodes = %v, want %v", got, want)
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			found = found || g == w
		}
		if !found {
			t.Errorf("nodes = %v, missing %s", got, w)
		}
	}

	for name, sc := range map[string]Scope{
		"other user": {UserID: f.other, RootID: reading},
		"unknown":    {UserID: f.user, RootID: "nope"},
	} {
		if nodes, err := f.s.SubtreeNodes(f.ctx, sc); err != nil || len(nodes) != 0 {
			t.Errorf("%s: %d nodes, err %v", name, len(nodes), err)
		}
	}
}

func TestRenameAndDetachTag(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	inside := f.link(reading, "Go", "dev")
	outside := f.link("", "Docs", "dev", "work")
	tagID := func(name string) string {
		t.Helper()
		var id string
		if err := f.s.DB.QueryRow(`SELECT id FROM tags WHERE user_id = ? AND name = ?`, f.user, name).Scan(&id); err != nil {
			t.Fatalf("tag %s: %v", name, err)
		}
		return id
	}
	dev := tagID("dev")

	if n, err := f.s.TagLinkCount(f.ctx, f.user, dev); err != nil || n != 2 {
		t.Errorf("TagLinkCount = %d, %v", n, err)
	}
	if _, err := f.s.TagLinkCount(f.ctx, f.other, dev); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user's TagLinkCount err = %v", err)
	}

	if _, err := f.s.RenameTag(f.ctx, f.user, dev, "WORK"); !errors.Is(err, ErrTagExists) {
		t.Errorf("rename to existing: %v", err)
	}
	if _, err := f.s.RenameTag(f.ctx, f.user, dev, " "); !errors.Is(err, ErrInvalidTag) {
		t.Errorf("rename to empty: %v", err)
	}
	if _, err := f.s.RenameTag(f.ctx, f.other, dev, "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("rename other user's tag: %v", err)
	}
	if tag, err := f.s.RenameTag(f.ctx, f.user, dev, " Development "); err != nil || tag.Name != "Development" {
		t.Fatalf("rename = %+v, %v", tag, err)
	}
	if got := find(f.tree(), outside).Tags; !reflect.DeepEqual(got, []string{"Development", "work"}) {
		t.Errorf("tags after rename = %v", got)
	}

	// Detaching from the inside link keeps the tag on the outside link.
	if err := f.s.DetachTag(f.ctx, f.user, dev, []string{inside}); err != nil {
		t.Fatal(err)
	}
	if got := find(f.tree(), inside).Tags; len(got) != 0 {
		t.Errorf("inside tags = %v", got)
	}
	if got := find(f.tree(), outside).Tags; !reflect.DeepEqual(got, []string{"Development", "work"}) {
		t.Errorf("outside tags = %v", got)
	}
	// Detaching from the last link removes the tag.
	if err := f.s.DetachTag(f.ctx, f.user, dev, []string{outside}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.TagLinkCount(f.ctx, f.user, dev); !errors.Is(err, ErrNotFound) {
		t.Errorf("unused tag still exists: %v", err)
	}
}

func TestIntegerIDs(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	work := f.folder("", "Work")
	goLink := f.link(reading, "Go", "dev")
	nums := func() (map[string]int64, map[string]int64) {
		t.Helper()
		nodes, err := f.s.SubtreeNodes(f.ctx, Scope{UserID: f.user, RootID: f.tree().ID})
		if err != nil {
			t.Fatal(err)
		}
		byNode, byTag := map[string]int64{}, map[string]int64{}
		for _, n := range nodes {
			if n.Num <= 0 {
				t.Errorf("%s has num %d", f.name(n.ID), n.Num)
			}
			byNode[n.ID] = n.Num
			for _, tag := range n.Tags {
				byTag[tag.Name] = tag.Num
				if tag.CreatedAt.IsZero() {
					t.Errorf("tag %s has no creation time", tag.Name)
				}
			}
		}
		return byNode, byTag
	}
	before, tags := nums()
	if tags["dev"] <= 0 {
		t.Fatalf("tag nums = %v", tags)
	}

	// Moving and renaming keeps the number.
	if err := f.s.Move(f.ctx, f.user, goLink, work, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Update(f.ctx, f.user, goLink, Patch{Title: ptr("Golang")}); err != nil {
		t.Fatal(err)
	}
	after, _ := nums()
	if after[goLink] != before[goLink] {
		t.Errorf("num after move and rename = %d, want %d", after[goLink], before[goLink])
	}

	// Numbers aren't reused after the highest one is deleted.
	if err := f.s.Delete(f.ctx, f.user, goLink, DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	next := f.link(reading, "Rust")
	latest, _ := nums()
	if latest[next] <= before[goLink] {
		t.Errorf("new num %d, want more than the deleted %d", latest[next], before[goLink])
	}

	// The web API doesn't show integer IDs.
	data, err := json.Marshal(Tag{ID: "t", Name: "dev", Num: 3})
	if err != nil || string(data) != `{"id":"t","name":"dev"}` {
		t.Errorf("tag JSON = %s, err %v", data, err)
	}
}
