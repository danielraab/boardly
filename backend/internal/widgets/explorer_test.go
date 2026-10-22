package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"famodash/internal/dashboards"
	"famodash/internal/dbtest"
	"famodash/internal/favourites"
	"famodash/internal/images"
	"famodash/internal/users"
)

type fixture struct {
	t         *testing.T
	ctx       context.Context
	dash      *dashboards.Store
	fav       *favourites.Store
	ann, bob  string
	dashboard string
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
	imageStore, err := images.NewStore(database, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, ctx: ctx, dash: dashboards.NewStore(database), fav: favourites.NewStore(database), ann: ann.ID, bob: bob.ID}
	Register(f.dash, f.fav, images.NewFavicons(imageStore, false))
	d, err := f.dash.Create(ctx, ann.ID, dashboards.Fields{Title: "Home", Slug: "home", IsPublic: true})
	if err != nil {
		t.Fatal(err)
	}
	f.dashboard = d.ID
	return f
}

func (f *fixture) node(userID, parentID string, kind favourites.Kind, title, url string) string {
	f.t.Helper()
	n := favourites.NewNode{ParentID: parentID, Kind: kind, Title: &title}
	if url != "" {
		n.URL = &url
	}
	created, err := f.fav.Create(f.ctx, userID, n)
	if err != nil {
		f.t.Fatal(err)
	}
	return created.ID
}

// save saves one section with the given widgets and returns the error.
func (f *fixture) save(widgets ...dashboards.Widget) error {
	l, err := f.dash.Lookup(f.ctx, "ann", "home", f.ann)
	if err != nil {
		f.t.Fatal(err)
	}
	_, err = f.dash.SaveLayout(f.ctx, f.ann, f.dashboard, l.LayoutVersion, []dashboards.Widget{
		{Kind: dashboards.KindSection, Width: 6, Children: widgets},
	})
	return err
}

// lookup returns the widgets in the first section as JSON, for viewerID.
func (f *fixture) lookup(viewerID string) []map[string]any {
	f.t.Helper()
	v, err := f.dash.Lookup(f.ctx, "ann", "home", viewerID)
	if err != nil {
		f.t.Fatal(err)
	}
	raw, err := json.Marshal(v.Sections[0].Children)
	if err != nil {
		f.t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		f.t.Fatal(err)
	}
	return out
}

// canonical returns the JSON s with sorted keys, as lookup's maps marshal.
func canonical(t *testing.T, s string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func explorer(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindFavouriteExplorer, Width: 2, Config: json.RawMessage(config)}
}

