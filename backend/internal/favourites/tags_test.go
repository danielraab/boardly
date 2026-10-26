package favourites

import (
	"errors"
	"reflect"
	"testing"
)

// tagID returns the ID of the user's tag with the given name.
func (f *fixture) tagID(name string) string {
	f.t.Helper()
	tag, err := f.s.TagByName(f.ctx, f.user, name)
	if err != nil {
		f.t.Fatalf("tag %q: %v", name, err)
	}
	return tag.ID
}

func TestListTags(t *testing.T) {
	f := newFixture(t)
	f.link("", "A", "reading", "Tools")
	f.link("", "B", "reading")
	if _, err := f.s.Create(f.ctx, f.other, NewNode{Kind: KindLink, URL: ptr("https://example.org/x"), Tags: []string{"bobs"}}); err != nil {
		t.Fatal(err)
	}

	tags, err := f.s.ListTags(f.ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	got := []TagUsage{}
	for _, tag := range tags {
		got = append(got, TagUsage{Name: tag.Name, LinkCount: tag.LinkCount})
	}
	want := []TagUsage{{Name: "reading", LinkCount: 2}, {Name: "Tools", LinkCount: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %+v, want %+v", got, want)
	}
}

func TestMergeTag(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A", "Reading")
	b := f.link("", "B", "Reading", "reading-list")
	c := f.link("", "C", "reading-list")
	target := f.tagID("Reading")

	merged, err := f.s.MergeTag(f.ctx, f.user, f.tagID("reading-list"), target)
	if err != nil {
		t.Fatal(err)
	}
	if merged.ID != target || merged.Name != "Reading" {
		t.Fatalf("merged = %+v", merged)
	}
	for _, id := range []string{a, b, c} {
		if got := f.tags(id); !reflect.DeepEqual(got, []string{"Reading"}) {
			t.Fatalf("tags of %s = %q", id, got)
		}
	}
	if _, err := f.s.TagByName(f.ctx, f.user, "reading-list"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("source still exists: %v", err)
	}

	// Unknown, foreign or the same tag.
	theirs, err := f.s.Create(f.ctx, f.other, NewNode{Kind: KindLink, URL: ptr("https://example.org/x"), Tags: []string{"bobs"}})
	if err != nil {
		t.Fatal(err)
	}
	var bobs string
	if err := f.s.DB.QueryRow(`SELECT tag_id FROM favourite_node_tags WHERE node_id = ?`, theirs.ID).Scan(&bobs); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"nope", target}, {target, "nope"}, {target, target}, {bobs, target}, {target, bobs}} {
		if _, err := f.s.MergeTag(f.ctx, f.user, pair[0], pair[1]); !errors.Is(err, ErrNotFound) {
			t.Errorf("merge %v: %v, want ErrNotFound", pair, err)
		}
	}
}

func TestDeleteTag(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A", "old", "keep")
	b := f.link("", "B", "old")

	if err := f.s.DeleteTag(f.ctx, f.user, f.tagID("old")); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("links = %q", got)
	}
	if got := f.tags(a); !reflect.DeepEqual(got, []string{"keep"}) {
		t.Fatalf("A's tags = %q", got)
	}
	if got := f.tags(b); len(got) != 0 {
		t.Fatalf("B's tags = %q", got)
	}
	if n := f.tagCount(); n != 1 {
		t.Fatalf("tags left: %d", n)
	}
	if err := f.s.DeleteTag(f.ctx, f.user, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown tag: %v", err)
	}
	if err := f.s.DeleteTag(f.ctx, f.other, f.tagID("keep")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user's tag: %v", err)
	}
}
