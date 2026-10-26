package favourites

import (
	"errors"
	"reflect"
	"testing"
)

func (f *fixture) tags(id string) []string {
	f.t.Helper()
	n := find(f.tree(), id)
	if n == nil {
		f.t.Fatalf("node %s not in tree", id)
	}
	return n.Tags
}

func TestBulkDelete(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A", "old")
	f.link("", "B")
	c := f.link("", "C")
	work := f.folder("", "Work")
	d := f.link(work, "D", "keep")

	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{c, a, d}, Action: BulkDelete}); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"B", "Work"}) {
		t.Fatalf("root after delete: %q", got)
	}
	if got := f.titles(work); len(got) != 0 {
		t.Fatalf("Work after delete: %q", got)
	}
	if n := f.tagCount(); n != 0 {
		t.Fatalf("unused tags left: %d", n)
	}
}

func TestBulkTags(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A", "Reading", "x")
	b := f.link("", "B")
	c := f.link("", "C", "old")
	other := f.link("", "Other", "old")

	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{a, b}, Action: BulkAddTags, Tags: []string{"reading", "later"}}); err != nil {
		t.Fatal(err)
	}
	if got := f.tags(a); !reflect.DeepEqual(got, []string{"later", "Reading", "x"}) {
		t.Fatalf("A after add: %q", got)
	}
	if got := f.tags(b); !reflect.DeepEqual(got, []string{"later", "Reading"}) {
		t.Fatalf("B after add: %q", got)
	}

	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{a, c}, Action: BulkRemoveTags, Tags: []string{"OLD", "x"}}); err != nil {
		t.Fatal(err)
	}
	if got := f.tags(c); len(got) != 0 {
		t.Fatalf("C after remove: %q", got)
	}
	if got := f.tags(other); !reflect.DeepEqual(got, []string{"old"}) {
		t.Fatalf("Other keeps its tag: %q", got)
	}

	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{a, b, c}, Action: BulkSetTags, Tags: []string{"archive"}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a, b, c} {
		if got := f.tags(id); !reflect.DeepEqual(got, []string{"archive"}) {
			t.Fatalf("after set: %q", got)
		}
	}
	// "later", "Reading" and "x" are on no link anymore; "old" and "archive" are.
	if n := f.tagCount(); n != 2 {
		t.Fatalf("tags left: %d", n)
	}

	// Set to nothing clears them.
	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{a}, Action: BulkSetTags, Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	if got := f.tags(a); len(got) != 0 {
		t.Fatalf("after clearing: %q", got)
	}
}

func TestBulkMove(t *testing.T) {
	f := newFixture(t)
	inbox := f.folder("", "Inbox")
	a := f.link(inbox, "A")
	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	f.link(tools, "Git")
	b := f.link(work, "B")
	archive := f.folder("", "Archive")
	f.link(archive, "Old")

	if err := f.s.Bulk(f.ctx, f.user, BulkOp{IDs: []string{a, tools, b}, Action: BulkMove, ParentID: archive}); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(archive); !reflect.DeepEqual(got, []string{"Old", "A", "Tools", "B"}) {
		t.Fatalf("Archive after move: %q", got)
	}
	if got := f.titles(tools); !reflect.DeepEqual(got, []string{"Git"}) {
		t.Fatalf("Tools keeps its contents: %q", got)
	}
	if got := f.titles(work); len(got) != 0 {
		t.Fatalf("Work after move: %q", got)
	}
	if got := f.titles(inbox); len(got) != 0 {
		t.Fatalf("Inbox after move: %q", got)
	}
}

func TestBulkRejects(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	git := f.link(tools, "Git")
	a := f.link("", "A")
	root, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)
	theirs, err := f.s.Create(f.ctx, f.other, NewNode{Kind: KindLink, URL: ptr("https://example.org/x")})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		op   BulkOp
		want error
	}{
		{"no IDs", BulkOp{Action: BulkDelete}, ErrInvalidField},
		{"duplicate IDs", BulkOp{IDs: []string{a, a}, Action: BulkDelete}, ErrInvalidField},
		{"unknown action", BulkOp{IDs: []string{a}, Action: "copy"}, ErrInvalidField},
		{"unknown ID", BulkOp{IDs: []string{a, "nope"}, Action: BulkDelete}, ErrNotFound},
		{"other user's node", BulkOp{IDs: []string{a, theirs.ID}, Action: BulkDelete}, ErrNotFound},
		{"folder delete", BulkOp{IDs: []string{a, work}, Action: BulkDelete}, ErrInvalidField},
		{"folder tags", BulkOp{IDs: []string{work}, Action: BulkAddTags, Tags: []string{"x"}}, ErrInvalidField},
		{"no tags to add", BulkOp{IDs: []string{a}, Action: BulkAddTags}, ErrInvalidField},
		{"invalid tag", BulkOp{IDs: []string{a}, Action: BulkSetTags, Tags: []string{" "}}, ErrInvalidTag},
		{"root", BulkOp{IDs: []string{root}, Action: BulkMove, ParentID: work}, ErrInvalidField},
		{"no target", BulkOp{IDs: []string{a}, Action: BulkMove}, ErrInvalidField},
		{"link target", BulkOp{IDs: []string{git}, Action: BulkMove, ParentID: a}, ErrInvalidParent},
		{"nested IDs", BulkOp{IDs: []string{work, git}, Action: BulkMove, ParentID: root}, ErrInvalidMove},
		{"into own subtree", BulkOp{IDs: []string{a, work}, Action: BulkMove, ParentID: tools}, ErrInvalidMove},
		{"into itself", BulkOp{IDs: []string{work}, Action: BulkMove, ParentID: work}, ErrInvalidMove},
	}
	for _, c := range cases {
		if err := f.s.Bulk(f.ctx, f.user, c.op); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	// Nothing changed, also not the valid parts of rejected requests.
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"Work", "A"}) {
		t.Fatalf("root changed: %q", got)
	}
	if got := f.titles(tools); !reflect.DeepEqual(got, []string{"Git"}) {
		t.Fatalf("Tools changed: %q", got)
	}
}
