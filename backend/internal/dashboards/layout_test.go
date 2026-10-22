package dashboards

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// sec builds a section with border on and height auto.
func sec(width int, title string, children ...Widget) Widget {
	s := Widget{Kind: KindSection, Width: width, Border: true, Children: children}
	if title != "" {
		s.Title = &title
	}
	return s
}

// nested returns a chain of depth sections, one inside the other.
func nested(depth int) []Widget {
	if depth == 0 {
		return nil
	}
	return []Widget{sec(6, "", nested(depth-1)...)}
}

// titles renders a tree as "A(B,C),D" for comparisons.
func titles(list []Widget) string {
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

	saved, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Widget{
		sec(3, " Links "),
		sec(6, "Work", sec(2, "Tools"), sec(2, "Docs")),
		{Kind: KindSection, Width: 4, Title: ptr("  "), Height: 3},
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
	first, err := f.s.SaveLayout(f.ctx, f.ann, home.ID, 0, []Widget{sec(6, "A", sec(3, "B"))})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.s.SaveLayout(f.ctx, f.ann, work.ID, 0, []Widget{sec(6, "W")})
	if err != nil {
		t.Fatal(err)
	}
	a, b, w := first.Sections[0], first.Sections[0].Children[0], other.Sections[0]

	// B moves to the top level and keeps its ID; A's ID is used twice; W's ID
	// belongs to another dashboard and "x" is unknown.
	b.Children, a.Children = []Widget{}, []Widget{}
	dup, foreign, unknown := sec(6, "Dup"), sec(6, "Foreign"), sec(6, "Unknown")
	dup.ID, foreign.ID, unknown.ID = a.ID, w.ID, "x"
	saved, err := f.s.SaveLayout(f.ctx, f.ann, home.ID, 1, []Widget{b, a, dup, foreign, unknown})
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

	many := make([]Widget, MaxWidgets+1)
	for i := range many {
		many[i] = sec(1, "")
	}
	for name, layout := range map[string][]Widget{
		"six levels":      nested(MaxDepth + 1),
		"width 0":         {sec(0, "")},
		"width 13":        {sec(13, "")},
		"nested width 13": {sec(6, "", sec(13, ""))},
		"height 51":       {{Kind: KindSection, Width: 6, Height: MaxHeightRows + 1}},
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
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Widget{sec(6, "First")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Widget{sec(6, "Stale")}); !errors.Is(err, ErrLayoutConflict) {
		t.Fatalf("stale save: %v", err)
	}
	if _, err := f.s.SaveLayout(f.ctx, f.bob, d.ID, 1, []Widget{sec(6, "Bob's")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign save: %v", err)
	}
	if v := f.lookup("ann", "home", f.ann); v.LayoutVersion != 1 || titles(v.Sections) != "First" {
		t.Fatalf("layout = %d %s, want 1 First", v.LayoutVersion, titles(v.Sections))
	}
}

func TestHeightJSON(t *testing.T) {
	var s Widget
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

// noteConfig is the config of the test-only kind "note".
type noteConfig struct {
	Text    string `json:"text"`
	Private bool   `json:"private"`
}

// withNotes registers "note": the text is trimmed and must not be empty, the
// data is the text in brackets, and private notes are hidden from visitors.
func withNotes(f *fixture) {
	f.s.Kinds["note"] = Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c noteConfig
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&c); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidLayout, err)
			}
			if c.Text = strings.TrimSpace(c.Text); c.Text == "" {
				return nil, fmt.Errorf("%w: empty note", ErrInvalidLayout)
			}
			return json.Marshal(c)
		},
		Resolve: func(_ context.Context, viewer Viewer, raw json.RawMessage) (any, bool, error) {
			var c noteConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if c.Private && !viewer.IsOwner {
				return nil, false, nil
			}
			return "[" + c.Text + "]", true, nil
		},
	}
}

// note builds a note widget with height auto.
func note(width int, config string) Widget {
	return Widget{Kind: "note", Width: width, Config: json.RawMessage(config)}
}

