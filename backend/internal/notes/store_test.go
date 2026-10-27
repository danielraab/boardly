package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"boardly/internal/db"
	"boardly/internal/dbtest"
	"boardly/internal/users"
)

type fixture struct {
	t     *testing.T
	db    *sql.DB
	s     *Store
	ctx   context.Context
	now   time.Time
	user  string
	other string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	ctx := context.Background()
	u := users.NewStore(database)
	ann, err := u.EnsureByEmail(ctx, database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.EnsureByEmail(ctx, database, "bob@example.org")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, db: database, ctx: ctx, now: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC), user: ann.ID, other: bob.ID}
	f.s = &Store{DB: database, Now: func() time.Time { return f.now }}
	return f
}

// tick moves the clock on by a minute.
func (f *fixture) tick() { f.now = f.now.Add(time.Minute) }

func ptr[T any](v T) *T { return &v }

func (f *fixture) category(title string, parent *string) string {
	f.t.Helper()
	c, err := f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: title, ParentID: parent})
	if err != nil {
		f.t.Fatalf("create category %q: %v", title, err)
	}
	return c.ID
}

func (f *fixture) note(title string, category *string) Note {
	f.t.Helper()
	f.tick()
	n, err := f.s.CreateNote(f.ctx, f.user, NewNote{Title: &title, CategoryID: category})
	if err != nil {
		f.t.Fatalf("create note %q: %v", title, err)
	}
	return n
}

