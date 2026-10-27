package favourites

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func (f *fixture) folderLinks(folderID string, subfolders bool) []string {
	f.t.Helper()
	links, err := f.s.FolderLinks(f.ctx, f.user, folderID, subfolders)
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

func TestFolderLinks(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	f.link(work, "Mail")
	tools := f.folder(work, "Tools")
	f.link(tools, "GitLab", "monitoring", "code")
	deep := f.folder(tools, "Deep")
	f.link(deep, "Grafana", "monitoring")
	f.link(work, "Wiki")
	f.link("", "Outside", "monitoring")

	if got, want := f.folderLinks(work, false), []string{"Mail", "Wiki"}; !reflect.DeepEqual(got, want) {
		t.Errorf("without subfolders = %v, want %v", got, want)
	}
	if got, want := f.folderLinks(work, true), []string{"Mail", "GitLab", "Grafana", "Wiki"}; !reflect.DeepEqual(got, want) {
		t.Errorf("with subfolders = %v, want %v", got, want)
	}
	if got := f.folderLinks(deep, false); !reflect.DeepEqual(got, []string{"Grafana"}) {
		t.Errorf("deep = %v", got)
	}

	monitoring, err := f.s.TagByName(f.ctx, f.user, "monitoring")
	if err != nil {
		t.Fatal(err)
	}
	code, err := f.s.TagByName(f.ctx, f.user, "code")
	if err != nil {
		t.Fatal(err)
	}
	links, err := f.s.FolderLinks(f.ctx, f.user, work, true)
	if err != nil {
		t.Fatal(err)
	}
	tags := map[string][]string{}
	for _, l := range links {
		tags[*l.Node.Title] = l.TagIDs
		if !l.CreatedAt.After(time.Time{}) {
			t.Errorf("%s has no creation time", *l.Node.Title)
		}
	}
	if got := tags["GitLab"]; len(got) != 2 || !slices.Contains(got, monitoring.ID) || !slices.Contains(got, code.ID) {
		t.Errorf("GitLab tags = %v", got)
	}
	if got := tags["Grafana"]; !reflect.DeepEqual(got, []string{monitoring.ID}) {
		t.Errorf("Grafana tags = %v", got)
	}
	if got := tags["Mail"]; len(got) != 0 {
		t.Errorf("Mail tags = %v", got)
	}

	// Unknown folders, links and other users' folders are not found.
	for name, id := range map[string]string{"unknown": "nope", "link": links[0].Node.ID} {
		if _, err := f.s.FolderLinks(f.ctx, f.user, id, true); err != ErrNotFound {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
	if _, err := f.s.FolderLinks(f.ctx, f.other, work, true); err != ErrNotFound {
		t.Errorf("other user: err = %v, want ErrNotFound", err)
	}
}

func TestTagExists(t *testing.T) {
	f := newFixture(t)
	f.link("", "Mail", "reading")
	tag, err := f.s.TagByName(f.ctx, f.user, "reading")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		user, id string
		want     bool
	}{{f.user, tag.ID, true}, {f.other, tag.ID, false}, {f.user, "nope", false}} {
		if got, err := f.s.TagExists(f.ctx, c.user, c.id); err != nil || got != c.want {
			t.Errorf("TagExists(%s, %s) = %v, %v, want %v", c.user, c.id, got, err, c.want)
		}
	}
}
