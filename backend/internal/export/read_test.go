package export

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"boardly/internal/blobs"
	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/notes"
	"boardly/internal/users"
)

type fixture struct {
	t   *testing.T
	ctx context.Context
	db  *sql.DB
	now time.Time
	s   *Service
	ann users.User
	bob users.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	f := &fixture{t: t, ctx: context.Background(), db: database, now: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)}
	us := users.NewStore(database)
	us.Now = func() time.Time { return f.now }
	var err error
	if f.ann, err = us.EnsureByEmail(f.ctx, database, "ann@example.org"); err != nil {
		t.Fatal(err)
	}
	if f.bob, err = us.EnsureByEmail(f.ctx, database, "bob@example.org"); err != nil {
		t.Fatal(err)
	}
	imageDir, err := blobs.Open(filepath.Join(t.TempDir(), "images"))
	if err != nil {
		t.Fatal(err)
	}
	attachmentDir, err := blobs.Open(filepath.Join(t.TempDir(), "attachments"))
	if err != nil {
		t.Fatal(err)
	}
	offlineDir, err := blobs.Open(filepath.Join(t.TempDir(), "offline"))
	if err != nil {
		t.Fatal(err)
	}
	f.s = &Service{DB: database, ImageFiles: imageDir, AttachmentFiles: attachmentDir, OfflineFiles: offlineDir, Now: func() time.Time { return f.now }}
	return f
}

func (f *fixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.ExecContext(f.ctx, query, args...); err != nil {
		f.t.Fatalf("%s: %v", query, err)
	}
}

func ptr[T any](v T) *T { return &v }

// seeded holds the IDs of what seed created for Ann.
type seeded struct {
	folder, link, dashboard, category, note, archived, image, preview, attachment, deleted string
}

// seed gives Ann (and, to check isolation, Bob) one of everything.
func (f *fixture) seed() seeded {
	f.t.Helper()
	ctx, ann, bob := f.ctx, f.ann.ID, f.bob.ID
	now := db.FormatTime(f.now)
	var s seeded

	favs := favourites.NewStore(f.db)
	folder, err := favs.Create(ctx, ann, favourites.NewNode{Kind: favourites.KindFolder, Title: ptr("Dev")})
	if err != nil {
		f.t.Fatal(err)
	}
	s.folder = folder.ID
	link, err := favs.Create(ctx, ann, favourites.NewNode{ParentID: folder.ID, Kind: favourites.KindLink, URL: ptr("https://go.dev"), Title: ptr("Go"), Tags: []string{"go", "lang"}})
	if err != nil {
		f.t.Fatal(err)
	}
	s.link = link.ID
	if _, err := favs.Create(ctx, ann, favourites.NewNode{ParentID: folder.ID, Kind: favourites.KindLink, URL: ptr("https://example.org/second")}); err != nil {
		f.t.Fatal(err)
	}
	if _, err := favs.Create(ctx, bob, favourites.NewNode{Kind: favourites.KindLink, URL: ptr("https://bob.example.org"), Tags: []string{"bob-tag"}}); err != nil {
		f.t.Fatal(err)
	}
	f.exec(`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES ('unused', ?, 'Unused', 'unused', ?)`, ann, now)
	f.exec(`UPDATE users SET slug = 'ann', share_folder_id = ? WHERE id = ?`, folder.ID, ann)

	dash := dashboards.NewStore(f.db)
	d, err := dash.Create(ctx, ann, dashboards.Fields{Title: "Home", Slug: "home", IsPublic: true})
	if err != nil {
		f.t.Fatal(err)
	}
	s.dashboard = d.ID
	if _, err := dash.Create(ctx, bob, dashboards.Fields{Title: "Bob's", Slug: "bobs"}); err != nil {
		f.t.Fatal(err)
	}
	f.exec(`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config) VALUES ('s1', ?, NULL, 'section', 0, 6, '{"title":"Top"}')`, d.ID)
	f.exec(`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config) VALUES ('w2', ?, 's1', 'link-list', 1, 2, json_object('folderId', ?))`, d.ID, folder.ID)
	f.exec(`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config) VALUES ('w1', ?, 's1', 'note', 0, 2, '{}')`, d.ID)

	s.image, s.preview, s.attachment, s.deleted = blobs.NewID(), blobs.NewID(), blobs.NewID(), blobs.NewID()
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 3, ?)`, s.image, ann, now)
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'preview', ?, 'image/webp', 4, ?)`, s.preview, ann, now)
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'favicon', NULL, 'image/png', 3, ?)`, blobs.NewID(), now)
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 3, ?)`, blobs.NewID(), bob, now)
	f.exec(`UPDATE favourite_nodes SET preview_image_id = ? WHERE id = ?`, s.preview, link.ID)

	ns := notes.NewStore(f.db)
	cat, err := ns.CreateCategory(ctx, ann, notes.NewCategory{Title: "Work", Icon: &notes.Icon{Type: notes.IconImage, ImageID: s.image}, Color: ptr("blue")})
	if err != nil {
		f.t.Fatal(err)
	}
	s.category = cat.ID
	f.exec(`UPDATE users SET note_category_id = ? WHERE id = ?`, cat.ID, ann)
	note, err := ns.CreateNote(ctx, ann, notes.NewNote{Title: ptr("Plan"), CategoryID: &cat.ID, Pinned: true})
	if err != nil {
		f.t.Fatal(err)
	}
	s.note = note.ID
	archived, err := ns.CreateNote(ctx, ann, notes.NewNote{Title: ptr("Old"), Archived: true})
	if err != nil {
		f.t.Fatal(err)
	}
	s.archived = archived.ID
	if _, err := ns.CreateNote(ctx, bob, notes.NewNote{Title: ptr("Bob's note")}); err != nil {
		f.t.Fatal(err)
	}
	f.exec(`INSERT INTO attachments (id, user_id, name, content_type, size, created_at) VALUES (?, ?, 'invoice.pdf', 'application/pdf', 3, ?)`, s.attachment, ann, now)
	f.exec(`INSERT INTO attachments (id, user_id, name, content_type, size, created_at, deleted_at) VALUES (?, ?, 'gone.pdf', 'application/pdf', 3, ?, ?)`, s.deleted, ann, now, now)
	f.exec(`INSERT INTO note_attachments (note_id, attachment_id, position) VALUES (?, ?, 0)`, note.ID, s.attachment)

	f.exec(`INSERT INTO media_sources (id, user_id, kind, title, url, icon_symbol, position, created_at, updated_at)
		VALUES ('m1', ?, 'stream', 'Radio', 'https://radio.example.org', 'radio', 0, ?, ?)`, ann, now, now)
	f.exec(`INSERT INTO media_sources (id, user_id, kind, title, url, position, created_at, updated_at)
		VALUES ('m2', ?, 'stream', 'Bob FM', 'https://bob.example.org', 0, ?, ?)`, bob, now, now)
	return s
}