func TestExplorerConfig(t *testing.T) {
	f := newFixture(t)
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")

	if err := f.save(explorer(`{"rootFolderId":"` + work + `","size":"L"}`)); err != nil {
		t.Fatal(err)
	}
	if got := f.lookup(f.ann)[0]["config"].(map[string]any); got["showIfPublic"] != false || got["title"] != nil {
		t.Fatalf("config = %v, want showIfPublic false and no title", got)
	}
	for title, want := range map[string]any{`"  Work links "`: "Work links", `"   "`: nil, `"` + strings.Repeat("ä", 200) + `"`: strings.Repeat("ä", 200)} {
		if err := f.save(explorer(`{"rootFolderId":"` + work + `","title":` + title + `,"size":"M"}`)); err != nil {
			t.Fatal(err)
		}
		if got := f.lookup(f.ann)[0]["config"].(map[string]any)["title"]; got != want {
			t.Errorf("title %s saved as %v, want %v", title, got, want)
		}
	}

	for name, config := range map[string]string{
		"size XL":        `{"rootFolderId":"` + work + `","size":"XL"}`,
		"no size":        `{"rootFolderId":"` + work + `"}`,
		"no folder":      `{"size":"M"}`,
		"empty folder":   `{"rootFolderId":"","size":"M"}`,
		"long folder ID": `{"rootFolderId":"` + strings.Repeat("a", 65) + `","size":"M"}`,
		"unknown field":  `{"rootFolderId":"` + work + `","size":"M","depth":2}`,
		"long title":     `{"rootFolderId":"` + work + `","title":"` + strings.Repeat("a", 201) + `","size":"M"}`,
	} {
		if err := f.save(explorer(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}

	// A deleted folder doesn't block saving.
	if err := f.fav.Delete(f.ctx, f.ann, work, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	if err := f.save(explorer(`{"rootFolderId":"` + work + `","size":"M"}`)); err != nil {
		t.Fatalf("save with a deleted folder: %v", err)
	}
}

func TestExplorerResolve(t *testing.T) {
	f := newFixture(t)
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")
	tools := f.node(f.ann, work, favourites.KindFolder, "Tools", "")
	f.node(f.ann, tools, favourites.KindLink, "GitLab", "https://gitlab.example.org/")
	f.node(f.ann, work, favourites.KindLink, "Mail", "https://mail.example.org/inbox")
	f.node(f.ann, "", favourites.KindLink, "Secret", "https://secret.example.org/")
	empty := f.node(f.ann, "", favourites.KindFolder, "Empty", "")
	gone := f.node(f.ann, "", favourites.KindFolder, "Gone", "")
	bobs := f.node(f.bob, "", favourites.KindFolder, "Bob's", "")
	if err := f.fav.Delete(f.ctx, f.ann, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	// Mail's origin has a favicon.
	db := f.fav.DB
	if _, err := db.Exec(`INSERT INTO images (id, source, content_type, size, created_at) VALUES ('11111111-1111-1111-1111-111111111111', 'favicon', 'image/png', 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://mail.example.org', '11111111-1111-1111-1111-111111111111', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	cfg := func(folder string, public bool) string {
		c, _ := json.Marshal(explorerConfig{RootFolderID: folder, Size: "M", ShowIfPublic: public})
		return string(c)
	}
	if err := f.save(
		explorer(cfg(work, false)),
		explorer(cfg(work, true)),
		explorer(cfg(empty, true)),
		explorer(cfg(gone, true)),
		explorer(cfg(bobs, true)),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 5 {
		t.Fatalf("owner sees %d widgets, want 5", len(owner))
	}
	data, _ := json.Marshal(owner[0]["data"])
	want := `{"children":[{"id":"` + tools + `","kind":"folder","title":"Tools","children":[{"id":"%s","kind":"link","title":"GitLab","url":"https://gitlab.example.org/"}]},` +
		`{"id":"%s","kind":"link","title":"Mail","url":"https://mail.example.org/inbox","iconImageId":"11111111-1111-1111-1111-111111111111"}]}`
	sub, err := f.fav.Subtree(f.ctx, f.ann, work)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(strings.Replace(want, "%s", sub.Children[0].Children[0].ID, 1), "%s", sub.Children[1].ID, 1); string(data) != canonical(t, want) {
		t.Fatalf("owner data =\n%s\nwant\n%s", data, want)
	}
	// Another user's folder is as good as a missing one.
	for i, want := range map[int]string{2: `{"children":[]}`, 3: `{"missing":true}`, 4: `{"missing":true}`} {
		if got, _ := json.Marshal(owner[i]["data"]); string(got) != want {
			t.Errorf("owner widget %d data = %s, want %s", i, got, want)
		}
	}

	// Visitors only get the public explorers of existing folders.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 2 {
			t.Fatalf("visitor %q sees %d widgets, want 2", viewer, len(visitor))
		}
		if got, _ := json.Marshal(visitor[1]["data"]); string(got) != `{"children":[]}` {
			t.Fatalf("visitor empty folder = %s", got)
		}
		raw, _ := json.Marshal(visitor)
		if !strings.Contains(string(raw), "GitLab") || strings.Contains(string(raw), "Secret") || strings.Contains(string(raw), `"title":"Work"`) {
			t.Fatalf("visitor response = %s", raw)
		}
	}
}
