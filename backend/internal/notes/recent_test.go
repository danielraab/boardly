package notes

import (
	"reflect"
	"testing"
	"time"
)

func TestRecentNotes(t *testing.T) {
	f := newFixture(t)
	groceries := f.note("Groceries", nil)
	f.now = f.now.Add(24 * time.Hour)
	f.note("Plumber", nil)
	f.now = f.now.Add(24 * time.Hour)
	f.note("Sprint", nil)
	f.tick()
	if _, err := f.s.UpdateNote(f.ctx, f.user, groceries.ID, 1, NotePatch{Color: ptr(ptr("green"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.CreateNote(f.ctx, f.other, NewNote{Title: ptr("Bob's")}); err != nil {
		t.Fatal(err)
	}

	recent := func(byUpdated bool, since time.Time, limit int) []string {
		t.Helper()
		list, err := f.s.RecentNotes(f.ctx, f.user, byUpdated, since, limit)
		if err != nil {
			t.Fatal(err)
		}
		return titles(list)
	}
	if got := recent(false, time.Time{}, 10); !reflect.DeepEqual(got, []string{"Sprint", "Plumber", "Groceries"}) {
		t.Errorf("by created = %q", got)
	}
	if got := recent(true, time.Time{}, 10); !reflect.DeepEqual(got, []string{"Groceries", "Sprint", "Plumber"}) {
		t.Errorf("by updated = %q", got)
	}
	if got := recent(false, time.Time{}, 2); !reflect.DeepEqual(got, []string{"Sprint", "Plumber"}) {
		t.Errorf("limit 2 = %q", got)
	}
	since := f.now.Add(-36 * time.Hour)
	if got := recent(false, since, 10); !reflect.DeepEqual(got, []string{"Sprint", "Plumber"}) {
		t.Errorf("created since = %q", got)
	}
	if got := recent(true, f.now.Add(-30*time.Second), 10); !reflect.DeepEqual(got, []string{"Groceries"}) {
		t.Errorf("updated since = %q", got)
	}
	if list, _ := f.s.RecentNotes(f.ctx, f.other, false, time.Time{}, 10); !reflect.DeepEqual(titles(list), []string{"Bob's"}) {
		t.Errorf("Bob's recent = %q", titles(list))
	}
}

func TestRecentNotesTieBreak(t *testing.T) {
	f := newFixture(t)
	a, err := f.s.CreateNote(f.ctx, f.user, NewNote{Title: ptr("A")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.s.CreateNote(f.ctx, f.user, NewNote{Title: ptr("B")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"B", "A"}
	if a.ID > b.ID {
		want = []string{"A", "B"}
	}
	list, err := f.s.RecentNotes(f.ctx, f.user, false, time.Time{}, 10)
	if err != nil || !reflect.DeepEqual(titles(list), want) {
		t.Errorf("same time = %q, %v; want %q", titles(list), err, want)
	}
}

func TestDisplayTitle(t *testing.T) {
	text := func(s string) Inline { return Inline{Type: InlineText, Text: &s} }
	p := func(s string) Block {
		c := []Inline{}
		if s != "" {
			c = append(c, text(s))
		}
		return Block{Type: BlockParagraph, Content: &c}
	}
	doc := func(blocks ...Block) Doc { return Doc{V: 1, Blocks: blocks} }
	shown := func(title *string, d Doc) string {
		got := DisplayTitle(Note{Title: title, Content: d})
		if got == nil {
			return "<nil>"
		}
		return *got
	}
	broken := []Inline{text("first"), {Type: InlineBreak}, text("second")}
	link := []Inline{{Type: InlineLink, URL: "https://example.org", Content: &[]Inline{text("docs")}}}
	nested := Block{Type: BlockList, Style: ListBullet, Items: &[]Item{{Content: []Inline{}, Children: &Block{
		Type: BlockList, Style: ListBullet, Items: &[]Item{{Content: []Inline{text("deep")}}}}}}}
	tests := []struct {
		name  string
		title *string
		doc   Doc
		want  string
	}{
		{"title", ptr("Groceries"), doc(p("milk")), "Groceries"},
		{"first line", nil, doc(Block{Type: BlockDivider}, p(""), p("  call the plumber tomorrow ")), "call the plumber tomorrow"},
		{"line break", nil, doc(Block{Type: BlockParagraph, Content: &broken}), "first"},
		{"task list link", nil, doc(Block{Type: BlockList, Style: ListTask, Items: &[]Item{{ID: "a", Checked: ptr(false), Content: link}}}), "docs"},
		{"code", nil, doc(Block{Type: BlockCode, Text: ptr("\nmake check")}), "make check"},
		{"nested list", nil, doc(nested), "deep"},
		{"long", nil, doc(p(repeat("x", 100))), repeat("x", 80) + "…"},
		{"long multibyte", nil, doc(p(repeat("ä", 81))), repeat("ä", 80) + "…"},
		{"empty", nil, doc(p("")), "<nil>"},
	}
	for _, tt := range tests {
		if got := shown(tt.title, tt.doc); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}