func TestDocument(t *testing.T) {
	f := newFixture(t)
	s := f.seed()
	doc, err := f.s.Document(f.ctx, f.ann.ID)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Format != "boardly" || doc.Version != 2 || doc.ExportedAt != "2026-11-02T10:00:00.000Z" || doc.WithFiles {
		t.Errorf("header = %q %d %q %v", doc.Format, doc.Version, doc.ExportedAt, doc.WithFiles)
	}
	p := doc.Profile
	if p.Email != "ann@example.org" || p.Slug == nil || *p.Slug != "ann" ||
		p.DefaultFolderID == nil || *p.DefaultFolderID != s.folder ||
		p.DefaultNoteCategoryID == nil || *p.DefaultNoteCategoryID != s.category {
		t.Errorf("profile = %+v", p)
	}

	// Tree: root → Dev → [Go, second], in order, with tags and preview.
	root := doc.Favourites
	if root == nil || root.Kind != "folder" || len(root.Children) != 1 {
		t.Fatalf("root = %+v", root)
	}
	dev := root.Children[0]
	if dev.ID != s.folder || len(dev.Children) != 2 {
		t.Fatalf("Dev = %+v", dev)
	}
	goLink := dev.Children[0]
	if goLink.ID != s.link || goLink.URL != "https://go.dev/" && goLink.URL != "https://go.dev" ||
		!slices.Equal(goLink.Tags, []string{"go", "lang"}) || goLink.PreviewImageID == nil || *goLink.PreviewImageID != s.preview {
		t.Errorf("Go link = %+v", goLink)
	}
	if dev.Children[1].URL != "https://example.org/second" {
		t.Errorf("second link = %+v", dev.Children[1])
	}

	var tagNames []string
	for _, tag := range doc.Tags {
		tagNames = append(tagNames, tag.Name)
	}
	if !slices.Equal(tagNames, []string{"go", "lang", "Unused"}) {
		t.Errorf("tags = %v", tagNames)
	}

	// Dashboard with the section and its two widgets in position order.
	if len(doc.Dashboards) != 1 || doc.Dashboards[0].ID != s.dashboard || !doc.Dashboards[0].IsPublic {
		t.Fatalf("dashboards = %+v", doc.Dashboards)
	}
	widgets := doc.Dashboards[0].Widgets
	if len(widgets) != 1 || widgets[0].ID != "s1" || len(widgets[0].Children) != 2 ||
		widgets[0].Children[0].ID != "w1" || widgets[0].Children[1].ID != "w2" {
		t.Fatalf("widgets = %+v", widgets)
	}
	var cfg struct {
		FolderID string `json:"folderId"`
	}
	if err := json.Unmarshal(widgets[0].Children[1].Config, &cfg); err != nil || cfg.FolderID != s.folder {
		t.Errorf("widget config = %s, want the folder ID of Dev", widgets[0].Children[1].Config)
	}

	if len(doc.NoteCategories) != 1 || doc.NoteCategories[0].Icon == nil || doc.NoteCategories[0].Icon.ImageID != s.image {
		t.Errorf("categories = %+v", doc.NoteCategories)
	}
	if len(doc.Notes) != 2 {
		t.Fatalf("notes = %+v", doc.Notes)
	}
	byID := map[string]Note{}
	for _, n := range doc.Notes {
		byID[n.ID] = n
	}
	plan := byID[s.note]
	if !plan.Pinned || plan.CategoryID == nil || *plan.CategoryID != s.category ||
		!slices.Equal(plan.AttachmentIDs, []string{s.attachment}) || !json.Valid(plan.Content) {
		t.Errorf("note Plan = %+v", plan)
	}
	if !byID[s.archived].Archived {
		t.Errorf("archived note = %+v", byID[s.archived])
	}

	if len(doc.MediaSources) != 1 || doc.MediaSources[0].Icon == nil || doc.MediaSources[0].Icon.Name != "radio" {
		t.Errorf("media sources = %+v", doc.MediaSources)
	}
	var imageIDs []string
	for _, img := range doc.Images {
		imageIDs = append(imageIDs, img.ID)
		if img.File != nil {
			t.Errorf("image %s has a file without files", img.ID)
		}
	}
	slices.Sort(imageIDs)
	want := []string{s.image, s.preview}
	slices.Sort(want)
	if !slices.Equal(imageIDs, want) {
		t.Errorf("images = %v, want upload and preview %v", imageIDs, want)
	}
	if len(doc.Attachments) != 1 || doc.Attachments[0].ID != s.attachment || doc.Attachments[0].Name != "invoice.pdf" || doc.Attachments[0].File != nil {
		t.Errorf("attachments = %+v, want only invoice.pdf without file", doc.Attachments)
	}

	// Nothing of Bob's, in any field.
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, bobs := range []string{"bob", "Bob"} {
		if strings.Contains(string(raw), bobs) {
			t.Errorf("document contains %q: %s", bobs, raw)
		}
	}
}

