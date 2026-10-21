package dashboards

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// sec builds a section with border on and height auto.
func sec(width int, title string, children ...Section) Section {
	s := Section{Width: width, Border: true, Children: children}
	if title != "" {
		s.Title = &title
	}
	return s
}

// nested returns a chain of depth sections, one inside the other.
func nested(depth int) []Section {
	if depth == 0 {
		return nil
	}
	return []Section{sec(6, "", nested(depth-1)...)}
}

// titles renders a tree as "A(B,C),D" for comparisons.
func titles(list []Section) string {
	var out []string
	for _, s := range list {
		t := "-"
		if s.Title != nil {
			t = *s.Title
		}
		if len(s.Children) > 0 {
			t += "(" + titles(s.Children) + ")"
		}
		out = append(out, t)
	}
	return strings.Join(out, ",")
}

func (f *fixture) lookup(userSlug, dashboardSlug, viewerID string) View {
	f.t.Helper()
	v, err := f.s.Lookup(f.ctx, userSlug, dashboardSlug, viewerID)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func TestLayoutSaveAndLookup(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home", IsPublic: true})

	v := f.lookup("ann", "home", f.ann)
	if v.ID != d.ID || v.LayoutVersion != 0 || v.Sections == nil || len(v.Sections) != 0 || v.Columns != DefaultColumns {
		t.Fatalf("empty dashboard = %+v", v)
	}

	saved, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Section{
		sec(3, " Links "),
		sec(6, "Work", sec(2, "Tools"), sec(2, "Docs")),
		{Width: 4, Title: ptr("  "), Height: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 || titles(saved.Sections) != "Links,Work(Tools,Docs),-" {
		t.Fatalf("saved = %d %s", saved.Version, titles(saved.Sections))
	}
	last := saved.Sections[2]
	if last.Title != nil || last.Border || last.Height != 3 || last.Width != 4 || last.Children == nil {
		t.Fatalf("third section = %+v", last)
	}

	visitor := f.lookup("ann", "home", "")
	if visitor.ID != "" || visitor.LayoutVersion != 1 || titles(visitor.Sections) != "Links,Work(Tools,Docs),-" {
		t.Fatalf("visitor view = %+v", visitor)
	}
	data, err := json.Marshal(visitor)
	if err != nil || strings.Contains(string(data), `"id":"`+d.ID) || !strings.Contains(string(data), `"height":"auto"`) {
		t.Fatalf("visitor JSON = %s, %v", data, err)
	}

	// Renaming or changing the columns doesn't change the layout version.
	if _, err := f.s.Update(f.ctx, f.ann, d.ID, Patch{Title: ptr("Start"), DesktopColumns: ptr(8)}); err != nil {
		t.Fatal(err)
	}
	if v := f.lookup("ann", "home", f.ann); v.LayoutVersion != 1 {
		t.Fatalf("version after rename = %d, want 1", v.LayoutVersion)
	}
}

func TestLayoutKeepsKnownIDs(t *testing.T) {
	f := newFixture(t)
	home := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	work := f.create(f.ann, Fields{Title: "Work", Slug: "work"})
	first, err := f.s.SaveLayout(f.ctx, f.ann, home.ID, 0, []Section{sec(6, "A", sec(3, "B"))})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.s.SaveLayout(f.ctx, f.ann, work.ID, 0, []Section{sec(6, "W")})
	if err != nil {
		t.Fatal(err)
	}
	a, b, w := first.Sections[0], first.Sections[0].Children[0], other.Sections[0]

	// B moves to the top level and keeps its ID; A's ID is used twice; W's ID
	// belongs to another dashboard and "x" is unknown.
	b.Children, a.Children = []Section{}, []Section{}
	dup, foreign, unknown := sec(6, "Dup"), sec(6, "Foreign"), sec(6, "Unknown")
	dup.ID, foreign.ID, unknown.ID = a.ID, w.ID, "x"
	saved, err := f.s.SaveLayout(f.ctx, f.ann, home.ID, 1, []Section{b, a, dup, foreign, unknown})
	if err != nil {
		t.Fatal(err)
	}
	got := saved.Sections
	if got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("known IDs not kept: %+v", got)
	}
	for _, s := range got[2:] {
		if s.ID == a.ID || s.ID == w.ID || s.ID == "x" {
			t.Errorf("%s kept ID %q", *s.Title, s.ID)
		}
	}
	if v := f.lookup("ann", "work", f.ann); len(v.Sections) != 1 || v.Sections[0].ID != w.ID {
		t.Fatalf("other dashboard changed: %+v", v.Sections)
	}
}

func TestLayoutValidation(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, nested(MaxDepth)); err != nil {
		t.Fatalf("%d levels: %v", MaxDepth, err)
	}

	many := make([]Section, MaxWidgets+1)
	for i := range many {
		many[i] = sec(1, "")
	}
	for name, layout := range map[string][]Section{
		"six levels":      nested(MaxDepth + 1),
		"width 0":         {sec(0, "")},
		"width 13":        {sec(13, "")},
		"nested width 13": {sec(6, "", sec(13, ""))},
		"height 51":       {{Width: 6, Height: MaxHeightRows + 1}},
		"title 101":       {sec(6, strings.Repeat("ä", MaxSectionTitleLength+1))},
		"too many":        many,
	} {
		if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 1, layout); !errors.Is(err, ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
	if v := f.lookup("ann", "home", f.ann); v.LayoutVersion != 1 || len(v.Sections) != 1 {
		t.Fatalf("invalid saves changed the layout: %+v", v)
	}
}

func TestLayoutConflictAndForeign(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Section{sec(6, "First")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Section{sec(6, "Stale")}); !errors.Is(err, ErrLayoutConflict) {
		t.Fatalf("stale save: %v", err)
	}
	if _, err := f.s.SaveLayout(f.ctx, f.bob, d.ID, 1, []Section{sec(6, "Bob's")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign save: %v", err)
	}
	if v := f.lookup("ann", "home", f.ann); v.LayoutVersion != 1 || titles(v.Sections) != "First" {
		t.Fatalf("layout = %d %s, want 1 First", v.LayoutVersion, titles(v.Sections))
	}
}

func TestHeightJSON(t *testing.T) {
	var s Section
	if err := json.Unmarshal([]byte(`{"width":2,"height":4}`), &s); err != nil || s.Height != 4 {
		t.Fatalf("rows: %+v, %v", s, err)
	}
	if err := json.Unmarshal([]byte(`{"width":2,"height":"auto"}`), &s); err != nil || s.Height != 0 {
		t.Fatalf("auto: %+v, %v", s, err)
	}
	for _, bad := range []string{`"tall"`, `0`, `-1`, `1.5`} {
		if err := json.Unmarshal([]byte(`{"height":`+bad+`}`), &s); err == nil {
			t.Errorf("height %s accepted", bad)
		}
	}
}
