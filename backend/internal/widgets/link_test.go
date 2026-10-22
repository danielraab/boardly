package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"famodash/internal/dashboards"
	"famodash/internal/favourites"
	"famodash/internal/images"
)

func link(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindLink, Width: 2, Config: json.RawMessage(config)}
}

// urlLink is the config of a plain URL link with the given extra fields.
func urlLink(extra string) string {
	return `{"source":"url","url":"https://example.org/","title":null,"iconSource":"favicon","imageId":null,"icon":null,"target":"_blank","size":"M"` + extra + `}`
}

// favLink is the config of a link to the favourite id.
func favLink(id, extra string) string {
	return `{"source":"favourite","favouriteId":"` + id + `","url":"https://old.example.org/","favouriteTitle":"Old","title":null,"iconSource":"favicon","imageId":null,"icon":null,"target":"_blank","size":"M"` + extra + `}`
}

// savedConfigs saves the widgets and returns their stored configs.
func (f *fixture) savedConfigs(widgets ...dashboards.Widget) []map[string]any {
	f.t.Helper()
	if err := f.save(widgets...); err != nil {
		f.t.Fatal(err)
	}
	var out []map[string]any
	for _, w := range f.lookup(f.ann) {
		out = append(out, w["config"].(map[string]any))
	}
	return out
}

