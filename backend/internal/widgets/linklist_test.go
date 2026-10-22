package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"boardly/internal/dashboards"
	"boardly/internal/favourites"
)

func linkList(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindLinkList, Width: 2, Config: json.RawMessage(config)}
}

func TestLinkListConfig(t *testing.T) {
	f := newFixture(t)
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")

	if err := f.save(linkList(`{"folderId":"` + work + `","title":"  Work links ","layout":"tiles","size":"L"}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"folderId":"` + work + `","includeSubfolders":false,"layout":"tiles","showIfPublic":false,"size":"L","title":"Work links"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}

	for name, config := range map[string]string{
		"layout grid":    `{"folderId":"` + work + `","layout":"grid","size":"M"}`,
		"no layout":      `{"folderId":"` + work + `","size":"M"}`,
		"size XL":        `{"folderId":"` + work + `","layout":"list","size":"XL"}`,
		"no folder":      `{"layout":"list","size":"M"}`,
		"empty folder":   `{"folderId":"","layout":"list","size":"M"}`,
		"long folder ID": `{"folderId":"` + strings.Repeat("a", 65) + `","layout":"list","size":"M"}`,
		"unknown field":  `{"folderId":"` + work + `","layout":"list","size":"M","target":"_self"}`,
		"long title":     `{"folderId":"` + work + `","title":"` + strings.Repeat("a", 201) + `","layout":"list","size":"M"}`,
	} {
		if err := f.save(linkList(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}

	// A deleted folder doesn't block saving.
	if err := f.fav.Delete(f.ctx, f.ann, work, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	if err := f.save(linkList(`{"folderId":"` + work + `","layout":"list","size":"M"}`)); err != nil {
		t.Fatalf("save with a deleted folder: %v", err)
	}
}

// titles returns the titles of a link list's links in data.
func titles(t *testing.T, data any) string {
	t.Helper()
	raw, _ := json.Marshal(data)
	var d struct {
		Links []struct{ Title string } `json:"links"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, l := range d.Links {
		out = append(out, l.Title)
	}
	return strings.Join(out, ",")
}

func TestLinkListResolve(t *testing.T) {
	f := newFixture(t)
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")
	mail := f.node(f.ann, work, favourites.KindLink, "Mail", "https://mail.example.org/inbox")
	tools := f.node(f.ann, work, favourites.KindFolder, "Tools", "")
	f.node(f.ann, tools, favourites.KindLink, "GitLab", "https://gitlab.example.org/")
	deep := f.node(f.ann, tools, favourites.KindFolder, "Deep", "")
	f.node(f.ann, deep, favourites.KindLink, "Grafana", "https://grafana.example.org/")
	f.node(f.ann, work, favourites.KindLink, "Wiki", "https://wiki.example.org/")
	f.node(f.ann, "", favourites.KindLink, "Secret", "https://secret.example.org/")
	gone := f.node(f.ann, "", favourites.KindFolder, "Gone", "")
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

	cfg := func(folder string, subfolders, public bool) string {
		c, _ := json.Marshal(linkListConfig{FolderID: folder, IncludeSubfolders: subfolders, Layout: "list", Size: "M", ShowIfPublic: public})
		return string(c)
	}
	if err := f.save(
		linkList(cfg(work, false, false)),
		linkList(cfg(work, true, true)),
		linkList(cfg(tools, false, true)),
		linkList(cfg(gone, true, true)),
		linkList(cfg(work, false, true)),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 5 {
		t.Fatalf("owner sees %d widgets, want 5", len(owner))
	}
	data, _ := json.Marshal(owner[0]["data"])
	want := `{"links":[{"id":"` + mail + `","title":"Mail","url":"https://mail.example.org/inbox","iconImageId":"11111111-1111-1111-1111-111111111111"},` +
		`{"id":"%s","title":"Wiki","url":"https://wiki.example.org/"}]}`
	sub, err := f.fav.Subtree(f.ctx, f.ann, work)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(want, "%s", sub.Children[2].ID, 1); string(data) != canonical(t, want) {
		t.Fatalf("owner data =\n%s\nwant\n%s", data, want)
	}
	if got := titles(t, owner[1]["data"]); got != "Mail,GitLab,Grafana,Wiki" {
		t.Errorf("with subfolders = %s", got)
	}
	if got, _ := json.Marshal(owner[2]["data"]); titles(t, owner[2]["data"]) != "GitLab" {
		t.Errorf("tools without subfolders = %s", got)
	}
	if got, _ := json.Marshal(owner[3]["data"]); string(got) != `{"missing":true}` {
		t.Errorf("deleted folder = %s", got)
	}

	// Visitors get the public link lists of existing folders, and only their links.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 3 {
			t.Fatalf("visitor %q sees %d widgets, want 3", viewer, len(visitor))
		}
		if got := titles(t, visitor[0]["data"]); got != "Mail,GitLab,Grafana,Wiki" {
			t.Errorf("visitor with subfolders = %s", got)
		}
		raw, _ := json.Marshal(visitor[2])
		if strings.Contains(string(raw), "GitLab") || strings.Contains(string(raw), "Tools") {
			t.Errorf("visitor without subfolders = %s", raw)
		}
		raw, _ = json.Marshal(visitor)
		for _, hidden := range []string{"Secret", `"Work"`, `"Tools"`, `"Deep"`, `"folder"`} {
			if strings.Contains(string(raw), hidden) {
				t.Errorf("visitor response contains %s: %s", hidden, raw)
			}
		}
	}
}