func TestContentWidgets(t *testing.T) {
	f := newFixture(t)
	withNotes(f)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home", IsPublic: true})

	n := note(2, `{"text":"  hi "}`)
	n.Height, n.Data = 3, "ignored"
	saved, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Widget{
		sec(6, "Work", n, sec(2, "Tools"), note(2, `{"text":"secret","private":true}`)),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := saved.Sections[0].Children
	if len(got) != 3 || got[0].Kind != "note" || got[0].Height != 3 || got[0].Width != 2 ||
		string(got[0].Config) != `{"text":"hi","private":false}` || got[0].Data != "[hi]" || got[0].Children != nil {
		t.Fatalf("saved children = %+v", got)
	}
	var stored string
	if err := f.s.DB.QueryRow(`SELECT config FROM dashboard_widgets WHERE id = ?`, got[0].ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "ignored") {
		t.Fatalf("data was stored: %s", stored)
	}

	owner := f.lookup("ann", "home", f.ann)
	if titles(owner.Sections) != "Work(-,Tools,-)" || owner.Sections[0].Children[2].Data != "[secret]" {
		t.Fatalf("owner view = %+v", owner.Sections)
	}
	visitor := f.lookup("ann", "home", "")
	if len(visitor.Sections[0].Children) != 2 || visitor.LayoutVersion != 1 {
		t.Fatalf("visitor view = %+v", visitor.Sections)
	}
	data, err := json.Marshal(visitor)
	if err != nil || strings.Contains(string(data), "secret") {
		t.Fatalf("visitor JSON = %s, %v", data, err)
	}
	if !strings.Contains(string(data), `"kind":"note","width":2,"height":3,"config":{"text":"hi","private":false},"data":"[hi]"}`) {
		t.Fatalf("visitor JSON = %s", data)
	}

	// A widget at level 5 is allowed, because widgets aren't nesting levels.
	deep := nested(MaxDepth)
	inner := &deep[0]
	for len(inner.Children) > 0 {
		inner = &inner.Children[0]
	}
	inner.Children = []Widget{note(1, `{"text":"deep"}`)}
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 1, deep); err != nil {
		t.Fatalf("widget at level %d: %v", MaxDepth, err)
	}
}

func TestContentWidgetValidation(t *testing.T) {
	f := newFixture(t)
	withNotes(f)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home"})

	withChild := note(2, `{"text":"a"}`)
	withChild.Children = []Widget{sec(1, "")}
	withTitle := note(2, `{"text":"a"}`)
	withTitle.Title = ptr("T")
	tall := note(2, `{"text":"a"}`)
	tall.Height = MaxHeightRows + 1
	sectionConfig := sec(2, "")
	sectionConfig.Config = json.RawMessage(`{}`)
	for name, layout := range map[string][]Widget{
		"unknown kind":   {sec(6, "", Widget{Kind: "clock", Width: 2})},
		"missing kind":   {sec(6, "", Widget{Width: 2})},
		"top level":      {note(2, `{"text":"a"}`)},
		"children":       {sec(6, "", withChild)},
		"title":          {sec(6, "", withTitle)},
		"invalid config": {sec(6, "", note(2, `{"text":" "}`))},
		"unknown field":  {sec(6, "", note(2, `{"text":"a","color":"red"}`))},
		"width 13":       {sec(6, "", note(13, `{"text":"a"}`))},
		"height 51":      {sec(6, "", tall)},
		"section config": {sectionConfig},
	} {
		if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, layout); !errors.Is(err, ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
	if v := f.lookup("ann", "home", f.ann); v.LayoutVersion != 0 {
		t.Fatalf("invalid saves changed the layout: %+v", v)
	}
}

func TestResolveCache(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	loads := 0
	f.s.Kinds["counter"] = Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) { return raw, nil },
		Resolve: func(ctx context.Context, _ Viewer, raw json.RawMessage) (any, bool, error) {
			v, err := Cached(ctx, string(raw), func() (int, error) { loads++; return loads, nil })
			return v, true, err
		},
	}
	w := func(config string) Widget { return Widget{Kind: "counter", Width: 1, Config: json.RawMessage(config)} }
	if _, err := f.s.SaveLayout(f.ctx, f.ann, d.ID, 0, []Widget{sec(6, "", w(`"a"`), w(`"b"`), sec(3, "", w(`"a"`)))}); err != nil {
		t.Fatal(err)
	}
	if loads != 2 {
		t.Fatalf("save loaded %d times, want 2", loads)
	}
	v := f.lookup("ann", "home", f.ann)
	// A new lookup starts with an empty cache.
	if got := v.Sections[0].Children; got[0].Data != 3 || got[1].Data != 4 || got[2].Children[0].Data != 3 {
		t.Fatalf("data = %v %v %v", got[0].Data, got[1].Data, got[2].Children[0].Data)
	}
}
