package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/notes"
)

func recentNotesWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindRecentNotes, Width: 2, Config: json.RawMessage(config)}
}

func TestRecentNotesConfig(t *testing.T) {
	f := newFixture(t)
	if err := f.save(recentNotesWidget(`{"sortBy":"updated","limit":5,"days":7,"title":"  New ","size":"L","showIfPublic":true}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"days":7,"limit":5,"showIfPublic":true,"size":"L","sortBy":"updated","title":"New"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}
	if err := f.save(recentNotesWidget(`{"sortBy":"created","limit":100,"days":null,"size":"M"}`)); err != nil {
		t.Fatalf("no days: %v", err)
	}

	for name, config := range map[string]string{
		"no sort":       `{"limit":10,"size":"M"}`,
		"sort title":    `{"sortBy":"title","limit":10,"size":"M"}`,
		"no limit":      `{"sortBy":"created","size":"M"}`,
		"limit 0":       `{"sortBy":"created","limit":0,"size":"M"}`,
		"limit 101":     `{"sortBy":"created","limit":101,"size":"M"}`,
		"days 0":        `{"sortBy":"created","limit":10,"days":0,"size":"M"}`,
		"days 366":      `{"sortBy":"created","limit":10,"days":366,"size":"M"}`,
		"size XL":       `{"sortBy":"created","limit":10,"size":"XL"}`,
		"unknown field": `{"sortBy":"created","limit":10,"size":"M","categoryId":"x"}`,
		"long title":    `{"sortBy":"created","limit":10,"title":"` + strings.Repeat("a", 201) + `","size":"M"}`,
	} {
		if err := f.save(recentNotesWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
}

// noteTitles returns the titles of a recent notes widget's data, joined by
// commas.
func noteTitles(t *testing.T, data any) string {
	t.Helper()
	raw, _ := json.Marshal(data)
	var d struct {
		Notes []struct{ Title string } `json:"notes"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, n := range d.Notes {
		out = append(out, n.Title)
	}
	return strings.Join(out, ",")
}

func TestRecentNotesResolve(t *testing.T) {
	f := newFixture(t)
	fixed := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	f.notes.Now = func() time.Time { return fixed }
	day := 24 * time.Hour
	times := func(id string, created, updated time.Duration) {
		if _, err := f.notes.DB.Exec(`UPDATE notes SET created_at = ?, updated_at = ? WHERE id = ?`,
			db.FormatTime(fixed.Add(-created)), db.FormatTime(fixed.Add(-updated)), id); err != nil {
			t.Fatal(err)
		}
	}
	home, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	garden, err := f.notes.CreateCategory(f.ctx, f.ann, notes.NewCategory{Title: "Garden", ParentID: &home.ID})
	if err != nil {
		t.Fatal(err)
	}
	title := "Groceries"
	groceries := f.note(f.ann, notes.NewNote{Title: &title, CategoryID: &garden.ID,
		Icon: &notes.Icon{Type: notes.IconSymbol, Name: "shopping-cart"}})
	times(groceries.ID, 30*day, time.Hour)
	plumber := f.note(f.ann, notes.NewNote{Content: json.RawMessage(
		`{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"call the plumber"}]},` +
			`{"type":"list","style":"task","items":[{"id":"t1","checked":false,"content":[{"type":"text","text":"secret task"}]}]}]}`)})
	times(plumber.ID, day, day)
	sprintTitle := "Sprint"
	sprint := f.note(f.ann, notes.NewNote{Title: &sprintTitle})
	times(sprint.ID, 3*day, 3*day)
	bobsTitle := "Bob's"
	f.note(f.bob, notes.NewNote{Title: &bobsTitle})

	cfg := func(sortBy string, limit, days int, public bool) string {
		c := recentNotesConfig{SortBy: sortBy, Limit: limit, Size: "M", ShowIfPublic: public}
		if days > 0 {
			c.Days = &days
		}
		raw, _ := json.Marshal(c)
		return string(raw)
	}
	if err := f.save(
		recentNotesWidget(cfg(sortByCreated, 10, 0, false)),
		recentNotesWidget(cfg(sortByUpdated, 10, 0, true)),
		recentNotesWidget(cfg(sortByCreated, 2, 0, true)),
		recentNotesWidget(cfg(sortByCreated, 10, 7, true)),
		recentNotesWidget(cfg(sortByUpdated, 10, 7, true)),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 5 {
		t.Fatalf("owner sees %d widgets, want 5", len(owner))
	}
	for i, want := range []string{
		"call the plumber,Sprint,Groceries",
		"Groceries,call the plumber,Sprint",
		"call the plumber,Sprint",
		"call the plumber,Sprint",
		"Groceries,call the plumber,Sprint",
	} {
		if got := noteTitles(t, owner[i]["data"]); got != want {
			t.Errorf("widget %d = %s, want %s", i, got, want)
		}
	}
	data, _ := json.Marshal(owner[1]["data"])
	want := canonical(t, `{"notes":[`+
		`{"id":"`+groceries.ID+`","title":"Groceries","icon":{"type":"symbol","name":"shopping-cart"},"time":"2026-10-27T11:00:00Z","category":{"title":"Garden","path":["Home","Garden"]}},`+
		`{"id":"`+plumber.ID+`","title":"call the plumber","icon":null,"time":"2026-10-26T12:00:00Z","category":null},`+
		`{"id":"`+sprint.ID+`","title":"Sprint","icon":null,"time":"2026-10-24T12:00:00Z","category":null}]}`)
	if string(data) != want {
		t.Errorf("by updated =\n%s\nwant\n%s", data, want)
	}
	if got, _ := json.Marshal(owner[3]["data"]); !strings.Contains(string(got), `"time":"2026-10-26T12:00:00Z"`) {
		t.Errorf("by created has no creation time: %s", got)
	}

	// Visitors get the public widgets, and only what their rows show.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 4 {
			t.Fatalf("visitor %q sees %d widgets, want 4", viewer, len(visitor))
		}
		data, _ := json.Marshal(visitor[0]["data"])
		want := canonical(t, `{"notes":[`+
			`{"title":"Groceries","icon":{"type":"symbol","name":"shopping-cart"},"time":"2026-10-27T11:00:00Z","category":{"title":"Garden"}},`+
			`{"title":"call the plumber","icon":null,"time":"2026-10-26T12:00:00Z","category":null},`+
			`{"title":"Sprint","icon":null,"time":"2026-10-24T12:00:00Z","category":null}]}`)
		if string(data) != want {
			t.Errorf("visitor %q data =\n%s\nwant\n%s", viewer, data, want)
		}
		raw, _ := json.Marshal(visitor)
		for _, hidden := range []string{"Home", "secret task", "Bob's", "version", groceries.ID, plumber.ID} {
			if strings.Contains(string(raw), hidden) {
				t.Errorf("visitor %q response contains %s: %s", viewer, hidden, raw)
			}
		}
	}
}