func TestDocumentEmptyUser(t *testing.T) {
	f := newFixture(t)
	doc, err := f.s.Document(f.ctx, f.bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"tags", "dashboards", "noteCategories", "notes", "mediaSources", "images", "attachments"} {
		if !strings.Contains(string(raw), `"`+field+`":[]`) {
			t.Errorf("%s isn't an empty list: %s", field, raw)
		}
	}
}

func TestDocumentUnknownUser(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.Document(f.ctx, "nobody"); !errors.Is(err, ErrNoUser) {
		t.Fatalf("err = %v, want ErrNoUser", err)
	}
}

func TestDocumentRoundTrip(t *testing.T) {
	f := newFixture(t)
	f.seed()
	doc, err := f.s.Document(f.ctx, f.ann.ID)
	if err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var back Document
	if err := json.Unmarshal(first, &back); err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(&back)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("round trip changed the document:\n%s\n%s", first, second)
	}
}

func TestFeedsInDocument(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := database.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []string{"ann", "bob"} {
		exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, 'x', 'x')`, u, u+"@example.org")
	}
	for _, f := range []string{"f1", "f2", "f3"} {
		exec(`INSERT INTO feeds (id, url, title, next_fetch_at, created_at) VALUES (?, ?, ?, 'x', 'x')`, f, "https://a.example/"+f, "Title "+f)
	}
	for _, i := range [][2]string{{"i1", "f1"}, {"i2", "f1"}, {"i3", "f2"}, {"i4", "f3"}} {
		exec(`INSERT INTO feed_items (id, feed_id, identity, title, item_time, first_seen_at, audio_url) VALUES (?, ?, ?, ?, '2026-01-01T00:00:00.000Z', 'x', 'https://a.example/a.mp3')`, i[0], i[1], "g-"+i[0], "Item "+i[0])
	}
	exec(`INSERT INTO feed_subscriptions (id, user_id, feed_id, custom_title, position, created_at, updated_at) VALUES ('s1', 'ann', 'f1', 'Mine', 0, 'x', 'x')`)
	exec(`INSERT INTO feed_subscriptions (id, user_id, feed_id, position, created_at, updated_at) VALUES ('s9', 'bob', 'f3', 0, 'x', 'x')`)
	exec(`INSERT INTO feed_item_flags (user_id, item_id, liked_at) VALUES ('ann', 'i1', 'y'), ('ann', 'i3', 'y'), ('bob', 'i4', 'y')`)

	s := &Service{DB: database, Now: time.Now}
	doc, err := s.Document(ctx, "ann")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Feeds) != 2 || doc.Feeds[0].ID != "s1" || !doc.Feeds[0].Subscribed || *doc.Feeds[0].CustomTitle != "Mine" {
		t.Fatalf("feeds = %+v", doc.Feeds)
	}
	if doc.Feeds[1].ID != "f2" || doc.Feeds[1].Subscribed || doc.Feeds[1].Position != nil {
		t.Fatalf("saved feed = %+v", doc.Feeds[1])
	}
	if len(doc.FeedItems) != 2 {
		t.Fatalf("items = %+v", doc.FeedItems)
	}
	byGUID := map[string]FeedItem{}
	for _, it := range doc.FeedItems {
		byGUID[it.GUID] = it
	}
	if byGUID["g-i1"].FeedID != "s1" || byGUID["g-i3"].FeedID != "f2" || byGUID["g-i1"].Audio == nil {
		t.Fatalf("items = %+v", byGUID)
	}
}
