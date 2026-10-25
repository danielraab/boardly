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
	report, err := f.s.Import(f.ctx, f.user, folderID, ImportTree{Root: ImportFolder{Items: items}})
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
	}}})
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
		if _, err := f.s.Import(f.ctx, f.user, tt.id, ImportTree{}); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
}
