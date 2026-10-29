package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

func noteWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindNote, Width: 2, Config: json.RawMessage(config)}
}

func (f *fixture) note(userID string, n notes.NewNote) notes.Note {
	f.t.Helper()
	created, err := f.notes.CreateNote(f.ctx, userID, n)
	if err != nil {
		f.t.Fatal(err)
	}
	return created
}

func TestNoteConfig(t *testing.T) {
	f := newFixture(t)
	groceries := f.note(f.ann, notes.NewNote{})

	if err := f.save(noteWidget(`{"noteId":"` + groceries.ID + `","size":"L","showIfPublic":true}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"noteId":"` + groceries.ID + `","showIfPublic":true,"size":"L"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}

	for name, config := range map[string]string{
		"no note":       `{"size":"M"}`,
		"empty note":    `{"noteId":"","size":"M"}`,
		"long note":     `{"noteId":"` + strings.Repeat("a", 65) + `","size":"M"}`,
		"size XL":       `{"noteId":"` + groceries.ID + `","size":"XL"}`,
		"no size":       `{"noteId":"` + groceries.ID + `"}`,
		"unknown field": `{"noteId":"` + groceries.ID + `","size":"M","title":"x"}`,
	} {
		if err := f.save(noteWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}

	// A deleted note doesn't block saving.
	if err := f.notes.DeleteNote(f.ctx, f.ann, groceries.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.save(noteWidget(`{"noteId":"` + groceries.ID + `","size":"M"}`)); err != nil {
		t.Fatalf("save with a deleted note: %v", err)
	}
}

func TestNoteResolve(t *testing.T) {
	f := newFixture(t)
	green := "green"
	home, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	garden, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Garden", Color: &green, ParentID: &home.ID})
	if err != nil {
		t.Fatal(err)
	}
	title := "Groceries"
	content := `{"v":1,"blocks":[{"type":"list","style":"task","items":[{"id":"t1","checked":false,"content":[{"type":"text","text":"milk"}]}]}]}`
	groceries := f.note(f.ann, notes.NewNote{
		Title: &title, Color: &green, CategoryID: &garden.ID,
		Icon: &notes.Icon{Type: notes.IconSymbol, Name: "shopping-cart"}, Content: json.RawMessage(content),
	})
	plain := f.note(f.ann, notes.NewNote{})
	secretTitle := "Secret plans"
	f.note(f.ann, notes.NewNote{Title: &secretTitle})
	gone := f.note(f.ann, notes.NewNote{})
	if err := f.notes.DeleteNote(f.ctx, f.ann, gone.ID); err != nil {
		t.Fatal(err)
	}
	bobs := f.note(f.bob, notes.NewNote{})

	cfg := func(id string, public bool) string {
		raw, _ := json.Marshal(noteConfig{NoteID: id, Size: "M", ShowIfPublic: public})
		return string(raw)
	}
	if err := f.save(
		noteWidget(cfg(groceries.ID, true)),
		noteWidget(cfg(plain.ID, false)),
		noteWidget(cfg(gone.ID, true)),
		noteWidget(cfg(bobs.ID, true)),
	); err != nil {
		t.Fatal(err)
	}

	doc := mustJSON(t, groceries.Content)
	owner := f.lookup(f.ann)
	if len(owner) != 4 {
		t.Fatalf("owner sees %d widgets, want 4", len(owner))
	}
	got, _ := json.Marshal(owner[0]["data"])
	want := canonical(t, `{"id":"`+groceries.ID+`","version":1,"title":"Groceries","icon":{"type":"symbol","name":"shopping-cart"},"color":"green","content":`+doc+
		`,"pinned":false,"archived":false,"category":{"title":"Garden","icon":null,"color":"green","path":["Home","Garden"]}}`)
	if string(got) != want {
		t.Errorf("owner data =\n%s\nwant\n%s", got, want)
	}
	if got, _ := json.Marshal(owner[1]["data"]); !strings.Contains(string(got), `"category":null`) {
		t.Errorf("note without category = %s", got)
	}
	for i, name := range map[int]string{2: "deleted note", 3: "other user's note"} {
		if got, _ := json.Marshal(owner[i]["data"]); string(got) != `{"missing":true}` {
			t.Errorf("%s = %s", name, got)
		}
	}

	// Visitors get the public widget of an existing note, and only what it shows.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 1 {
			t.Fatalf("visitor %q sees %d widgets, want 1", viewer, len(visitor))
		}
		got, _ := json.Marshal(visitor[0]["data"])
		want := canonical(t, `{"title":"Groceries","icon":{"type":"symbol","name":"shopping-cart"},"color":"green","content":`+doc+
			`,"pinned":false,"archived":false,"category":{"title":"Garden","icon":null,"color":"green"}}`)
		if string(got) != want {
			t.Errorf("visitor %q data =\n%s\nwant\n%s", viewer, got, want)
		}
		raw, _ := json.Marshal(visitor)
		for _, hidden := range []string{"Home", "Secret plans", "version", `"id":"` + groceries.ID} {
			if strings.Contains(string(raw), hidden) {
				t.Errorf("visitor %q response contains %s: %s", viewer, hidden, raw)
			}
		}
	}
}