func (f *fixture) image(userID, id string) {
	f.t.Helper()
	if _, err := f.db.Exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 1, ?)`,
		id, userID, db.FormatTime(f.now)); err != nil {
		f.t.Fatal(err)
	}
}

func titles(notes []Note) []string {
	out := []string{}
	for _, n := range notes {
		out = append(out, *n.Title)
	}
	return out
}

func TestIconsAndColors(t *testing.T) {
	f := newFixture(t)
	f.image(f.user, "mine")
	f.image(f.other, "bobs")
	for name, tc := range map[string]struct {
		icon  *Icon
		color *string
		want  error
	}{
		"no icon":           {},
		"symbol":            {icon: &Icon{Type: IconSymbol, Name: "home"}},
		"own image":         {icon: &Icon{Type: IconImage, ImageID: "mine"}, color: ptr("green")},
		"unknown symbol":    {icon: &Icon{Type: IconSymbol, Name: "nope"}, want: ErrInvalidIcon},
		"other user's":      {icon: &Icon{Type: IconImage, ImageID: "bobs"}, want: ErrInvalidIcon},
		"missing image":     {icon: &Icon{Type: IconImage, ImageID: "none"}, want: ErrInvalidIcon},
		"unknown type":      {icon: &Icon{Type: "favicon"}, want: ErrInvalidIcon},
		"symbol with image": {icon: &Icon{Type: IconSymbol, Name: "home", ImageID: "mine"}, want: ErrInvalidIcon},
		"unknown colour":    {color: ptr("magenta"), want: ErrInvalidColor},
	} {
		t.Run(name, func(t *testing.T) {
			n, err := f.s.CreateNote(f.ctx, f.user, NewNote{Icon: tc.icon, Color: tc.color})
			if !errors.Is(err, tc.want) {
				t.Fatalf("CreateNote = %v, want %v", err, tc.want)
			}
			if err == nil && !reflect.DeepEqual(n.Icon, tc.icon) {
				t.Errorf("icon = %+v, want %+v", n.Icon, tc.icon)
			}
			_, err = f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: "C", Icon: tc.icon, Color: tc.color})
			if !errors.Is(err, tc.want) {
				t.Fatalf("CreateCategory = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCategoryNestingAndMoves(t *testing.T) {
	f := newFixture(t)
	home := f.category("Home", nil)
	garden := f.category("Garden", &home)
	beds := f.category("Beds", &garden)
	projects := f.category("Projects", nil)

	if _, err := f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: "  "}); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("blank title: %v, want ErrInvalidTitle", err)
	}
	if _, err := f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: strings.Repeat("x", 201)}); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("long title: %v, want ErrInvalidTitle", err)
	}
	c, err := f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: "  Work  "})
	if err != nil || c.Title != "Work" {
		t.Errorf("trimmed title = %q, %v", c.Title, err)
	}

	for name, parent := range map[string]string{"itself": home, "child": garden, "grandchild": beds} {
		if _, err := f.s.UpdateCategory(f.ctx, f.user, home, CategoryPatch{ParentID: ptr(&parent)}); !errors.Is(err, ErrInvalidParent) {
			t.Errorf("parent %s: %v, want ErrInvalidParent", name, err)
		}
	}
	bobs, err := f.s.CreateCategory(f.ctx, f.other, NewCategory{Title: "Bob's"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.UpdateCategory(f.ctx, f.user, garden, CategoryPatch{ParentID: ptr(&bobs.ID)}); !errors.Is(err, ErrInvalidParent) {
		t.Errorf("other user's parent: %v, want ErrInvalidParent", err)
	}
	if _, err := f.s.CreateCategory(f.ctx, f.user, NewCategory{Title: "X", ParentID: &bobs.ID}); !errors.Is(err, ErrInvalidParent) {
		t.Errorf("create below other user's: %v, want ErrInvalidParent", err)
	}
	if _, err := f.s.UpdateCategory(f.ctx, f.other, home, CategoryPatch{Title: ptr("Mine")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("update other user's: %v, want ErrNotFound", err)
	}

	moved, err := f.s.UpdateCategory(f.ctx, f.user, garden, CategoryPatch{ParentID: ptr(&projects), Title: ptr("Yard"), Color: ptr(ptr("teal"))})
	if err != nil || *moved.ParentID != projects || moved.Title != "Yard" || *moved.Color != "teal" {
		t.Fatalf("move = %+v, %v", moved, err)
	}
	top, err := f.s.UpdateCategory(f.ctx, f.user, garden, CategoryPatch{ParentID: ptr[*string](nil)})
	if err != nil || top.ParentID != nil {
		t.Fatalf("move to top = %+v, %v", top, err)
	}

	all, err := f.s.Categories(f.ctx, f.user)
	if err != nil || len(all) != 5 {
		t.Fatalf("categories = %d, %v; want 5", len(all), err)
	}
}

func TestDeleteCategory(t *testing.T) {
	f := newFixture(t)
	home := f.category("Home", nil)
	garden := f.category("Garden", &home)
	work := f.category("Work", nil)
	inHome := f.note("Groceries", &home)
	inGarden := f.note("Seeds", &garden)
	f.note("Sprint", &work)

	f.tick()
	if err := f.s.DeleteCategory(f.ctx, f.other, home); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete other user's: %v, want ErrNotFound", err)
	}
	if err := f.s.DeleteCategory(f.ctx, f.user, home); err != nil {
		t.Fatal(err)
	}
	n, err := f.s.Note(f.ctx, f.user, inHome.ID)
	if err != nil || n.CategoryID != nil || !n.UpdatedAt.Equal(inHome.UpdatedAt) || n.Version != 1 {
		t.Errorf("note of deleted category = %+v, %v; want uncategorized and unchanged", n, err)
	}
	if n, _ := f.s.Note(f.ctx, f.user, inGarden.ID); *n.CategoryID != garden {
		t.Errorf("note in subcategory lost its category")
	}
	all, _ := f.s.Categories(f.ctx, f.user)
	for _, c := range all {
		if c.ID == garden && c.ParentID != nil {
			t.Errorf("Garden parent = %v, want top level", *c.ParentID)
		}
	}
	if len(all) != 2 {
		t.Errorf("categories = %d, want 2", len(all))
	}

	// A subcategory of a nested category moves to its grandparent.
	beds := f.category("Beds", &garden)
	mid := f.category("Mid", &beds)
	if err := f.s.DeleteCategory(f.ctx, f.user, beds); err != nil {
		t.Fatal(err)
	}
	all, _ = f.s.Categories(f.ctx, f.user)
	for _, c := range all {
		if c.ID == mid && (c.ParentID == nil || *c.ParentID != garden) {
			t.Errorf("Mid parent = %v, want Garden", c.ParentID)
		}
	}
}

func TestNotesOrderAndVersions(t *testing.T) {
	f := newFixture(t)
	groceries := f.note("Groceries", nil)
	sprint := f.note("Sprint plan", nil)
	if groceries.Version != 1 || groceries.Content.V != 1 || len(groceries.Content.Blocks) != 0 {
		t.Errorf("new note = %+v", groceries)
	}
	list, err := f.s.Notes(f.ctx, f.user)
	if err != nil || !reflect.DeepEqual(titles(list), []string{"Sprint plan", "Groceries"}) {
		t.Fatalf("notes = %q, %v", titles(list), err)
	}

	f.tick()
	edited, err := f.s.UpdateNote(f.ctx, f.user, groceries.ID, 1, NotePatch{Color: ptr(ptr("yellow"))})
	if err != nil || edited.Version != 2 || *edited.Color != "yellow" || !edited.UpdatedAt.Equal(f.now) {
		t.Fatalf("update = %+v, %v", edited, err)
	}
	list, _ = f.s.Notes(f.ctx, f.user)
	if !reflect.DeepEqual(titles(list), []string{"Groceries", "Sprint plan"}) {
		t.Errorf("after edit notes = %q", titles(list))
	}

	if _, err := f.s.UpdateNote(f.ctx, f.user, groceries.ID, 1, NotePatch{Title: ptr(ptr("Stale"))}); !errors.Is(err, ErrVersionConflict) {
		t.Errorf("stale version: %v, want ErrVersionConflict", err)
	}
	if n, _ := f.s.Note(f.ctx, f.user, groceries.ID); *n.Title != "Groceries" || n.Version != 2 {
		t.Errorf("note changed by a stale update: %+v", n)
	}
	same, err := f.s.UpdateNote(f.ctx, f.user, groceries.ID, 2, NotePatch{})
	if err != nil || same.Version != 2 {
		t.Errorf("empty patch = %+v, %v; want unchanged", same, err)
	}

	cleared, err := f.s.UpdateNote(f.ctx, f.user, sprint.ID, 1, NotePatch{Title: ptr(ptr("   "))})
	if err != nil || cleared.Title != nil {
		t.Errorf("blank title = %v, %v; want none", cleared.Title, err)
	}
	if _, err := f.s.UpdateNote(f.ctx, f.user, sprint.ID, 2, NotePatch{Title: ptr(ptr(strings.Repeat("x", 201)))}); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("long title: %v, want ErrInvalidTitle", err)
	}
	if _, err := f.s.UpdateNote(f.ctx, f.user, sprint.ID, 2, NotePatch{Content: json.RawMessage(`{"v":1,"blocks":[{"type":"iframe"}]}`)}); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("invalid content: %v, want ErrInvalidDocument", err)
	}

	content, err := f.s.UpdateNote(f.ctx, f.user, sprint.ID, 2, NotePatch{Content: json.RawMessage(
		`{"v":1,"blocks":[{"type":"list","style":"task","items":[{"content":[{"type":"text","text":"review PR"}]}]}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	if ids := taskIDs(content.Content); len(ids) != 1 || ids[0] == "" {
		t.Errorf("task IDs = %q, want one assigned", ids)
	}
}

func TestNotesOfOtherUsers(t *testing.T) {
	f := newFixture(t)
	n := f.note("Mine", nil)
	bobs, err := f.s.CreateCategory(f.ctx, f.other, NewCategory{Title: "Bob's"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Note(f.ctx, f.other, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("get: %v, want ErrNotFound", err)
	}
	if _, err := f.s.UpdateNote(f.ctx, f.other, n.ID, 1, NotePatch{Title: ptr(ptr("x"))}); !errors.Is(err, ErrNotFound) {
		t.Errorf("update: %v, want ErrNotFound", err)
	}
	if err := f.s.DeleteNote(f.ctx, f.other, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete: %v, want ErrNotFound", err)
	}
	if _, err := f.s.CreateNote(f.ctx, f.user, NewNote{CategoryID: &bobs.ID}); !errors.Is(err, ErrInvalidCategory) {
		t.Errorf("create with other user's category: %v, want ErrInvalidCategory", err)
	}
	if _, err := f.s.UpdateNote(f.ctx, f.user, n.ID, 1, NotePatch{CategoryID: ptr(&bobs.ID)}); !errors.Is(err, ErrInvalidCategory) {
		t.Errorf("update with other user's category: %v, want ErrInvalidCategory", err)
	}
	if list, _ := f.s.Notes(f.ctx, f.other); len(list) != 0 {
		t.Errorf("Bob sees %d notes", len(list))
	}
	if err := f.s.DeleteNote(f.ctx, f.user, n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Note(f.ctx, f.user, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted note: %v, want ErrNotFound", err)
	}
}
