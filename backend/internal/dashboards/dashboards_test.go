package dashboards

import (
	"context"
	"errors"
	"strings"
	"testing"

	"famodash/internal/dbtest"
	"famodash/internal/users"
)

type fixture struct {
	t        *testing.T
	s        *Store
	ctx      context.Context
	ann, bob string
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
	if _, err := u.SetSlug(ctx, ann.ID, "ann"); err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, s: NewStore(database), ctx: ctx, ann: ann.ID, bob: bob.ID}
}

func (f *fixture) create(userID string, fields Fields) Dashboard {
	f.t.Helper()
	d, err := f.s.Create(f.ctx, userID, fields)
	if err != nil {
		f.t.Fatalf("create %+v: %v", fields, err)
	}
	return d
}

func (f *fixture) list(userID string) []Dashboard {
	f.t.Helper()
	l, err := f.s.List(f.ctx, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	return l
}

func defaults(l []Dashboard) string {
	var out []string
	for _, d := range l {
		if d.IsDefault {
			out = append(out, d.Slug)
		}
	}
	return strings.Join(out, ",")
}

func ptr[T any](v T) *T { return &v }

func TestListSortedByTitle(t *testing.T) {
	f := newFixture(t)
	f.create(f.ann, Fields{Title: "Work", Slug: "work"})
	f.create(f.ann, Fields{Title: "home", Slug: "Home"})
	f.create(f.bob, Fields{Title: "Bob's", Slug: "bobs"})

	l := f.list(f.ann)
	if len(l) != 2 || l[0].Slug != "home" || l[1].Slug != "work" {
		t.Fatalf("list = %+v", l)
	}
}

func TestDefaultSwitching(t *testing.T) {
	f := newFixture(t)
	home := f.create(f.ann, Fields{Title: "Home", Slug: "home", IsDefault: true})
	work := f.create(f.ann, Fields{Title: "Work", Slug: "work", IsDefault: true})
	f.create(f.bob, Fields{Title: "Home", Slug: "home", IsDefault: true})
	if got := defaults(f.list(f.ann)); got != "work" {
		t.Fatalf("default after create = %q, want work", got)
	}

	if _, err := f.s.Update(f.ctx, f.ann, home.ID, Patch{IsDefault: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if got := defaults(f.list(f.ann)); got != "home" {
		t.Fatalf("default after switch = %q, want home", got)
	}
	if got := defaults(f.list(f.bob)); got != "home" {
		t.Fatalf("bob's default changed: %q", got)
	}

	if _, err := f.s.Update(f.ctx, f.ann, home.ID, Patch{IsDefault: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if got := defaults(f.list(f.ann)); got != "" {
		t.Fatalf("default after clearing = %q, want none", got)
	}

	if _, err := f.s.Update(f.ctx, f.ann, work.ID, Patch{IsDefault: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Delete(f.ctx, f.ann, work.ID); err != nil {
		t.Fatal(err)
	}
	if l := f.list(f.ann); len(l) != 1 || defaults(l) != "" {
		t.Fatalf("after deleting the default: %+v", l)
	}
}

func TestSlugsAndTitles(t *testing.T) {
	f := newFixture(t)
	home := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	work := f.create(f.ann, Fields{Title: "Work", Slug: "work"})
	f.create(f.bob, Fields{Title: "Home", Slug: "home"})

	if _, err := f.s.Create(f.ctx, f.ann, Fields{Title: "Again", Slug: "HOME"}); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("create duplicate: %v", err)
	}
	if _, err := f.s.Update(f.ctx, f.ann, work.ID, Patch{Slug: ptr("home")}); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("rename to duplicate: %v", err)
	}
	if _, err := f.s.Create(f.ctx, f.ann, Fields{Title: "X", Slug: "x_y"}); !errors.Is(err, ErrInvalidSlug) {
		t.Errorf("invalid slug: %v", err)
	}
	for _, title := range []string{"   ", strings.Repeat("ä", 101)} {
		if _, err := f.s.Create(f.ctx, f.ann, Fields{Title: title, Slug: "x"}); !errors.Is(err, ErrInvalidTitle) {
			t.Errorf("title %q: %v", title, err)
		}
	}

	got, err := f.s.Update(f.ctx, f.ann, home.ID, Patch{Title: ptr("  Start "), Slug: ptr("Start"), IsPublic: ptr(true)})
	if err != nil || got != (Dashboard{ID: home.ID, Slug: "start", Title: "Start", IsPublic: true, Columns: DefaultColumns}) {
		t.Fatalf("update = %+v, %v", got, err)
	}
}

func TestColumns(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.ann, Fields{Title: "Home", Slug: "home"})
	if d.Columns != DefaultColumns {
		t.Fatalf("defaults = %+v, want %+v", d.Columns, DefaultColumns)
	}
	custom := f.create(f.ann, Fields{Title: "Wide", Slug: "wide", DesktopColumns: ptr(12), MobileColumns: ptr(1), MobileBreakpoint: ptr(1024)})
	if custom.Columns != (Columns{12, 1, 1024}) {
		t.Fatalf("custom = %+v", custom.Columns)
	}

	for name, fields := range map[string]Fields{
		"desktop 0":         {DesktopColumns: ptr(0)},
		"desktop 13":        {DesktopColumns: ptr(13)},
		"mobile 0":          {MobileColumns: ptr(0)},
		"mobile above desk": {DesktopColumns: ptr(4), MobileColumns: ptr(5)},
		"breakpoint 800":    {MobileBreakpoint: ptr(800)},
	} {
		fields.Title, fields.Slug = "X", "x"
		if _, err := f.s.Create(f.ctx, f.ann, fields); !errors.Is(err, ErrInvalidColumns) {
			t.Errorf("create %s: %v", name, err)
		}
	}

	// A patch of one count is checked against the stored other one.
	if _, err := f.s.Update(f.ctx, f.ann, d.ID, Patch{DesktopColumns: ptr(1)}); !errors.Is(err, ErrInvalidColumns) {
		t.Errorf("desktop below stored mobile: %v", err)
	}
	if _, err := f.s.Update(f.ctx, f.ann, d.ID, Patch{MobileColumns: ptr(7)}); !errors.Is(err, ErrInvalidColumns) {
		t.Errorf("mobile above stored desktop: %v", err)
	}
	if _, err := f.s.Update(f.ctx, f.ann, d.ID, Patch{MobileBreakpoint: ptr(100)}); !errors.Is(err, ErrInvalidColumns) {
		t.Errorf("breakpoint 100: %v", err)
	}
	if got := f.list(f.ann)[0]; got.Columns != DefaultColumns {
		t.Fatalf("failed patches changed the columns: %+v", got.Columns)
	}

	got, err := f.s.Update(f.ctx, f.ann, d.ID, Patch{DesktopColumns: ptr(8)})
	if err != nil || got.Columns != (Columns{8, 2, 768}) || got.Title != "Home" {
		t.Fatalf("patch desktop = %+v, %v", got, err)
	}
}

func TestForeignDashboard(t *testing.T) {
	f := newFixture(t)
	f.create(f.ann, Fields{Title: "Home", Slug: "home", IsDefault: true})
	bobs := f.create(f.bob, Fields{Title: "Bob", Slug: "bob"})

	if _, err := f.s.Update(f.ctx, f.ann, bobs.ID, Patch{Title: ptr("Mine"), IsDefault: ptr(true)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update foreign: %v", err)
	}
	if err := f.s.Delete(f.ctx, f.ann, bobs.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete foreign: %v", err)
	}
	if got := defaults(f.list(f.ann)); got != "home" {
		t.Fatalf("ann's default changed by a failed update: %q", got)
	}
	if l := f.list(f.bob); len(l) != 1 || l[0].Title != "Bob" {
		t.Fatalf("bob's dashboards changed: %+v", l)
	}
}

func TestLookup(t *testing.T) {
	f := newFixture(t)
	f.create(f.ann, Fields{Title: "Home", Slug: "home", IsPublic: true})
	f.create(f.ann, Fields{Title: "Links", Slug: "links", IsPublic: true})
	f.create(f.ann, Fields{Title: "Work", Slug: "work"})

	entries := func(v View) string {
		var out []string
		for _, e := range v.Dashboards {
			out = append(out, e.Slug)
		}
		return strings.Join(out, ",")
	}

	v, err := f.s.Lookup(f.ctx, "Ann", "WORK", f.ann)
	if err != nil || !v.IsOwner || v.Dashboard.Title != "Work" || v.OwnerSlug != "ann" || entries(v) != "home,links,work" {
		t.Fatalf("owner/private = %+v, %v", v, err)
	}

	for name, viewer := range map[string]string{"signed out": "", "other user": f.bob} {
		v, err := f.s.Lookup(f.ctx, "ann", "home", viewer)
		if err != nil || v.IsOwner || entries(v) != "home,links" {
			t.Errorf("%s/public = %+v, %v", name, v, err)
		}
		for _, path := range [][2]string{{"ann", "work"}, {"nobody", "home"}, {"ann", "nothing"}} {
			if _, err := f.s.Lookup(f.ctx, path[0], path[1], viewer); !errors.Is(err, ErrNotFound) {
				t.Errorf("%s %v: err = %v, want ErrNotFound", name, path, err)
			}
		}
	}
}