func TestLinkConfig(t *testing.T) {
	f := newFixture(t)

	got := f.savedConfigs(link(strings.Replace(urlLink(`,"iconSource":"symbol","icon":"mail"`), `"title":null`, `"title":"  Mail "`, 1)))[0]
	want := map[string]any{"source": "url", "url": "https://example.org/", "title": "Mail", "iconSource": "symbol", "imageId": nil, "icon": "mail", "target": "_blank", "size": "M"}
	if raw, _ := json.Marshal(got); string(raw) != canonical(t, mustJSON(t, want)) {
		t.Fatalf("config = %s", raw)
	}
	// An empty title is no title, and favourite fields are dropped from a URL link.
	got = f.savedConfigs(link(`{"source":"url","favouriteId":"x","favouriteTitle":"T","url":"https://example.org/","title":"  ","iconSource":"image","imageId":"img","icon":null,"target":"_self","size":"S"}`))[0]
	if got["title"] != nil || got["favouriteId"] != nil || got["favouriteTitle"] != nil || got["imageId"] != "img" {
		t.Fatalf("config = %v", got)
	}

	for name, config := range map[string]string{
		"no source":     strings.Replace(urlLink(""), `"source":"url"`, `"source":"tab"`, 1),
		"javascript":    strings.Replace(urlLink(""), "https://example.org/", "javascript:alert(1)", 1),
		"relative":      strings.Replace(urlLink(""), "https://example.org/", "/docs", 1),
		"long url":      strings.Replace(urlLink(""), "https://example.org/", "https://example.org/"+strings.Repeat("a", 2048), 1),
		"long title":    strings.Replace(urlLink(""), `"title":null`, `"title":"`+strings.Repeat("a", 201)+`"`, 1),
		"empty image":   urlLink(`,"iconSource":"image","imageId":""`),
		"long image":    urlLink(`,"iconSource":"image","imageId":"` + strings.Repeat("a", 65) + `"`),
		"no image":      urlLink(`,"iconSource":"image"`),
		"unknown icon":  urlLink(`,"iconSource":"symbol","icon":"github"`),
		"no icon":       urlLink(`,"iconSource":"symbol"`),
		"icon source":   urlLink(`,"iconSource":"emoji"`),
		"target":        strings.Replace(urlLink(""), `"_blank"`, `"_top"`, 1),
		"size":          strings.Replace(urlLink(""), `"size":"M"`, `"size":"XL"`, 1),
		"unknown field": urlLink(`,"color":"red"`),
		"no favourite":  favLink("", ""),
	} {
		if err := f.save(link(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
	// 200 characters are fine.
	if err := f.save(link(strings.Replace(urlLink(""), `"title":null`, `"title":"`+strings.Repeat("ä", 200)+`"`, 1))); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIconSources(t *testing.T) {
	f := newFixture(t)
	got := f.savedConfigs(
		// Only the field of the source is kept.
		link(urlLink(`,"iconSource":"favicon","imageId":"img","icon":"mail"`)),
		link(urlLink(`,"iconSource":"none","imageId":"img","icon":"mail"`)),
		link(urlLink(`,"iconSource":"image","imageId":"img","icon":"mail"`)),
		link(urlLink(`,"iconSource":"symbol","imageId":"img","icon":"mail"`)),
		// Saved before icon sources existed.
		link(`{"source":"url","url":"https://example.org/","title":null,"imageId":"img","icon":"mail","target":"_blank","size":"M"}`),
		link(`{"source":"url","url":"https://example.org/","title":null,"imageId":null,"icon":"mail","target":"_blank","size":"M"}`),
		link(`{"source":"url","url":"https://example.org/","title":null,"imageId":null,"icon":null,"target":"_blank","size":"M"}`),
	)
	want := []string{"favicon - -", "none - -", "image img -", "symbol - mail", "image img -", "symbol - mail", "favicon - -"}
	for i, c := range got {
		str := func(v any) string {
			if v == nil {
				return "-"
			}
			return v.(string)
		}
		if g := c["iconSource"].(string) + " " + str(c["imageId"]) + " " + str(c["icon"]); g != want[i] {
			t.Errorf("widget %d: %s, want %s", i, g, want[i])
		}
	}
}

// addFavicon makes id the known favicon of origin.
func (f *fixture) addFavicon(origin, id string) {
	f.t.Helper()
	if _, err := f.fav.DB.Exec(`INSERT INTO images (id, source, content_type, size, created_at) VALUES (?, 'favicon', 'image/png', 1, '2026-01-01T00:00:00Z')`, id); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.fav.DB.Exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES (?, ?, '2026-01-01T00:00:00Z')`, origin, id); err != nil {
		f.t.Fatal(err)
	}
}

func TestLinkWebsiteIcon(t *testing.T) {
	f := newFixture(t)
	f.addFavicon("https://example.org", "11111111-1111-1111-1111-111111111111")
	f.addFavicon("https://github.com", "22222222-2222-2222-2222-222222222222")
	f.addFavicon("https://gitlab.com", "33333333-3333-3333-3333-333333333333")
	repo := f.node(f.ann, "", favourites.KindLink, "Repo", "https://gitlab.com/ann")
	if err := f.save(
		link(urlLink("")),
		link(urlLink(`,"iconSource":"symbol","icon":"mail"`)),
		link(strings.Replace(urlLink(""), "https://example.org/", "https://unknown.example.net/", 1)),
		link(favLink(repo, "")),
		link(favLink(repo, `,"iconSource":"none"`)),
	); err != nil {
		t.Fatal(err)
	}
	data := func() []string {
		var out []string
		for _, w := range f.lookup("") {
			out = append(out, mustJSON(t, w["data"]))
		}
		return out
	}
	want := []string{
		`{"iconImageId":"11111111-1111-1111-1111-111111111111"}`,
		"null",
		"null",
		`{"iconImageId":"33333333-3333-3333-3333-333333333333","title":"Repo","url":"https://gitlab.com/ann"}`,
		`{"title":"Repo","url":"https://gitlab.com/ann"}`,
	}
	if got := data(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("visitor data =\n%s", strings.Join(got, "\n"))
	}

	// The favourite moves to another site: its icon follows without a save.
	newURL := "https://github.com/ann"
	if _, err := f.fav.Update(f.ctx, f.ann, repo, favourites.Patch{URL: &newURL}); err != nil {
		t.Fatal(err)
	}
	if got := data()[3]; got != `{"iconImageId":"22222222-2222-2222-2222-222222222222","title":"Repo","url":"https://github.com/ann"}` {
		t.Fatalf("moved favourite data = %s", got)
	}
}

func TestLinkFitsContent(t *testing.T) {
	f := newFixture(t)
	tall := link(urlLink(""))
	tall.Height = 3
	if err := f.save(tall); err != nil {
		t.Fatal(err)
	}
	if got := f.lookup(f.ann)[0]["height"]; got != "auto" {
		t.Fatalf("height = %v, want auto", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestLinkFollowsFavourite(t *testing.T) {
	f := newFixture(t)
	mail := f.node(f.ann, "", favourites.KindLink, "Mail", "https://mail.example.org/")
	gone := f.node(f.ann, "", favourites.KindLink, "Gone", "https://gone.example.org/")
	bobs := f.node(f.bob, "", favourites.KindLink, "Bob's", "https://bob.example.org/")
	if err := f.fav.Delete(f.ctx, f.ann, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}

	got := f.savedConfigs(
		link(favLink(mail, "")),
		link(favLink(gone, "")),
		link(favLink(gone, `,"title":"Mine"`)),
		link(favLink(bobs, "")),
	)
	if c := got[0]; c["source"] != "favourite" || c["favouriteId"] != mail || c["url"] != "https://mail.example.org/" || c["favouriteTitle"] != "Mail" || c["title"] != nil {
		t.Errorf("refreshed = %v", c)
	}
	// A missing favourite becomes a URL link with the last copies.
	if c := got[1]; c["source"] != "url" || c["favouriteId"] != nil || c["favouriteTitle"] != nil || c["url"] != "https://old.example.org/" || c["title"] != "Old" {
		t.Errorf("converted = %v", c)
	}
	if c := got[2]; c["source"] != "url" || c["title"] != "Mine" {
		t.Errorf("converted with own title = %v", c)
	}
	// Another user's favourite is as good as a missing one.
	if c := got[3]; c["source"] != "url" || c["url"] != "https://old.example.org/" || strings.Contains(mustJSON(t, c), "bob.example.org") {
		t.Errorf("foreign = %v", c)
	}
}

func TestLinkResolve(t *testing.T) {
	f := newFixture(t)
	title, url := "Mail", "https://mail.example.org/"
	n, err := f.fav.Create(f.ctx, f.ann, favourites.NewNode{Kind: favourites.KindLink, Title: &title, URL: &url, Tags: []string{"private-tag"}})
	if err != nil {
		t.Fatal(err)
	}
	f.node(f.ann, "", favourites.KindLink, "Other", "https://other.example.org/")
	if err := f.save(link(favLink(n.ID, "")), link(urlLink(""))); err != nil {
		t.Fatal(err)
	}

	// Renamed after the save: the lookup shows the live values.
	newTitle, newURL := "Webmail", "https://webmail.example.org/"
	if _, err := f.fav.Update(f.ctx, f.ann, n.ID, favourites.Patch{Title: &newTitle, URL: &newURL}); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []string{f.ann, "", f.bob} {
		widgets := f.lookup(viewer)
		if len(widgets) != 2 {
			t.Fatalf("viewer %q sees %d widgets", viewer, len(widgets))
		}
		if got := mustJSON(t, widgets[0]["data"]); got != `{"title":"Webmail","url":"https://webmail.example.org/"}` {
			t.Errorf("viewer %q data = %s", viewer, got)
		}
		if _, ok := widgets[1]["data"]; ok {
			t.Errorf("URL link has data: %v", widgets[1]["data"])
		}
		raw := mustJSON(t, widgets)
		if strings.Contains(raw, "private-tag") || strings.Contains(raw, "Other") {
			t.Errorf("viewer %q response = %s", viewer, raw)
		}
	}

	// Deleted: the copies from the last save, for everyone, until the next save.
	if err := f.fav.Delete(f.ctx, f.ann, n.ID, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []string{f.ann, ""} {
		if got := mustJSON(t, f.lookup(viewer)[0]["data"]); got != `{"title":"Mail","url":"https://mail.example.org/"}` {
			t.Errorf("deleted, viewer %q data = %s", viewer, got)
		}
	}
}

func TestLinkImageSurvivesCleanup(t *testing.T) {
	f := newFixture(t)
	store, err := images.NewStore(f.fav.DB, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	used, err := store.Save(f.ctx, "upload", f.ann, png)
	if err != nil {
		t.Fatal(err)
	}
	unused, err := store.Save(f.ctx, "upload", f.ann, png)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.save(link(urlLink(`,"iconSource":"image","imageId":"` + used.ID + `"`))); err != nil {
		t.Fatal(err)
	}

	store.Now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	if _, err := store.Sweep(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, file, err := store.Open(f.ctx, used.ID); err != nil {
		t.Fatalf("used image: %v", err)
	} else {
		_ = file.Close()
	}
	if _, _, err := store.Open(f.ctx, unused.ID); err == nil {
		t.Fatal("the unused image survived")
	}
}
