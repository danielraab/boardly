package favourites

import (
	"errors"
	"testing"
)

func TestScope(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	later := f.folder(reading, "Later")
	article := f.link(later, "Article")
	work := f.folder("", "Work")
	docs := f.link(work, "Docs")
	root, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)
	bobRoot, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.other)

	sc := Scope{UserID: f.user, RootID: reading}
	for id, want := range map[string]bool{
		reading: true, later: true, article: true,
		work: false, docs: false, root: false, bobRoot: false, "nope": false,
	} {
		if got, err := f.s.Contains(f.ctx, sc, id); err != nil || got != want {
			t.Errorf("Contains(%s) = %v, err %v; want %v", f.name(id), got, err, want)
		}
	}
	// Another user's scope sees none of Ann's nodes, even with Ann's root as its root.
	if got, _ := f.s.Contains(f.ctx, Scope{UserID: f.other, RootID: reading}, article); got {
		t.Error("another user's scope contains Ann's link")
	}

	whole := Scope{UserID: f.user, RootID: root}
	for _, id := range []string{root, reading, article, docs} {
		if got, err := f.s.Contains(f.ctx, whole, id); err != nil || !got {
			t.Errorf("whole tree: Contains(%s) = %v, err %v", f.name(id), got, err)
		}
	}

	for name, tc := range map[string]struct {
		err  error
		want error
	}{
		"create in root":         {f.s.CheckCreate(f.ctx, sc, reading), nil},
		"create below":           {f.s.CheckCreate(f.ctx, sc, later), nil},
		"create outside":         {f.s.CheckCreate(f.ctx, sc, work), ErrNotFound},
		"change inside":          {f.s.CheckChange(f.ctx, sc, article), nil},
		"change root":            {f.s.CheckChange(f.ctx, sc, reading), ErrRootImmutable},
		"change outside":         {f.s.CheckChange(f.ctx, sc, docs), ErrNotFound},
		"move inside":            {f.s.CheckMove(f.ctx, sc, article, reading), nil},
		"move out":               {f.s.CheckMove(f.ctx, sc, article, work), ErrNotFound},
		"move in from outside":   {f.s.CheckMove(f.ctx, sc, docs, reading), ErrNotFound},
		"move root":              {f.s.CheckMove(f.ctx, sc, reading, later), ErrRootImmutable},
		"whole tree: move":       {f.s.CheckMove(f.ctx, whole, docs, reading), nil},
		"whole tree: change top": {f.s.CheckChange(f.ctx, whole, root), ErrRootImmutable},
	} {
		if !errors.Is(tc.err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, tc.err, tc.want)
		}
	}
}

// name returns a node's title for test messages.
func (f *fixture) name(id string) string {
	if n := find(f.tree(), id); n != nil && n.Title != nil {
		return *n.Title
	}
	return id
}
