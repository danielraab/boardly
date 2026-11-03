package dataimport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"boardly/internal/attachments"
	"boardly/internal/blobs"
	"boardly/internal/dashboards"
	"boardly/internal/dbtest"
	"boardly/internal/export"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/media"
	"boardly/internal/notes"
	"boardly/internal/users"
	"boardly/internal/widgets"
)

var (
	pngData  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	pdfData  = []byte("%PDF-1.7\nhello")
	allParts = Parts{Favourites: true, Dashboards: true, Notes: true, Media: true}
)

type fixture struct {
	t        *testing.T
	ctx      context.Context
	db       *sql.DB
	users    *users.Store
	fav      *favourites.Store
	notes    *notes.Store
	dash     *dashboards.Store
	media    *media.Store
	images   *images.Store
	att      *attachments.Store
	export   *export.Service
	s        *Service
	ann, bob string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	ctx := context.Background()
	f := &fixture{t: t, ctx: ctx, db: database, users: users.NewStore(database)}
	for _, email := range []string{"ann@example.org", "bob@example.org"} {
		u, err := f.users.EnsureByEmail(ctx, database, email)
		if err != nil {
			t.Fatal(err)
		}
		if f.ann == "" {
			f.ann = u.ID
		} else {
			f.bob = u.ID
		}
	}
	var err error
	if f.images, err = images.NewStore(database, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if f.att, err = attachments.NewStore(database, t.TempDir(), 1<<20, attachments.DefaultAllowed(), 30); err != nil {
		t.Fatal(err)
	}
	f.fav, f.notes, f.dash = favourites.NewStore(database), notes.NewStore(database), dashboards.NewStore(database)
	noCheck := func(_ context.Context, u string) (string, error) { return u, nil }
	noLogo := func(context.Context, string, string) (string, error) { return "", errors.New("no logos") }
	f.media = media.NewStore(database, noCheck, noLogo)
	widgets.Register(f.dash, f.fav, f.notes, f.media, images.NewFavicons(f.images, false))
	f.export = &export.Service{DB: database, ImageFiles: blobs.Dir(f.images.Dir), AttachmentFiles: f.att.Files, Now: time.Now}
	f.s = &Service{DB: database, Favourites: f.fav, Notes: f.notes, Dashboards: f.dash, Media: f.media, Images: f.images, Attachments: f.att}
	return f
}

// seeded holds what seed created for Ann.
type seeded struct {
	dev, link, category, note, station, dashboard, icon, attachment string
}

// seed gives Ann one of everything, with references between the parts.
func (f *fixture) seed() seeded {
	t, ctx, ann := f.t, f.ctx, f.ann
	t.Helper()
	var s seeded
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	root, err := f.fav.Tree(ctx, ann)
	must(err)
	dev, err := f.fav.Create(ctx, ann, favourites.NewNode{ParentID: root.ID, Kind: favourites.KindFolder, Title: ptr("Dev")})
	must(err)
	s.dev = dev.ID
	link, err := f.fav.Create(ctx, ann, favourites.NewNode{ParentID: dev.ID, Kind: favourites.KindLink, URL: ptr("https://go.dev"), Title: ptr("Go"), Tags: []string{"go"}})
	must(err)
	s.link = link.ID

	icon, err := f.images.Save(ctx, images.SourceUpload, ann, pngData)
	must(err)
	s.icon = icon.ID
	cat, err := f.notes.CreateCategory(ctx, ann, notes.NewCategory{Title: "Work", Icon: &notes.Icon{Type: notes.IconImage, ImageID: icon.ID}})
	must(err)
	s.category = cat.ID
	att, err := f.att.Create(ctx, ann, "invoice.pdf", pdfData)
	must(err)
	s.attachment = att.ID
	doc := notes.FromText("pay it")
	doc.Blocks = append(doc.Blocks, notes.Block{Type: notes.BlockAttachment, ID: att.ID})
	content, err := json.Marshal(doc)
	must(err)
	note, err := f.notes.CreateNote(ctx, ann, notes.NewNote{Title: ptr("Plan"), CategoryID: &cat.ID, Content: content, Pinned: true})
	must(err)
	s.note = note.ID
	_, err = f.db.ExecContext(ctx, `UPDATE users SET share_folder_id = ?, note_category_id = ? WHERE id = ?`, dev.ID, cat.ID, ann)
	must(err)

	station, err := f.media.Create(ctx, ann, media.NewSource{Kind: media.KindStream, Title: "Radio", URL: "https://radio.example.org/stream"})
	must(err)
	s.station = station.ID

	d, err := f.dash.Create(ctx, ann, dashboards.Fields{Title: "Home", Slug: "home", IsDefault: true, IsPublic: true})
	must(err)
	s.dashboard = d.ID
	section := dashboards.Widget{Kind: dashboards.KindSection, Width: 6, Children: []dashboards.Widget{
		{Kind: widgets.KindLinkList, Width: 2, Config: json.RawMessage(`{"folderId":"` + dev.ID + `","size":"M","layout":"list"}`)},
		{Kind: widgets.KindNote, Width: 2, Config: json.RawMessage(`{"noteId":"` + note.ID + `","size":"M"}`)},
	}}
	_, err = f.dash.SaveLayout(ctx, ann, d.ID, 0, []dashboards.Widget{section})
	must(err)
	return s
}

// exportOf returns the user's export, as JSON or as a ZIP with files.
func (f *fixture) exportOf(userID string, withFiles bool) []byte {
	f.t.Helper()
	doc, err := f.export.Document(f.ctx, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	var buf bytes.Buffer
	if withFiles {
		err = f.export.WriteArchive(&buf, doc)
	} else {
		err = export.WriteJSON(&buf, doc)
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return buf.Bytes()
}

func (f *fixture) root(userID string) string {
	f.t.Helper()
	root, err := f.fav.Tree(f.ctx, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	return root.ID
}

func (f *fixture) importInto(userID string, data []byte, opts Options) Report {
	f.t.Helper()
	report, err := f.s.Import(f.ctx, userID, data, opts)
	if err != nil {
		f.t.Fatalf("import: %v", err)
	}
	return report
}

func (f *fixture) count(query string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRowContext(f.ctx, query, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

// widgetConfigs returns the content widget configs of the user's dashboard by kind.
func (f *fixture) widgetConfigs(dashboardID string) map[string]map[string]any {
	f.t.Helper()
	rows, err := f.db.QueryContext(f.ctx, `SELECT kind, config FROM dashboard_widgets WHERE dashboard_id = ? AND kind <> 'section'`, dashboardID)
	if err != nil {
		f.t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]map[string]any{}
	for rows.Next() {
		var kind, raw string
		if err := rows.Scan(&kind, &raw); err != nil {
			f.t.Fatal(err)
		}
		var stored struct {
			Config map[string]any `json:"config"`
		}
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			f.t.Fatal(err)
		}
		out[kind] = stored.Config
	}
	return out
}

func TestImportIntoNewAccount(t *testing.T) {
	f := newFixture(t)
	f.seed()
	report := f.importInto(f.bob, f.exportOf(f.ann, true), Options{Parts: allParts, FolderID: f.root(f.bob), SkipDuplicates: true})

	if report.Favourites == nil || report.Favourites.Links != 1 || report.Favourites.Folders != 1 || report.Favourites.Tags != 1 {
		t.Errorf("favourites report = %+v", report.Favourites)
	}
	if report.Notes == nil || report.Notes.Notes != 1 || report.Notes.Categories != 1 {
		t.Errorf("notes report = %+v", report.Notes)
	}
	if report.Media == nil || report.Media.Sources != 1 || report.Dashboards == nil || report.Dashboards.Dashboards != 1 || report.Dashboards.SkippedWidgets != 0 {
		t.Errorf("media / dashboards report = %+v / %+v", report.Media, report.Dashboards)
	}
	if report.Files != (FilesReport{Images: 1, Attachments: 1}) {
		t.Errorf("files report = %+v", report.Files)
	}

	// The dashboard: not public, not default, widgets pointing at Bob's items.
	var dashID string
	var public, def bool
	if err := f.db.QueryRowContext(f.ctx, `SELECT id, is_public, is_default FROM dashboards WHERE user_id = ?`, f.bob).Scan(&dashID, &public, &def); err != nil {
		t.Fatal(err)
	}
	if public || def {
		t.Errorf("imported dashboard public %v, default %v", public, def)
	}
	configs := f.widgetConfigs(dashID)
	var bobDev, bobNote string
	if err := f.db.QueryRowContext(f.ctx, `SELECT id FROM favourite_nodes WHERE user_id = ? AND title = 'Dev'`, f.bob).Scan(&bobDev); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT id FROM notes WHERE user_id = ? AND title = 'Plan'`, f.bob).Scan(&bobNote); err != nil {
		t.Fatal(err)
	}
	if configs[widgets.KindLinkList]["folderId"] != bobDev || configs[widgets.KindNote]["noteId"] != bobNote {
		t.Errorf("widget configs = %v, want folder %s and note %s", configs, bobDev, bobNote)
	}

	// The note: pinned, in Bob's "Work" with the imported icon, with the imported attachment.
	n, err := f.notes.Note(f.ctx, f.bob, bobNote)
	if err != nil {
		t.Fatal(err)
	}
	if !n.Pinned || len(n.Attachments) != 1 || n.Attachments[0].Name != "invoice.pdf" {
		t.Errorf("note = %+v", n)
	}
	if f.count(`SELECT COUNT(*) FROM note_categories WHERE user_id = ? AND icon_image_id IN (SELECT id FROM images WHERE user_id = ?)`, f.bob, f.bob) != 1 {
		t.Error("category icon isn't one of Bob's images")
	}

	// Bob had no defaults, so they come from the file.
	u, err := f.users.Get(f.ctx, f.bob)
	if err != nil {
		t.Fatal(err)
	}
	if u.ShareFolderID == nil || *u.ShareFolderID != bobDev || u.NoteCategoryID == nil {
		t.Errorf("defaults = %v / %v", u.ShareFolderID, u.NoteCategoryID)
	}
}

func TestImportOwnExportAgain(t *testing.T) {
	f := newFixture(t)
	s := f.seed()
	before := map[string]int{}
	tables := []string{"favourite_nodes", "notes", "note_categories", "media_sources", "images", "attachments", "dashboards"}
	for _, table := range tables {
		before[table] = f.count(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, f.ann)
	}

	report := f.importInto(f.ann, f.exportOf(f.ann, true), Options{Parts: allParts, FolderID: f.root(f.ann), SkipDuplicates: true})
	if report.Favourites.Links != 0 || report.Favourites.Skipped.Duplicates != 1 || report.Notes.Notes != 0 || report.Notes.DuplicateNotes != 1 ||
		report.Media.Sources != 0 || report.Media.Duplicates != 1 || report.Files.Images != 0 || report.Files.Attachments != 0 {
		t.Errorf("report = %+v %+v %+v %+v", report.Favourites, report.Notes, report.Media, report.Files)
	}
	for _, table := range tables {
		want := before[table]
		if table == "dashboards" {
			want++
		}
		if got := f.count(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, f.ann); got != want {
			t.Errorf("%s: %d rows, want %d", table, got, want)
		}
	}

	// The new dashboard's widgets point at Ann's existing folder and note.
	var dashID, slug string
	if err := f.db.QueryRowContext(f.ctx, `SELECT id, slug FROM dashboards WHERE user_id = ? AND id <> ?`, f.ann, s.dashboard).Scan(&dashID, &slug); err != nil {
		t.Fatal(err)
	}
	if slug != "home-2" {
		t.Errorf("slug = %q, want home-2", slug)
	}
	configs := f.widgetConfigs(dashID)
	if configs[widgets.KindLinkList]["folderId"] != s.dev || configs[widgets.KindNote]["noteId"] != s.note {
		t.Errorf("widget configs = %v", configs)
	}
}

func TestImportDashboardsOnlyKeepsReferences(t *testing.T) {
	f := newFixture(t)
	s := f.seed()
	report := f.importInto(f.ann, f.exportOf(f.ann, false), Options{Parts: Parts{Dashboards: true}})
	if report.Favourites != nil || report.Notes != nil || report.Media != nil || report.Dashboards.Dashboards != 1 {
		t.Errorf("report = %+v", report)
	}
	var dashID string
	if err := f.db.QueryRowContext(f.ctx, `SELECT id FROM dashboards WHERE user_id = ? AND id <> ?`, f.ann, s.dashboard).Scan(&dashID); err != nil {
		t.Fatal(err)
	}
	if configs := f.widgetConfigs(dashID); configs[widgets.KindLinkList]["folderId"] != s.dev {
		t.Errorf("widget lost its folder: %v", configs)
	}
}

func TestImportJSONDropsFiles(t *testing.T) {
	f := newFixture(t)
	f.seed()
	report := f.importInto(f.bob, f.exportOf(f.ann, false), Options{Parts: Parts{Notes: true}})
	if report.Files.Images != 0 || report.Files.Attachments != 0 || report.Files.Skipped != 2 {
		t.Errorf("files = %+v, want the icon and the attachment skipped", report.Files)
	}
	var id string
	var iconImage *string
	if err := f.db.QueryRowContext(f.ctx, `SELECT id, (SELECT icon_image_id FROM note_categories WHERE user_id = ?) FROM notes WHERE user_id = ?`, f.bob, f.bob).Scan(&id, &iconImage); err != nil {
		t.Fatal(err)
	}
	if iconImage != nil {
		t.Errorf("category kept the icon %s without its image", *iconImage)
	}
	n, err := f.notes.Note(f.ctx, f.bob, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Attachments) != 0 {
		t.Errorf("note kept attachments %+v", n.Attachments)
	}
}

func TestImportOnlyUsedImages(t *testing.T) {
	f := newFixture(t)
	f.seed()
	report := f.importInto(f.bob, f.exportOf(f.ann, true), Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.bob)})
	if report.Files.Images != 0 || report.Files.Attachments != 0 {
		t.Errorf("files = %+v, want none for favourites without previews", report.Files)
	}
	if f.count(`SELECT COUNT(*) FROM images WHERE user_id = ?`, f.bob) != 0 {
		t.Error("Bob got images")
	}
}

func TestImportDisallowedAttachment(t *testing.T) {
	f := newFixture(t)
	f.seed()
	data := f.exportOf(f.ann, true)
	f.att.Allowed = []string{"image/png"}
	report := f.importInto(f.bob, data, Options{Parts: Parts{Notes: true}})
	if report.Files.Attachments != 0 || report.Files.Skipped != 1 || report.Notes.Notes != 1 {
		t.Errorf("report = %+v / %+v", report.Files, report.Notes)
	}
}

func TestImportExistingDefaultsKept(t *testing.T) {
	f := newFixture(t)
	f.seed()
	bobRoot := f.root(f.bob)
	other, err := f.fav.Create(f.ctx, f.bob, favourites.NewNode{ParentID: bobRoot, Kind: favourites.KindFolder, Title: ptr("Mine")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE users SET share_folder_id = ? WHERE id = ?`, other.ID, f.bob); err != nil {
		t.Fatal(err)
	}
	f.importInto(f.bob, f.exportOf(f.ann, false), Options{Parts: Parts{Favourites: true}, FolderID: bobRoot})
	u, err := f.users.Get(f.ctx, f.bob)
	if err != nil {
		t.Fatal(err)
	}
	if u.ShareFolderID == nil || *u.ShareFolderID != other.ID {
		t.Errorf("default folder changed to %v", u.ShareFolderID)
	}
}

func TestImportErrorsAddNothing(t *testing.T) {
	f := newFixture(t)
	f.seed()
	data := f.exportOf(f.ann, true)
	annRoot := f.root(f.ann)

	if _, err := f.s.Import(f.ctx, f.bob, data, Options{Parts: allParts, FolderID: annRoot}); !errors.Is(err, ErrInvalidFolder) {
		t.Errorf("other user's folder: %v", err)
	}
	if _, err := f.s.Import(f.ctx, f.bob, data, Options{Parts: Parts{Notes: true}, CategoryID: ptr("nope")}); !errors.Is(err, ErrInvalidCategory) {
		t.Errorf("unknown category: %v", err)
	}
	// A note title that's too long fails in the transaction after images and
	// favourites were written.
	var doc map[string]any
	if err := json.Unmarshal(f.exportOf(f.ann, false), &doc); err != nil {
		t.Fatal(err)
	}
	long := make([]byte, 500)
	for i := range long {
		long[i] = 'x'
	}
	doc["notes"].([]any)[0].(map[string]any)["title"] = string(long)
	broken, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Import(f.ctx, f.bob, broken, Options{Parts: allParts, FolderID: f.root(f.bob)}); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("too long title: %v", err)
	}
	for _, table := range []string{"notes", "media_sources", "dashboards", "images", "attachments", "tags"} {
		if n := f.count(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, f.bob); n != 0 {
			t.Errorf("%s: %d rows after failed imports", table, n)
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM favourite_nodes WHERE user_id = ? AND parent_id IS NOT NULL`, f.bob); n != 0 {
		t.Errorf("favourites: %d nodes after failed imports", n)
	}
}

func TestParseParts(t *testing.T) {
	if p, ok := ParseParts("favourites, notes"); !ok || p != (Parts{Favourites: true, Notes: true}) {
		t.Errorf("ParseParts = %+v, %v", p, ok)
	}
	for _, s := range []string{"", " ", "favourites,passkeys", "favourites,"} {
		if _, ok := ParseParts(s); ok {
			t.Errorf("ParseParts(%q) accepted", s)
		}
	}
}
