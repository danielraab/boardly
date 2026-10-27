package favourites

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"boardly/internal/db"
)

var recentNow = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

// addedAgo sets when the node was created, relative to recentNow.
func (f *fixture) addedAgo(id string, ago time.Duration) {
	f.t.Helper()
	if _, err := f.s.DB.Exec(`UPDATE favourite_nodes SET created_at = ? WHERE id = ?`, db.FormatTime(recentNow.Add(-ago)), id); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) recent(folderID string, since time.Time, limit int) []string {
	f.t.Helper()
	links, err := f.s.RecentLinks(f.ctx, f.user, folderID, since, limit)
	if err != nil {
		f.t.Fatal(err)
	}
	titles := []string{}
	for _, l := range links {
		titles = append(titles, *l.Node.Title)
		if l.CreatedAt.IsZero() || l.Node.URL == nil || l.Node.Kind != KindLink {
			f.t.Errorf("link %+v", l)
		}
	}
	return titles
}

func TestRecentLinks(t *testing.T) {
	f := newFixture(t)
	day := 24 * time.Hour
	work := f.folder("", "Work")
	f.addedAgo(f.link(work, "Mail"), 3*day)
	tools := f.folder(work, "Tools")
	f.addedAgo(tools, 0)
	f.addedAgo(f.link(tools, "GitLab"), day)
	deep := f.folder(tools, "Deep")
	f.addedAgo(f.link(deep, "Grafana"), 10*day)
	f.addedAgo(f.link("", "Outside"), 0)

	if got, want := f.recent(work, time.Time{}, 10), []string{"GitLab", "Mail", "Grafana"}; !reflect.DeepEqual(got, want) {
		t.Errorf("all = %v, want %v", got, want)
	}
	if got, want := f.recent(work, time.Time{}, 2), []string{"GitLab", "Mail"}; !reflect.DeepEqual(got, want) {
		t.Errorf("limit = %v, want %v", got, want)
	}
	if got, want := f.recent(work, recentNow.Add(-7*day), 10), []string{"GitLab", "Mail"}; !reflect.DeepEqual(got, want) {
		t.Errorf("since = %v, want %v", got, want)
	}
	if got := f.recent(work, recentNow.Add(-time.Hour), 10); len(got) != 0 {
		t.Errorf("nothing recent = %v", got)
	}
	if got, want := f.recent(f.tree().ID, time.Time{}, 1), []string{"Outside"}; !reflect.DeepEqual(got, want) {
		t.Errorf("root = %v, want %v", got, want)
	}
}

func TestRecentLinksTieBreak(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A")
	b := f.link("", "B")
	f.addedAgo(a, time.Hour)
	f.addedAgo(b, time.Hour)
	want := []string{"A", "B"}
	if a < b {
		want = []string{"B", "A"}
	}
	if got := f.recent(f.tree().ID, time.Time{}, 10); !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestRecentLinksNotFound(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	link := f.link(work, "Mail")
	for name, c := range map[string]struct{ user, id string }{
		"other user": {f.other, work},
		"unknown":    {f.user, "nope"},
		"a link":     {f.user, link},
	} {
		if _, err := f.s.RecentLinks(f.ctx, c.user, c.id, time.Time{}, 10); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestRecentLinksImported(t *testing.T) {
	f := newFixture(t)
	old := time.Date(2024, 5, 1, 8, 0, 0, 0, time.UTC)
	f.link("", "New")
	f.importItems(f.tree().ID, ImportItem{Link: &ImportLink{URL: "https://example.org/old", Title: ptr("Old"), CreatedAt: old}})

	links, err := f.s.RecentLinks(f.ctx, f.user, f.tree().ID, time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || *links[0].Node.Title != "New" || *links[1].Node.Title != "Old" || !links[1].CreatedAt.Equal(old) {
		t.Errorf("links = %+v", links)
	}
	if got := f.recent(f.tree().ID, time.Now().Add(-7*24*time.Hour), 10); !reflect.DeepEqual(got, []string{"New"}) {
		t.Errorf("last 7 days = %v, want [New]", got)
	}
}

func TestTreeHasCreatedAt(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	mail := f.link(work, "Mail")
	f.addedAgo(mail, time.Hour)

	if n := find(f.tree(), mail); n.CreatedAt == nil || !n.CreatedAt.Equal(recentNow.Add(-time.Hour)) {
		t.Errorf("tree createdAt = %v", n.CreatedAt)
	}
	if n := find(f.tree(), work); n.CreatedAt == nil {
		t.Error("folder has no createdAt")
	}
	// The loaders that widgets use leave it out.
	sub, err := f.s.Subtree(f.ctx, f.user, work)
	if err != nil {
		t.Fatal(err)
	}
	if n := find(sub, mail); n.CreatedAt != nil {
		t.Errorf("subtree createdAt = %v", n.CreatedAt)
	}
}
