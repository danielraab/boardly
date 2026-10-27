package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/favourites"
)

func recentWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindRecentFavourites, Width: 2, Config: json.RawMessage(config)}
}

func TestRecentFavouritesConfig(t *testing.T) {
	f := newFixture(t)
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")

	if err := f.save(recentWidget(`{"folderId":"` + work + `","limit":5,"days":7,"title":"  New ","size":"L"}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"days":7,"folderId":"` + work + `","limit":5,"showIfPublic":false,"size":"L","title":"New"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}
	if err := f.save(recentWidget(`{"folderId":"` + work + `","limit":100,"days":null,"size":"M"}`)); err != nil {
		t.Fatalf("no days: %v", err)
	}

	for name, config := range map[string]string{
		"no limit":      `{"folderId":"` + work + `","size":"M"}`,
		"limit 0":       `{"folderId":"` + work + `","limit":0,"size":"M"}`,
		"limit 101":     `{"folderId":"` + work + `","limit":101,"size":"M"}`,
		"days 0":        `{"folderId":"` + work + `","limit":10,"days":0,"size":"M"}`,
		"days 366":      `{"folderId":"` + work + `","limit":10,"days":366,"size":"M"}`,
		"size XL":       `{"folderId":"` + work + `","limit":10,"size":"XL"}`,
		"no folder":     `{"limit":10,"size":"M"}`,
		"long folder":   `{"folderId":"` + strings.Repeat("a", 65) + `","limit":10,"size":"M"}`,
		"unknown field": `{"folderId":"` + work + `","limit":10,"size":"M","layout":"list"}`,
		"long title":    `{"folderId":"` + work + `","limit":10,"title":"` + strings.Repeat("a", 201) + `","size":"M"}`,
	} {
		if err := f.save(recentWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}

	// A deleted folder doesn't block saving.
	if err := f.fav.Delete(f.ctx, f.ann, work, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	if err := f.save(recentWidget(`{"folderId":"` + work + `","limit":10,"size":"M"}`)); err != nil {
		t.Fatalf("save with a deleted folder: %v", err)
	}
}

func TestRecentFavouritesResolve(t *testing.T) {
	f := newFixture(t)
	fixed := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = time.Now })
	added := func(id string, ago time.Duration) {
		if _, err := f.fav.DB.Exec(`UPDATE favourite_nodes SET created_at = ? WHERE id = ?`, db.FormatTime(fixed.Add(-ago)), id); err != nil {
			t.Fatal(err)
		}
	}
	day := 24 * time.Hour
	work := f.node(f.ann, "", favourites.KindFolder, "Work", "")
	mail := f.node(f.ann, work, favourites.KindLink, "Mail", "https://mail.example.org/inbox")
	added(mail, 3*day)
	tools := f.node(f.ann, work, favourites.KindFolder, "Tools", "")
	added(f.node(f.ann, tools, favourites.KindLink, "GitLab", "https://gitlab.example.org/"), 2*day)
	added(f.node(f.ann, tools, favourites.KindLink, "Grafana", "https://grafana.example.org/"), 30*day)
	added(f.node(f.ann, "", favourites.KindLink, "Secret", "https://secret.example.org/"), 0)
	gone := f.node(f.ann, "", favourites.KindFolder, "Gone", "")
	if err := f.fav.Delete(f.ctx, f.ann, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	// Mail's origin has a favicon.
	if _, err := f.fav.DB.Exec(`INSERT INTO images (id, source, content_type, size, created_at) VALUES ('11111111-1111-1111-1111-111111111111', 'favicon', 'image/png', 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.fav.DB.Exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://mail.example.org', '11111111-1111-1111-1111-111111111111', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	cfg := func(folder string, limit int, days int, public bool) string {
		c := recentConfig{FolderID: folder, Limit: limit, Size: "M", ShowIfPublic: public}
		if days > 0 {
			c.Days = &days
		}
		raw, _ := json.Marshal(c)
		return string(raw)
	}
	if err := f.save(
		recentWidget(cfg(work, 10, 0, false)),
		recentWidget(cfg(work, 2, 0, true)),
		recentWidget(cfg(work, 10, 7, true)),
		recentWidget(cfg(gone, 10, 0, true)),
		recentWidget(cfg(work, 10, 1, true)),
	); err != nil {
		t.Fatal(err)
	}

	owner := f.lookup(f.ann)
	if len(owner) != 5 {
		t.Fatalf("owner sees %d widgets, want 5", len(owner))
	}
	if got := titles(t, owner[0]["data"]); got != "GitLab,Mail,Grafana" {
		t.Errorf("all = %s", got)
	}
	if got := titles(t, owner[1]["data"]); got != "GitLab,Mail" {
		t.Errorf("limit 2 = %s", got)
	}
	data, _ := json.Marshal(owner[2]["data"])
	sub, err := f.fav.Subtree(f.ctx, f.ann, tools)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"links":[` +
		`{"id":"` + sub.Children[0].ID + `","title":"GitLab","url":"https://gitlab.example.org/","createdAt":"2026-10-25T12:00:00Z"},` +
		`{"id":"` + mail + `","title":"Mail","url":"https://mail.example.org/inbox","iconImageId":"11111111-1111-1111-1111-111111111111","createdAt":"2026-10-24T12:00:00Z"}]}`
	if string(data) != canonical(t, want) {
		t.Errorf("last 7 days =\n%s\nwant\n%s", data, canonical(t, want))
	}
	if got, _ := json.Marshal(owner[3]["data"]); string(got) != `{"missing":true}` {
		t.Errorf("deleted folder = %s", got)
	}
	if got, _ := json.Marshal(owner[4]["data"]); string(got) != `{"links":[]}` {
		t.Errorf("nothing in the last day = %s", got)
	}

	// Visitors get the public widgets of existing folders, and only their links.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 3 {
			t.Fatalf("visitor %q sees %d widgets, want 3", viewer, len(visitor))
		}
		if got := titles(t, visitor[0]["data"]); got != "GitLab,Mail" {
			t.Errorf("visitor limit 2 = %s", got)
		}
		raw, _ := json.Marshal(visitor)
		for _, hidden := range []string{"Secret", "Grafana", `"Work"`, `"Tools"`, `"folder"`, `"tags"`} {
			if strings.Contains(string(raw), hidden) {
				t.Errorf("visitor response contains %s: %s", hidden, raw)
			}
		}
	}
}
