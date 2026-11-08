package dashboards

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func (f *fixture) importDashboard(slug string, sections []Widget) (Dashboard, int) {
	f.t.Helper()
	tx, err := f.s.DB.BeginTx(f.ctx, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	d, dropped, err := f.s.ImportTx(f.ctx, tx, f.ann, "Home", slug, DefaultColumns, sections)
	if err != nil {
		_ = tx.Rollback()
		f.t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		f.t.Fatal(err)
	}
	return d, dropped
}

func TestImportSlugsAndFlags(t *testing.T) {
	f := newFixture(t)
	f.create(f.ann, Fields{Title: "Home", Slug: "home", IsDefault: true})
	f.create(f.ann, Fields{Title: "Home 2", Slug: "home-2"})

	d, _ := f.importDashboard("home", nil)
	if d.Slug != "home-3" || d.IsDefault || d.IsPublic {
		t.Errorf("imported = %+v, want slug home-3, not default, not public", d)
	}
	if d, _ := f.importDashboard("Not a slug!", nil); d.Slug == "" || strings.Contains(d.Slug, " ") {
		t.Errorf("invalid slug became %q", d.Slug)
	}
	long := strings.Repeat("a", 64)
	for range 2 {
		if d, _ := f.importDashboard(long, nil); d.Slug == "" {
			t.Errorf("long slug: %+v", d)
		}
	}
}

func TestImportOlderWidgetHeights(t *testing.T) {
	f := newFixture(t)
	withNotes(f)
	content := func(config string) Widget {
		t.Helper()
		w, err := StoredWidget("note", 2, json.RawMessage(config), nil)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	section := func(height string, children ...Widget) Widget {
		t.Helper()
		w, err := StoredWidget(KindSection, 6, json.RawMessage(`{"border":true,"height":`+height+`}`), children)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	fixed := `{"height":6,"config":{"text":"a"}}`
	withoutHeight := `{"config":{"text":"a"}}`
	inner := section(`4`)

	for name, tt := range map[string]struct {
		section Widget
		want    Height
	}{
		"only child":       {section(`"auto"`, content(fixed)), 6},
		"already fixed":    {section(`4`, content(fixed)), 4},
		"several children": {section(`"auto"`, content(fixed), content(fixed)), 0},
		"no heights":       {section(`"auto"`, content(withoutHeight)), 0},
		"nested section":   {section(`"auto"`, inner), 0},
	} {
		d, dropped := f.importDashboard("home", []Widget{tt.section})
		if dropped != 0 {
			t.Errorf("%s: dropped = %d", name, dropped)
		}
		layout, err := f.s.Lookup(f.ctx, "ann", d.Slug, f.ann)
		if err != nil {
			t.Fatal(err)
		}
		got := layout.Sections[0]
		if got.Height != tt.want || len(got.Children) != len(tt.section.Children) {
			t.Errorf("%s: section = %+v, want height %d", name, got, tt.want)
		}
		for _, child := range got.Children {
			if child.Kind != KindSection && child.Height != 0 {
				t.Errorf("%s: widget kept its height: %+v", name, child)
			}
		}
	}
}

func TestImportWidgets(t *testing.T) {
	f := newFixture(t)
	f.s.Kinds["counter"] = Kind{Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
		var c struct{ N int }
		if err := json.Unmarshal(raw, &c); err != nil || c.N < 0 {
			return nil, fmt.Errorf("%w: bad counter", ErrInvalidLayout)
		}
		return raw, nil
	}}
	section, err := StoredWidget(KindSection, 6, json.RawMessage(`{"title":"Top","border":true,"height":"auto"}`), []Widget{
		{Kind: "counter", Width: 2, Config: json.RawMessage(`{"N":1}`)},
		{Kind: "counter", Width: 2, Config: json.RawMessage(`{"N":-1}`)},
		{Kind: "unknown", Width: 2, Config: json.RawMessage(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if section.Title == nil || *section.Title != "Top" || !section.Border {
		t.Fatalf("section = %+v", section)
	}
	content, err := StoredWidget("counter", 2, json.RawMessage(`{"height":3,"config":{"N":5}}`), nil)
	if err != nil || content.Height != 3 || string(content.Config) != `{"N":5}` {
		t.Fatalf("content = %+v, %v", content, err)
	}

	d, dropped := f.importDashboard("home", []Widget{section})
	if dropped != 2 {
		t.Errorf("dropped = %d, want 2 (bad config and unknown kind)", dropped)
	}
	layout, err := f.s.Lookup(f.ctx, "ann", d.Slug, f.ann)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Sections) != 1 || len(layout.Sections[0].Children) != 1 || !layout.Sections[0].Border {
		t.Errorf("sections = %+v", layout.Sections)
	}
}
