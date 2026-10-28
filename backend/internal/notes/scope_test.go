package notes

import (
	"errors"
	"slices"
	"testing"
)

func TestScopedNotes(t *testing.T) {
	f := newFixture(t)
	home := f.category("Home", nil)
	garden := f.category("Garden", &home)
	work := f.category("Work", nil)
	inHome := f.note("In home", &home)
	inGarden := f.note("In garden", &garden)
	inWork := f.note("In work", &work)
	loose := f.note("Uncategorized", nil)
	bobs, err := f.s.CreateNote(f.ctx, f.other, NewNote{Title: ptr("Bob's")})
	if err != nil {
		t.Fatal(err)
	}

	titles := func(sc Scope) []string {
		t.Helper()
		ns, err := f.s.ScopedNotes(f.ctx, sc)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, n := range ns {
			out = append(out, *n.Title)
		}
		slices.Sort(out)
		return out
	}
	for name, tc := range map[string]struct {
		sc   Scope
		want []string
	}{
		"all":                   {Scope{UserID: f.user, Mode: ScopeAll}, []string{"In garden", "In home", "In work", "Uncategorized"}},
		"category":              {Scope{UserID: f.user, Mode: ScopeCategory, CategoryID: home}, []string{"In garden", "In home"}},
		"subcategory":           {Scope{UserID: f.user, Mode: ScopeCategory, CategoryID: garden}, []string{"In garden"}},
		"none":                  {Scope{UserID: f.user, Mode: ScopeNone}, nil},
		"other user's category": {Scope{UserID: f.other, Mode: ScopeCategory, CategoryID: home}, nil},
	} {
		if got := titles(tc.sc); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}

	homeScope := Scope{UserID: f.user, Mode: ScopeCategory, CategoryID: home}
	for _, id := range []string{inHome.ID, inGarden.ID} {
		if _, err := f.s.ScopedNote(f.ctx, homeScope, id); err != nil {
			t.Errorf("note %s in scope: %v", id, err)
		}
	}
	for _, id := range []string{inWork.ID, loose.ID, bobs.ID, "nope"} {
		if _, err := f.s.ScopedNote(f.ctx, homeScope, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("note %s outside the scope: err = %v", id, err)
		}
	}
	if _, err := f.s.ScopedNote(f.ctx, Scope{UserID: f.user, Mode: ScopeAll}, bobs.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another user's note with all notes: err = %v", err)
	}
}
