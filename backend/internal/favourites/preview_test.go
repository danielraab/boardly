package favourites

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"boardly/internal/jobs"
)

// scoped returns the user's node with the given ID as SubtreeNodes sees it.
func (f *fixture) scoped(id string) ScopedNode {
	f.t.Helper()
	nodes, err := f.s.SubtreeNodes(f.ctx, Scope{UserID: f.user, RootID: f.tree().ID})
	if err != nil {
		f.t.Fatal(err)
	}
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	f.t.Fatalf("node %s not in the subtree", id)
	return ScopedNode{}
}

func TestDescription(t *testing.T) {
	f := newFixture(t)
	n, err := f.s.Create(f.ctx, f.user, NewNode{Kind: KindLink, URL: ptr("https://example.org"), Description: ptr("  A tour  ")})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.scoped(n.ID).Description; got == nil || *got != "A tour" {
		t.Fatalf("description = %v, want %q", got, "A tour")
	}

	if _, err := f.s.Update(f.ctx, f.user, n.ID, Patch{Description: ptr("Notes")}); err != nil {
		t.Fatal(err)
	}
	if got := f.scoped(n.ID).Description; got == nil || *got != "Notes" {
		t.Fatalf("description = %v, want Notes", got)
	}
	// A patch without a description keeps it.
	if _, err := f.s.Update(f.ctx, f.user, n.ID, Patch{Title: ptr("T")}); err != nil {
		t.Fatal(err)
	}
	if got := f.scoped(n.ID).Description; got == nil || *got != "Notes" {
		t.Fatalf("description after title change = %v, want Notes", got)
	}
	if _, err := f.s.Update(f.ctx, f.user, n.ID, Patch{Description: ptr(" ")}); err != nil {
		t.Fatal(err)
	}
	if got := f.scoped(n.ID).Description; got != nil {
		t.Fatalf("description = %q, want none", *got)
	}

	long := strings.Repeat("ä", MaxDescriptionLength+1)
	if _, err := f.s.Update(f.ctx, f.user, n.ID, Patch{Description: &long}); !errors.Is(err, ErrInvalidDescription) {
		t.Fatalf("too long: err = %v, want ErrInvalidDescription", err)
	}
	if _, err := f.s.Create(f.ctx, f.user, NewNode{Kind: KindLink, URL: ptr("https://example.org"), Description: &long}); !errors.Is(err, ErrInvalidDescription) {
		t.Fatalf("too long on create: err = %v, want ErrInvalidDescription", err)
	}
	exact := strings.Repeat("ä", MaxDescriptionLength)
	if _, err := f.s.Update(f.ctx, f.user, n.ID, Patch{Description: &exact}); err != nil {
		t.Fatalf("2048 characters: %v", err)
	}

	folder := f.folder("", "Work")
	if _, err := f.s.Update(f.ctx, f.user, folder, Patch{Description: ptr("x")}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("folder description: err = %v, want ErrInvalidField", err)
	}
	if _, err := f.s.Create(f.ctx, f.user, NewNode{Kind: KindFolder, Title: ptr("F"), Description: ptr("x")}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("folder description on create: err = %v, want ErrInvalidField", err)
	}
}

