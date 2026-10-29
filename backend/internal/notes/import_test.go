package notes

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// importFile is a small import tree: a note at the top, and a note in
// "Recipes/Pasta" with a task.
func importFile() ImportTree {
	created := time.Date(2026, 9, 25, 19, 48, 21, 0, time.UTC)
	return ImportTree{Root: ImportCategory{
		Notes: []ImportNote{{Title: ptr("Door"), Content: FromText("code 1234")}},
		Categories: []ImportCategory{{Name: "recipes", Categories: []ImportCategory{{
			Name:  "Pasta",
			Notes: []ImportNote{{Title: ptr("Basil"), Content: FromMarkdown("- [ ] buy basil"), CreatedAt: created}},
		}}}},
	}, TooLarge: 2}
}

// outline lists the user's categories by path and the notes in each, sorted.
func (f *fixture) outline() string {
	f.t.Helper()
	categories, err := f.s.Categories(f.ctx, f.user)
	if err != nil {
		f.t.Fatal(err)
	}
	byID := map[string]Category{}
	for _, c := range categories {
		byID[c.ID] = c
	}
	path := func(id *string) string {
		var parts []string
		for id != nil {
			c := byID[*id]
			parts = append([]string{c.Title}, parts...)
			id = c.ParentID
		}
		return strings.Join(parts, "/")
	}
	var lines []string
	for _, c := range categories {
		lines = append(lines, path(&c.ID)+"/")
	}
	ns, err := f.s.Notes(f.ctx, f.user)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, n := range ns {
		title := "-"
		if n.Title != nil {
			title = *n.Title
		}
		lines = append(lines, path(n.CategoryID)+"/"+title)
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

func TestImportIntoTopLevel(t *testing.T) {
	f := newFixture(t)
	report, err := f.s.ImportTx(f.ctx, f.db, f.user, nil, importFile(), true)
	if err != nil {
		t.Fatal(err)
	}
	if want := (ImportReport{Notes: 2, Categories: 2, TooLargeNotes: 2}); report != want {
		t.Errorf("report = %+v, want %+v", report, want)
	}
	if got, want := f.outline(), "/Door\nrecipes/\nrecipes/Pasta/\nrecipes/Pasta/Basil"; got != want {
		t.Errorf("outline =\n%s\nwant\n%s", got, want)
	}

	ns, _ := f.s.Notes(f.ctx, f.user)
	for _, n := range ns {
		switch *n.Title {
		case "Basil":
			want := time.Date(2026, 9, 25, 19, 48, 21, 0, time.UTC)
			if !n.CreatedAt.Equal(want) || !n.UpdatedAt.Equal(want) || n.Version != 1 {
				t.Errorf("Basil: created %v, updated %v, version %d", n.CreatedAt, n.UpdatedAt, n.Version)
			}
			items := *n.Content.Blocks[0].Items
			if items[0].ID == "" {
				t.Error("Basil's task has no ID")
			}
		case "Door":
			if !n.CreatedAt.Equal(f.now) {
				t.Errorf("Door created %v, want the import time %v", n.CreatedAt, f.now)
			}
		}
	}
}

func TestImportIntoCategoryReusesCategories(t *testing.T) {
	f := newFixture(t)
	imported := f.category("Imported", nil)
	f.category("Recipes", &imported)
	f.category("Elsewhere", nil)

	report, err := f.s.ImportTx(f.ctx, f.db, f.user, &imported, importFile(), true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Categories != 1 || report.Notes != 2 {
		t.Errorf("report = %+v, want 1 category (Pasta) and 2 notes", report)
	}
	want := "Elsewhere/\nImported/\nImported/Door\nImported/Recipes/\nImported/Recipes/Pasta/\nImported/Recipes/Pasta/Basil"
	if got := f.outline(); got != want {
		t.Errorf("outline =\n%s\nwant\n%s", got, want)
	}
}

func TestImportInvalidCategory(t *testing.T) {
	f := newFixture(t)
	bobs, err := f.s.CreateCategory(f.ctx, f.other, NewCategory{Title: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{bobs.ID, "nope"} {
		if _, err := f.s.ImportTx(f.ctx, f.db, f.user, &id, importFile(), true); !errors.Is(err, ErrInvalidCategory) {
			t.Errorf("%s: err = %v, want ErrInvalidCategory", id, err)
		}
	}
	if got := f.outline(); got != "" {
		t.Errorf("outline = %q, want nothing imported", got)
	}
}

func TestImportDuplicateNotes(t *testing.T) {
	f := newFixture(t)
	// An existing note with the same title and content, in another category.
	other := f.category("Other", nil)
	raw := `{"v":1,"blocks":[{"type":"list","style":"task","items":[{"id":"x1","content":[{"type":"text","text":"buy basil"}]}]}]}`
	if _, err := f.s.CreateNote(f.ctx, f.user, NewNote{Title: ptr("Basil"), CategoryID: &other, Content: []byte(raw)}); err != nil {
		t.Fatal(err)
	}
	// Same text with another title is not a duplicate; a repeat within the
	// file is.
	file := importFile()
	file.Root.Notes = append(file.Root.Notes,
		ImportNote{Title: ptr("Gate"), Content: FromText("code 1234")},
		ImportNote{Title: ptr("Door"), Content: FromText("code 1234")},
	)

	report, err := f.s.ImportTx(f.ctx, f.db, f.user, nil, file, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Notes != 2 || report.DuplicateNotes != 2 {
		t.Errorf("report = %+v, want 2 notes (Door, Gate) and 2 duplicates", report)
	}
	report, err = f.s.ImportTx(f.ctx, f.db, f.user, nil, file, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Notes != 0 || report.DuplicateNotes != 4 || report.Categories != 0 {
		t.Errorf("second import: report = %+v, want everything a duplicate", report)
	}

	report, err = f.s.ImportTx(f.ctx, f.db, f.user, nil, file, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Notes != 4 || report.DuplicateNotes != 0 {
		t.Errorf("duplicates allowed: report = %+v, want 4 notes", report)
	}
	if got, want := strings.Count(f.outline(), "recipes/Pasta/Basil"), 1; got != want {
		t.Errorf("Basil in recipes/Pasta %d times, want %d:\n%s", got, want, f.outline())
	}
}

func TestImportArchivedNotes(t *testing.T) {
	f := newFixture(t)
	created := time.Date(2026, 9, 25, 19, 48, 21, 0, time.UTC)
	existing := f.note("Old plan", nil)
	tree := ImportTree{Root: ImportCategory{Notes: []ImportNote{
		{Title: ptr("Archived"), Content: FromText("old plan"), CreatedAt: created, Archived: true},
		{Title: ptr("Old plan"), Content: existing.Content, Archived: true},
	}}}
	report, err := f.s.ImportTx(f.ctx, f.db, f.user, nil, tree, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := (ImportReport{Notes: 1, DuplicateNotes: 1}); report != want {
		t.Errorf("report = %+v, want %+v", report, want)
	}
	ns, err := f.s.Notes(f.ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range ns {
		switch *n.Title {
		case "Archived":
			if !n.Archived || n.Pinned || n.Version != 1 || !n.CreatedAt.Equal(created) || !n.UpdatedAt.Equal(created) {
				t.Errorf("imported note = %+v, want archived, not pinned, version 1, created and changed at %v", n, created)
			}
		case "Old plan":
			// The skipped duplicate leaves the existing note as it was.
			if n.Archived || n.Version != 1 {
				t.Errorf("existing note = %+v, want unchanged", n)
			}
		}
	}
}
