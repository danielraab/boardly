package favourites

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"boardly/internal/db"
	"boardly/internal/jobs"
)

func importLink(url string, tags ...string) ImportItem {
	return ImportItem{Link: &ImportLink{URL: url, Title: ptr(url), Tags: tags}}
}

func importFolder(name string, items ...ImportItem) ImportItem {
	return ImportItem{Folder: &ImportFolder{Name: name, Items: items}}
}

func (f *fixture) importItems(folderID string, items ...ImportItem) ImportReport {
	f.t.Helper()
	return f.importItemsWith(true, folderID, items...)
}

func (f *fixture) importItemsWith(skipDuplicates bool, folderID string, items ...ImportItem) ImportReport {
	f.t.Helper()
	report, err := f.s.Import(f.ctx, f.user, folderID, ImportTree{Root: ImportFolder{Items: items}}, skipDuplicates)
	if err != nil {
		f.t.Fatalf("import: %v", err)
	}
	return report
}

// child returns the ID of the child of parentID ("" for the root) with the title.
func (f *fixture) child(parentID, title string) string {
	f.t.Helper()
	for _, c := range find(f.tree(), parentID).Children {
		if c.Title != nil && *c.Title == title {
			return c.ID
		}
	}
	f.t.Fatalf("no child %q in %s", title, parentID)
	return ""
}

func TestImportIntoRootAndSubfolder(t *testing.T) {
	f := newFixture(t)
	report := f.importItems(f.tree().ID,
		importFolder("Dev", importFolder("Docker", importLink("https://docker.com")), importLink("https://go.dev")),
		importLink("https://example.org/a"),
	)
	if report.Links != 3 || report.Folders != 2 {
		t.Fatalf("report = %+v, want 3 links and 2 folders", report)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"Dev", "https://example.org/a"}) {
		t.Fatalf("root = %v", got)
	}
	dev := f.child("", "Dev")
	if got := f.titles(dev); !reflect.DeepEqual(got, []string{"Docker", "https://go.dev"}) {
		t.Fatalf("Dev = %v", got)
	}

	imported := f.folder("", "Imported")
	f.importItems(imported, importLink("https://example.org/b"))
	if got := f.titles(imported); !reflect.DeepEqual(got, []string{"https://example.org/b"}) {
		t.Fatalf("Imported = %v", got)
	}
}

func TestImportReusesFoldersIgnoringCase(t *testing.T) {
	f := newFixture(t)
	reading := f.folder("", "Reading")
	f.link("", "Home")

	report := f.importItems(f.tree().ID,
		importFolder("reading", importFolder("Later", importLink("https://example.org/1"))),
		importFolder("READING", importLink("https://example.org/2")),
		importFolder("New", importLink("https://example.org/3")),
		importFolder("new", importLink("https://example.org/4")),
		importFolder("AI"),
	)
	if report.Folders != 3 {
		t.Fatalf("created folders = %d, want 3 (Later, New, AI)", report.Folders)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"Reading", "Home", "New", "AI"}) {
		t.Fatalf("root = %v", got)
	}
	if got := f.titles(reading); !reflect.DeepEqual(got, []string{"Later", "https://example.org/2"}) {
		t.Fatalf("Reading = %v", got)
	}
	if got := f.titles(f.child("", "New")); !reflect.DeepEqual(got, []string{"https://example.org/3", "https://example.org/4"}) {
		t.Fatalf("New = %v", got)
	}
}

