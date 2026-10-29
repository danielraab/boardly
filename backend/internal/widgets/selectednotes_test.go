package widgets

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

func selectedNotesWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindSelectedNotes, Width: 2, Config: json.RawMessage(config)}
}

func TestSelectedNotesConfig(t *testing.T) {
	f := newFixture(t)
	if err := f.save(selectedNotesWidget(`{"noteIds":["gone","b"],"title":"  Daily ","size":"L","showIfPublic":true}`)); err != nil {
		t.Fatalf("missing notes: %v", err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"noteIds":["gone","b"],"showIfPublic":true,"size":"L","title":"Daily"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}
	hundred := make([]string, maxRecentLimit)
	for i := range hundred {
		hundred[i] = fmt.Sprintf("n%d", i)
	}
	ids, _ := json.Marshal(hundred)
	if err := f.save(selectedNotesWidget(`{"noteIds":` + string(ids) + `,"size":"M"}`)); err != nil {
		t.Fatalf("100 notes: %v", err)
	}

	tooMany, _ := json.Marshal(append(hundred, "one more"))
	for name, config := range map[string]string{
		"no notes":      `{"size":"M"}`,
		"empty notes":   `{"noteIds":[],"size":"M"}`,
		"101 notes":     `{"noteIds":` + string(tooMany) + `,"size":"M"}`,
		"empty ID":      `{"noteIds":["a",""],"size":"M"}`,
		"long ID":       `{"noteIds":["` + strings.Repeat("a", 65) + `"],"size":"M"}`,
		"duplicate":     `{"noteIds":["a","b","a"],"size":"M"}`,
		"size XL":       `{"noteIds":["a"],"size":"XL"}`,
		"unknown field": `{"noteIds":["a"],"size":"M","sort":"title"}`,
		"long title":    `{"noteIds":["a"],"title":"` + strings.Repeat("a", 201) + `","size":"M"}`,
	} {
		if err := f.save(selectedNotesWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
}

func TestSelectedNotesResolve(t *testing.T) {
	f := newFixture(t)
	home, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	garden, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Garden", ParentID: &home.ID})
	if err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }
	plumber := f.note(f.ann, notes.NewNote{Content: json.RawMessage(
		`{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"call the plumber"}]},` +
			`{"type":"list","style":"task","items":[{"id":"t1","checked":false,"content":[{"type":"text","text":"secret task"}]}]}]}`)})
	green := "green"
	roses := f.note(f.ann, notes.NewNote{Title: str("Roses"), CategoryID: &garden.ID,
		Icon: &notes.Icon{Type: notes.IconSymbol, Name: "shopping-cart"}, Color: &green})
	draft := f.note(f.ann, notes.NewNote{Title: str("Draft"), CategoryID: &home.ID})
	gone := f.note(f.ann, notes.NewNote{Title: str("Gone")})
	bobs := f.note(f.bob, notes.NewNote{Title: str("Bob's")})
	if err := f.notes.DeleteNote(f.ctx, f.ann, gone.ID); err != nil {
		t.Fatal(err)
	}

	cfg := func(public bool, ids ...string) dashboards.Widget {
		raw, _ := json.Marshal(selectedNotesConfig{NoteIDs: ids, Size: "M", ShowIfPublic: public})
		return selectedNotesWidget(string(raw))
	}
	if err := f.save(
		cfg(false, roses.ID, plumber.ID),
		cfg(true, roses.ID, gone.ID, plumber.ID, bobs.ID),
		cfg(true, draft.ID, roses.ID),
		cfg(true, gone.ID, bobs.ID),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 4 {
		t.Fatalf("owner sees %d widgets, want 4", len(owner))
	}
	for i, want := range []string{"Roses,call the plumber", "Roses,call the plumber", "Draft,Roses"} {
		if got := noteTitles(t, owner[i]["data"]); got != want {
			t.Errorf("widget %d = %s, want %s", i, got, want)
		}
	}
	data, _ := json.Marshal(owner[1]["data"])
	want := canonical(t, `{"notes":[`+
		`{"id":"`+roses.ID+`","title":"Roses","icon":{"type":"symbol","name":"shopping-cart"},"color":"green","category":{"title":"Garden","path":["Home","Garden"]}},`+
		`{"id":"`+plumber.ID+`","title":"call the plumber","icon":null,"color":null,"category":null}]}`)
	if string(data) != want {
		t.Errorf("owner data =\n%s\nwant\n%s", data, want)
	}
	// Every note with a category names it, also one directly in a top-level
	// category.
	if got, _ := json.Marshal(owner[2]["data"]); !strings.Contains(string(got), `"category":{"path":["Home"],"title":"Home"}`) {
		t.Errorf("draft doesn't name its category: %s", got)
	}
	if got, _ := json.Marshal(owner[3]["data"]); string(got) != `{"missing":true}` {
		t.Errorf("all missing data = %s", got)
	}

	// Visitors get the public widgets with at least one note, and only what
	// their rows show.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 2 {
			t.Fatalf("visitor %q sees %d widgets, want 2", viewer, len(visitor))
		}
		data, _ := json.Marshal(visitor[0]["data"])
		want := canonical(t, `{"notes":[`+
			`{"title":"Roses","icon":{"type":"symbol","name":"shopping-cart"},"color":"green","category":{"title":"Garden"}},`+
			`{"title":"call the plumber","icon":null,"color":null,"category":null}]}`)
		if string(data) != want {
			t.Errorf("visitor %q data =\n%s\nwant\n%s", viewer, data, want)
		}
		for _, hidden := range []string{"Home", "secret task", "Bob's", "Gone", "version", "Time", plumber.ID, roses.ID} {
			if strings.Contains(string(data), hidden) {
				t.Errorf("visitor %q data contains %s: %s", viewer, hidden, data)
			}
		}
	}
}