func (f *fixture) setPreview(id string) {
	f.t.Helper()
	if _, err := f.s.DB.Exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at)
		VALUES ('img-' || ?, 'preview', ?, 'image/jpeg', 10, '2026-01-01T00:00:00.000Z')`, id, f.user); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.s.DB.Exec(`UPDATE favourite_nodes SET preview_image_id = 'img-' || ? WHERE id = ?`, id, id); err != nil {
		f.t.Fatal(err)
	}
}

func TestURLChangeClearsPreview(t *testing.T) {
	f := newFixture(t)
	id := f.link("", "Docs")
	if _, err := f.s.Update(f.ctx, f.user, id, Patch{Description: ptr("Mine")}); err != nil {
		t.Fatal(err)
	}
	f.setPreview(id)
	if got := f.scoped(id).PreviewImageID; got == nil || *got != "img-"+id {
		t.Fatalf("preview = %v", got)
	}

	// Title only, or the same URL: the preview stays.
	for _, p := range []Patch{{Title: ptr("Documentation")}, {URL: ptr("https://example.org/Docs")}} {
		if _, err := f.s.Update(f.ctx, f.user, id, p); err != nil {
			t.Fatal(err)
		}
		if f.scoped(id).PreviewImageID == nil {
			t.Fatalf("patch %+v removed the preview", p)
		}
	}

	if _, err := f.s.Update(f.ctx, f.user, id, Patch{URL: ptr("https://example.org/other")}); err != nil {
		t.Fatal(err)
	}
	n := f.scoped(id)
	if n.PreviewImageID != nil {
		t.Fatalf("preview after URL change = %q, want none", *n.PreviewImageID)
	}
	if n.Description == nil || *n.Description != "Mine" {
		t.Fatalf("description after URL change = %v, want Mine", n.Description)
	}
}

// queued returns the pending jobs as "kind key url".
func queued(f *fixture) []string {
	f.t.Helper()
	rows, err := f.s.DB.Query(`SELECT kind, COALESCE(dedup_key, ''), payload FROM jobs WHERE started_at IS NULL ORDER BY id`)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var kind, key, raw string
		if err := rows.Scan(&kind, &key, &raw); err != nil {
			f.t.Fatal(err)
		}
		var p jobs.PreviewFetchPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			f.t.Fatal(err)
		}
		if kind == string(jobs.KindPreviewFetch) && (p.UserID != f.user || p.NodeID != key) {
			f.t.Fatalf("preview fetch payload %s doesn't match user %s and key %s", raw, f.user, key)
		}
		out = append(out, kind+" "+key+" "+p.URL)
	}
	return out
}

func TestQueuesFetches(t *testing.T) {
	f := newFixture(t)
	f.s.Jobs = jobs.New(f.s.DB)

	folder := f.folder("", "Work")
	if got := queued(f); len(got) != 0 {
		t.Fatalf("jobs after folder create = %v", got)
	}

	id := f.link(folder, "Docs")
	want := []string{
		"favicon https://example.org https://example.org/Docs",
		"preview_fetch " + id + " https://example.org/Docs",
	}
	if got := queued(f); !equal(got, want) {
		t.Fatalf("jobs after create = %v, want %v", got, want)
	}

	// No URL change, no jobs.
	if _, err := f.s.Update(f.ctx, f.user, id, Patch{Title: ptr("Documentation"), URL: ptr("https://example.org/Docs")}); err != nil {
		t.Fatal(err)
	}
	if got := queued(f); !equal(got, want) {
		t.Fatalf("jobs after title change = %v, want %v", got, want)
	}

	// A new URL replaces the pending preview fetch and adds the new origin.
	if _, err := f.s.Update(f.ctx, f.user, id, Patch{URL: ptr("https://go.dev/tour")}); err != nil {
		t.Fatal(err)
	}
	want = []string{
		"favicon https://example.org https://example.org/Docs",
		"preview_fetch " + id + " https://go.dev/tour",
		"favicon https://go.dev https://go.dev/tour",
	}
	if got := queued(f); !equal(got, want) {
		t.Fatalf("jobs after URL change = %v, want %v", got, want)
	}

	// A failed create queues nothing.
	if _, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: "missing", Kind: KindLink, URL: ptr("https://new.example")}); err == nil {
		t.Fatal("create in a missing folder succeeded")
	}
	if got := queued(f); !equal(got, want) {
		t.Fatalf("jobs after failed create = %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Widgets serve subtrees on public dashboards: they never carry the
// owner's description or preview.
func TestSubtreeJSONHasNoPreviewFields(t *testing.T) {
	f := newFixture(t)
	folder := f.folder("", "Work")
	id := f.link(folder, "Docs")
	if _, err := f.s.Update(f.ctx, f.user, id, Patch{Description: ptr("Mine")}); err != nil {
		t.Fatal(err)
	}
	f.setPreview(id)
	sub, err := f.s.Subtree(f.ctx, f.user, folder)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"description", "Mine", "preview", "img-"} {
		if strings.Contains(string(data), field) {
			t.Fatalf("subtree JSON %s contains %q", data, field)
		}
	}
}

func TestTreeHasDescriptionAndPreview(t *testing.T) {
	f := newFixture(t)
	folder := f.folder("", "Work")
	id := f.link(folder, "Docs")
	if _, err := f.s.Update(f.ctx, f.user, id, Patch{Description: ptr("Notes")}); err != nil {
		t.Fatal(err)
	}
	f.setPreview(id)

	tree := f.tree()
	link := find(tree, id)
	if link.Description == nil || *link.Description != "Notes" {
		t.Fatalf("description = %v, want Notes", link.Description)
	}
	if link.PreviewImageID == nil || *link.PreviewImageID != "img-"+id {
		t.Fatalf("preview = %v, want img-%s", link.PreviewImageID, id)
	}
	if n := find(tree, folder); n.Description != nil || n.PreviewImageID != nil {
		t.Fatalf("folder has description %v, preview %v", n.Description, n.PreviewImageID)
	}

	// The loaders that widgets use leave both out.
	sub, err := f.s.Subtree(f.ctx, f.user, folder)
	if err != nil {
		t.Fatal(err)
	}
	if n := find(sub, id); n.Description != nil || n.PreviewImageID != nil {
		t.Fatalf("subtree link has description %v, preview %v", n.Description, n.PreviewImageID)
	}
	l, err := f.s.Link(f.ctx, f.user, id)
	if err != nil {
		t.Fatal(err)
	}
	if l.Description != nil || l.PreviewImageID != nil {
		t.Fatalf("Link has description %v, preview %v", l.Description, l.PreviewImageID)
	}
}

func TestRemovePreview(t *testing.T) {
	f := newFixture(t)
	f.s.Jobs = jobs.New(f.s.DB)
	id := f.link("", "Docs")
	f.setPreview(id)
	before := queued(f)

	// A title change keeps the preview and returns it.
	n, err := f.s.Update(f.ctx, f.user, id, Patch{Title: ptr("Documentation")})
	if err != nil {
		t.Fatal(err)
	}
	if n.PreviewImageID == nil || *n.PreviewImageID != "img-"+id {
		t.Fatalf("preview after title change = %v", n.PreviewImageID)
	}

	n, err = f.s.Update(f.ctx, f.user, id, Patch{RemovePreview: true})
	if err != nil {
		t.Fatal(err)
	}
	if n.PreviewImageID != nil || f.scoped(id).PreviewImageID != nil {
		t.Fatal("preview not removed")
	}
	if got := queued(f); !equal(got, before) {
		t.Fatalf("jobs after removing the preview = %v, want %v", got, before)
	}

	folder := f.folder("", "Work")
	if _, err := f.s.Update(f.ctx, f.user, folder, Patch{RemovePreview: true}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("folder: err = %v, want ErrInvalidField", err)
	}
}