func TestImportSkipsDuplicates(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	if _, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: work, Kind: KindLink, URL: ptr("https://go.dev/"), Tags: []string{"work"}}); err != nil {
		t.Fatal(err)
	}

	report := f.importItems(f.tree().ID,
		importFolder("Dev", importLink("https://go.dev", "dev")),
		importLink("https://example.org/a"),
		importLink("https://example.org/a#again"),
		importLink("javascript:alert(1)"),
	)
	want := ImportReport{Links: 1, Folders: 1, Skipped: ImportSkipped{Duplicates: 2, InvalidURLs: 1}}
	if report != want {
		t.Fatalf("report = %+v, want %+v", report, want)
	}
	goLink := find(f.tree(), work).Children[0]
	if !reflect.DeepEqual(goLink.Tags, []string{"work"}) {
		t.Fatalf("existing link's tags = %v, want unchanged", goLink.Tags)
	}

	// The same items again add nothing.
	again := f.importItems(f.tree().ID, importLink("https://example.org/a"))
	if again.Links != 0 || again.Skipped.Duplicates != 1 {
		t.Fatalf("second import = %+v", again)
	}
}

func TestImportAllowsDuplicates(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	if _, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: work, Kind: KindLink, URL: ptr("https://go.dev/"), Tags: []string{"work"}}); err != nil {
		t.Fatal(err)
	}

	items := []ImportItem{
		importFolder("Dev", importLink("https://go.dev", "dev")),
		importLink("https://example.org/a"),
		importLink("https://example.org/a#again"),
	}
	report := f.importItemsWith(false, f.tree().ID, items...)
	if want := (ImportReport{Links: 3, Folders: 1}); report != want {
		t.Fatalf("report = %+v, want %+v", report, want)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"Work", "Dev", "https://example.org/a", "https://example.org/a#again"}) {
		t.Fatalf("root = %v", got)
	}
	goLink := find(f.tree(), work).Children[0]
	if !reflect.DeepEqual(goLink.Tags, []string{"work"}) {
		t.Fatalf("existing link's tags = %v, want unchanged", goLink.Tags)
	}

	// The same items again are imported again, into the reused folder.
	again := f.importItemsWith(false, f.tree().ID, items...)
	if want := (ImportReport{Links: 3}); again != want {
		t.Fatalf("second import = %+v, want %+v", again, want)
	}
	if got := f.titles(f.child("", "Dev")); !reflect.DeepEqual(got, []string{"https://go.dev", "https://go.dev"}) {
		t.Fatalf("Dev = %v", got)
	}
}

func TestImportKeepsDataAndCreationDate(t *testing.T) {
	f := newFixture(t)
	f.link("", "Existing", "Dev")
	created := time.Unix(1790365701, 0)
	f.importItems(f.tree().ID, ImportItem{Link: &ImportLink{
		URL: "https://go.dev", Title: ptr("Go"), Tags: []string{"dev", "go"}, Description: ptr("Note: read"), CreatedAt: created,
	}}, importLink("https://example.org/now"))

	id := f.child("", "Go")
	n := f.scoped(id)
	if !reflect.DeepEqual(n.Tags, []Tag{n.Tags[0], n.Tags[1]}) || n.Tags[0].Name != "Dev" || n.Tags[1].Name != "go" {
		t.Fatalf("tags = %+v, want the existing Dev and a new go", n.Tags)
	}
	if n.Description == nil || *n.Description != "Note: read" {
		t.Fatalf("description = %v", n.Description)
	}
	var stored string
	if err := f.s.DB.QueryRow(`SELECT created_at FROM favourite_nodes WHERE id = ?`, id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != db.FormatTime(created) {
		t.Fatalf("created_at = %s, want %s", stored, db.FormatTime(created))
	}
	var nowStored string
	if err := f.s.DB.QueryRow(`SELECT created_at FROM favourite_nodes WHERE id = ?`, f.child("", "https://example.org/now")).Scan(&nowStored); err != nil {
		t.Fatal(err)
	}
	if nowStored == db.FormatTime(time.Time{}) {
		t.Fatal("link without a date got the zero time")
	}
}

func TestImportQueuesFetches(t *testing.T) {
	f := newFixture(t)
	f.s.Jobs = jobs.New(f.s.DB)
	f.importItems(f.tree().ID, importFolder("Dev"), importLink("https://go.dev/doc"))
	id := f.child("", "https://go.dev/doc")
	want := []string{
		"favicon https://go.dev https://go.dev/doc",
		"preview_fetch " + id + " https://go.dev/doc",
	}
	if got := queued(f); !equal(got, want) {
		t.Fatalf("jobs = %v, want %v", got, want)
	}
}

func TestImportIsAllOrNothing(t *testing.T) {
	f := newFixture(t)
	_, err := f.s.Import(f.ctx, f.user, f.tree().ID, ImportTree{Root: ImportFolder{Items: []ImportItem{
		importFolder("Dev", importLink("https://go.dev")),
		importLink("https://example.org", ""),
	}}}, true)
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("err = %v, want ErrInvalidTag", err)
	}
	if got := f.titles(""); len(got) != 0 {
		t.Fatalf("root after failed import = %v, want empty", got)
	}
	if n := f.tagCount(); n != 0 {
		t.Fatalf("tags after failed import = %d", n)
	}
}

