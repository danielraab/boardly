package notes

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestListedNotes(t *testing.T) {
	f := newFixture(t)
	home := f.category("Home", nil)
	garden := f.category("Garden", &home)
	f.category("Work", nil)
	plumber := f.note("Plumber", &home)
	f.now = f.now.Add(time.Hour)
	f.note("Roses", &garden)
	f.now = f.now.Add(time.Hour)
	f.note("Draft", nil)
	f.tick()
	if _, err := f.s.UpdateNote(f.ctx, f.user, plumber.ID, 1, NotePatch{Color: ptr(ptr("green"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.CreateNote(f.ctx, f.other, NewNote{Title: ptr("Bob's")}); err != nil {
		t.Fatal(err)
	}

	listed := func(sel Selection, order string, limit int) []string {
		t.Helper()
		list, err := f.s.ListedNotes(f.ctx, f.user, sel, order, limit)
		if err != nil {
			t.Fatal(err)
		}
		return titles(list)
	}
	all := Selection{Mode: SelectAll}
	for _, tc := range []struct {
		name  string
		sel   Selection
		order string
		limit int
		want  []string
	}{
		{"all by updated", all, SortUpdated, 10, []string{"Plumber", "Draft", "Roses"}},
		{"all by created", all, SortCreated, 10, []string{"Draft", "Roses", "Plumber"}},
		{"all by title", all, SortTitle, 10, []string{"Draft", "Plumber", "Roses"}},
		{"limit", all, SortCreated, 2, []string{"Draft", "Roses"}},
		{"title limit", all, SortTitle, 2, []string{"Draft", "Plumber"}},
		{"uncategorized", Selection{Mode: SelectUncategorized}, SortUpdated, 10, []string{"Draft"}},
		{"with subcategories", Selection{Mode: SelectCategory, CategoryID: home, Subcategories: true}, SortCreated, 10, []string{"Roses", "Plumber"}},
		{"without subcategories", Selection{Mode: SelectCategory, CategoryID: home}, SortCreated, 10, []string{"Plumber"}},
		{"subcategory", Selection{Mode: SelectCategory, CategoryID: garden, Subcategories: true}, SortCreated, 10, []string{"Roses"}},
		{"unknown category", Selection{Mode: SelectCategory, CategoryID: "nope", Subcategories: true}, SortCreated, 10, []string{}},
	} {
		if got := listed(tc.sel, tc.order, tc.limit); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestListedNotesByTitle(t *testing.T) {
	f := newFixture(t)
	text := func(s string) json.RawMessage {
		return json.RawMessage(`{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"` + s + `"}]}]}`)
	}
	for _, n := range []NewNote{
		{Title: ptr("zoo")},
		{},
		{Title: ptr("Ärzte")},
		{Content: text("beta")},
		{},
	} {
		f.tick()
		if _, err := f.s.CreateNote(f.ctx, f.user, n); err != nil {
			t.Fatal(err)
		}
	}
	list, err := f.s.ListedNotes(f.ctx, f.user, Selection{Mode: SelectAll}, SortTitle, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, n := range list {
		if title := DisplayTitle(n); title != nil {
			got = append(got, *title)
		} else {
			got = append(got, "-")
		}
	}
	if want := []string{"Ärzte", "beta", "zoo", "-", "-"}; !reflect.DeepEqual(got, want) {
		t.Errorf("by title = %q, want %q", got, want)
	}
	if list[3].ID > list[4].ID {
		t.Errorf("untitled notes not ordered by ID: %s, %s", list[3].ID, list[4].ID)
	}
}

func TestNotesByID(t *testing.T) {
	f := newFixture(t)
	a := f.note("A", nil)
	b := f.note("B", nil)
	bobs, err := f.s.CreateNote(f.ctx, f.other, NewNote{Title: ptr("Bob's")})
	if err != nil {
		t.Fatal(err)
	}
	byID := func(ids ...string) []string {
		t.Helper()
		list, err := f.s.NotesByID(f.ctx, f.user, ids)
		if err != nil {
			t.Fatal(err)
		}
		got := titles(list)
		slices.Sort(got)
		return got
	}
	for _, tc := range []struct {
		name string
		ids  []string
		want []string
	}{
		{"both", []string{b.ID, a.ID}, []string{"A", "B"}},
		{"unknown left out", []string{a.ID, "nope"}, []string{"A"}},
		{"other user's left out", []string{bobs.ID, b.ID}, []string{"B"}},
		{"none", nil, []string{}},
	} {
		if got := byID(tc.ids...); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPinnedNotesFirst(t *testing.T) {
	f := newFixture(t)
	home := f.category("Home", nil)
	work := f.category("Work", nil)
	zoo := f.note("zoo", &home)
	f.now = f.now.Add(24 * time.Hour)
	f.note("Ärzte", &home)
	f.now = f.now.Add(24 * time.Hour)
	f.note("Plumber", &home)
	f.note("Groceries", &work)
	f.tick()
	if _, err := f.s.CreateNote(f.ctx, f.user, NewNote{Title: ptr("Old plan"), CategoryID: &home, Archived: true}); err != nil {
		t.Fatal(err)
	}
	// Pin "zoo" without a change, so that it stays the oldest change.
	if _, err := f.s.DB.Exec(`UPDATE notes SET pinned = 1 WHERE id = ?`, zoo.ID); err != nil {
		t.Fatal(err)
	}

	sel := Selection{Mode: SelectCategory, CategoryID: home}
	for _, tc := range []struct {
		name  string
		order string
		limit int
		want  []string
	}{
		{"by title", SortTitle, 10, []string{"zoo", "Ärzte", "Old plan", "Plumber"}},
		{"by created", SortCreated, 10, []string{"zoo", "Old plan", "Plumber", "Ärzte"}},
		{"pinned within the count", SortUpdated, 2, []string{"zoo", "Old plan"}},
	} {
		list, err := f.s.ListedNotes(f.ctx, f.user, sel, tc.order, tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		if got := titles(list); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %q, want %q", tc.name, got, tc.want)
		}
	}

	recent, err := f.s.RecentNotes(f.ctx, f.user, false, time.Time{}, 2)
	if err != nil || !reflect.DeepEqual(titles(recent), []string{"zoo", "Old plan"}) {
		t.Errorf("recent = %q, %v; want the pinned note first", titles(recent), err)
	}
	recent, err = f.s.RecentNotes(f.ctx, f.user, false, f.now.Add(-time.Hour), 10)
	if err != nil || slices.Contains(titles(recent), "zoo") {
		t.Errorf("recent within an hour = %q, %v; want no pinned note outside the day limit", titles(recent), err)
	}
}
