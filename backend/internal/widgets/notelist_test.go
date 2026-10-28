package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

func noteListWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindNoteList, Width: 2, Config: json.RawMessage(config)}
}

func TestNoteListConfig(t *testing.T) {
	f := newFixture(t)
	if err := f.save(noteListWidget(`{"category":"category","categoryId":"gone","subcategories":true,"sort":"title","limit":5,"title":"  Work ","size":"L","showIfPublic":true}`)); err != nil {
		t.Fatalf("missing category: %v", err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"category":"category","categoryId":"gone","limit":5,"showIfPublic":true,"size":"L","sort":"title","subcategories":true,"title":"Work"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}
	if err := f.save(noteListWidget(`{"category":"all","categoryId":"x","sort":"updated","limit":100,"size":"M"}`)); err != nil {
		t.Fatalf("all: %v", err)
	}
	if got, _ := json.Marshal(f.lookup(f.ann)[0]["config"]); !strings.Contains(string(got), `"categoryId":null`) {
		t.Errorf("all keeps its category ID: %s", got)
	}

	for name, config := range map[string]string{
		"no category":    `{"sort":"updated","limit":10,"size":"M"}`,
		"unknown mode":   `{"category":"some","sort":"updated","limit":10,"size":"M"}`,
		"no category ID": `{"category":"category","sort":"updated","limit":10,"size":"M"}`,
		"long ID":        `{"category":"category","categoryId":"` + strings.Repeat("a", 65) + `","sort":"updated","limit":10,"size":"M"}`,
		"no sort":        `{"category":"all","limit":10,"size":"M"}`,
		"sort oldest":    `{"category":"all","sort":"oldest","limit":10,"size":"M"}`,
		"no limit":       `{"category":"all","sort":"updated","size":"M"}`,
		"limit 0":        `{"category":"all","sort":"updated","limit":0,"size":"M"}`,
		"limit 101":      `{"category":"all","sort":"updated","limit":101,"size":"M"}`,
		"size XL":        `{"category":"all","sort":"updated","limit":10,"size":"XL"}`,
		"unknown field":  `{"category":"all","sort":"updated","limit":10,"size":"M","days":3}`,
		"long title":     `{"category":"all","sort":"updated","limit":10,"title":"` + strings.Repeat("a", 201) + `","size":"M"}`,
	} {
		if err := f.save(noteListWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
}

func TestNoteListResolve(t *testing.T) {
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
	plumber := f.note(f.ann, notes.NewNote{CategoryID: &home.ID, Content: json.RawMessage(
		`{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"call the plumber"}]},` +
			`{"type":"list","style":"task","items":[{"id":"t1","checked":false,"content":[{"type":"text","text":"secret task"}]}]}]}`)})
	roses := f.note(f.ann, notes.NewNote{Title: str("Roses"), CategoryID: &garden.ID,
		Icon: &notes.Icon{Type: notes.IconSymbol, Name: "shopping-cart"}})
	f.note(f.ann, notes.NewNote{Title: str("Draft")})
	f.note(f.bob, notes.NewNote{Title: str("Bob's")})

	cfg := func(mode notes.SelectionMode, id *string, sub bool, sort string, public bool) dashboards.Widget {
		raw, _ := json.Marshal(noteListConfig{Category: mode, CategoryID: id, Subcategories: sub,
			Sort: sort, Limit: 10, Size: "M", ShowIfPublic: public})
		return noteListWidget(string(raw))
	}
	gone := "gone"
	if err := f.save(
		cfg(notes.SelectCategory, &home.ID, true, notes.SortTitle, false),
		cfg(notes.SelectCategory, &home.ID, true, notes.SortTitle, true),
		cfg(notes.SelectCategory, &home.ID, false, notes.SortTitle, true),
		cfg(notes.SelectUncategorized, nil, true, notes.SortTitle, true),
		cfg(notes.SelectAll, nil, true, notes.SortTitle, true),
		cfg(notes.SelectCategory, &gone, true, notes.SortTitle, true),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 6 {
		t.Fatalf("owner sees %d widgets, want 6", len(owner))
	}
	for i, want := range []string{
		"call the plumber,Roses",
		"call the plumber,Roses",
		"call the plumber",
		"Draft",
		"call the plumber,Draft,Roses",
	} {
		if got := noteTitles(t, owner[i]["data"]); got != want {
			t.Errorf("widget %d = %s, want %s", i, got, want)
		}
	}
	data, _ := json.Marshal(owner[1]["data"])
	want := canonical(t, `{"notes":[`+
		`{"id":"`+plumber.ID+`","title":"call the plumber","icon":null,"category":null},`+
		`{"id":"`+roses.ID+`","title":"Roses","icon":{"type":"symbol","name":"shopping-cart"},"category":{"title":"Garden","path":["Home","Garden"]}}]}`)
	if string(data) != want {
		t.Errorf("home =\n%s\nwant\n%s", data, want)
	}
	if got, _ := json.Marshal(owner[4]["data"]); !strings.Contains(string(got), `"category":{"path":["Home"],"title":"Home"}`) {
		t.Errorf("all notes doesn't name the category of a note in Home: %s", got)
	}
	if got, _ := json.Marshal(owner[5]["data"]); string(got) != `{"missing":true}` {
		t.Errorf("missing category data = %s", got)
	}

	// Visitors get the public widgets with existing categories, and only
	// what their rows show.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 4 {
			t.Fatalf("visitor %q sees %d widgets, want 4", viewer, len(visitor))
		}
		data, _ := json.Marshal(visitor[0]["data"])
		want := canonical(t, `{"notes":[`+
			`{"title":"call the plumber","icon":null,"category":null},`+
			`{"title":"Roses","icon":{"type":"symbol","name":"shopping-cart"},"category":{"title":"Garden"}}]}`)
		if string(data) != want {
			t.Errorf("visitor %q data =\n%s\nwant\n%s", viewer, data, want)
		}
		raw, _ := json.Marshal(visitor[0]["data"])
		for _, hidden := range []string{"Home", "secret task", "Bob's", "version", "Time", plumber.ID, roses.ID, home.ID} {
			if strings.Contains(string(raw), hidden) {
				t.Errorf("visitor %q data contains %s: %s", viewer, hidden, raw)
			}
		}
	}
}