func TestImportFolderMustBeOwnFolder(t *testing.T) {
	f := newFixture(t)
	link := f.link("", "Docs")
	other, err := f.s.EnsureRoot(f.ctx, f.s.DB, f.other)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, id string
		want     error
	}{
		{"missing", "", ErrNotFound},
		{"link", link, ErrInvalidParent},
		{"other user's root", other, ErrNotFound},
	} {
		if _, err := f.s.Import(f.ctx, f.user, tt.id, ImportTree{}, true); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestImportSourceIDs(t *testing.T) {
	f := newFixture(t)
	root := f.tree().ID
	f.importItems(root, importFolder("Dev", importLink("https://go.dev")))
	dev := f.child("", "Dev")
	existing := f.child(dev, "https://go.dev")

	ids := map[string]string{}
	tree := ImportTree{IDs: ids, Root: ImportFolder{Items: []ImportItem{
		{Folder: &ImportFolder{Name: "dev", SourceID: "old-dev", Items: []ImportItem{
			{Link: &ImportLink{URL: "https://go.dev/", SourceID: "old-go"}},
			{Link: &ImportLink{URL: "https://pkg.go.dev", Title: ptr("pkg"), SourceID: "old-pkg"}},
		}}},
		{Folder: &ImportFolder{Name: "New", SourceID: "old-new"}},
	}}}
	if _, err := f.s.Import(f.ctx, f.user, root, tree, true); err != nil {
		t.Fatal(err)
	}
	if ids["old-dev"] != dev {
		t.Errorf("reused folder maps to %q, want %q", ids["old-dev"], dev)
	}
	if ids["old-go"] != existing {
		t.Errorf("skipped duplicate maps to %q, want the existing link %q", ids["old-go"], existing)
	}
	if ids["old-pkg"] != f.child(dev, "pkg") {
		t.Errorf("new link maps to %q", ids["old-pkg"])
	}
	if ids["old-new"] != f.child("", "New") {
		t.Errorf("new folder maps to %q", ids["old-new"])
	}
}

func TestImportPreviewImage(t *testing.T) {
	f := newFixture(t)
	img := "7f1c1a0e-3b9a-4c43-8a0e-1d2b3c4d5e6f"
	if _, err := f.s.DB.ExecContext(f.ctx,
		`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'preview', ?, 'image/png', 3, ?)`,
		img, f.user, db.FormatTime(time.Now())); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	tree := ImportTree{IDs: ids, Root: ImportFolder{Items: []ImportItem{
		{Link: &ImportLink{URL: "https://go.dev", SourceID: "go", PreviewImageID: &img}},
	}}}
	if _, err := f.s.Import(f.ctx, f.user, f.tree().ID, tree, true); err != nil {
		t.Fatal(err)
	}
	var got *string
	if err := f.s.DB.QueryRowContext(f.ctx, `SELECT preview_image_id FROM favourite_nodes WHERE id = ?`, ids["go"]).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != img {
		t.Errorf("preview_image_id = %v, want %s", got, img)
	}
}
