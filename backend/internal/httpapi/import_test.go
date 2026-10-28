package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"boardly/internal/notes"
)

// importFile posts an import file; empty parameters are left out.
func (s *server) importFile(t *testing.T, cookie string, params map[string]string, file string) *httptest.ResponseRecorder {
	t.Helper()
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	return s.req("POST", "/api/favourites/import?"+q.Encode(), file, withCookie(cookie), func(r *http.Request) {
		r.Header.Set("Content-Type", "application/octet-stream")
	})
}

func testdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../importer/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestImportFormats(t *testing.T) {
	for _, tt := range []struct {
		format, file string
		want         importReport
		top          []string
	}{
		{"karakeep", "karakeep.json", importReport{Links: 4, Folders: 7, Notes: 1, Categories: 1, Skipped: importReportSkipped{InvalidURLs: 1, SmartLists: 1}}, []string{"Dev", "Linux", "Floccus", "Empty", "In a smart list only"}},
		{"linkwarden", "linkwarden.json", importReport{Links: 2, Folders: 2, Skipped: importReportSkipped{NotLinks: 1}}, []string{"Reading"}},
		{"netscape", "browser.html", importReport{Links: 4, Folders: 2}, []string{"Toolbar", "Top"}},
	} {
		t.Run(tt.format, func(t *testing.T) {
			s := newServer(t)
			ann := s.signIn(t, "ann@example.org")
			root := s.tree(t, ann)
			rec := s.importFile(t, ann, map[string]string{"format": tt.format, "folderId": root.ID}, testdata(t, tt.file))
			if rec.Code != http.StatusOK {
				t.Fatalf("import: %d %s", rec.Code, rec.Body)
			}
			if got := decodeJSONBody[importReport](t, rec); got != tt.want {
				t.Fatalf("report = %+v, want %+v", got, tt.want)
			}
			if got := childTitles(s.tree(t, ann)); !reflect.DeepEqual(got, tt.top) {
				t.Fatalf("root = %v, want %v", got, tt.top)
			}
		})
	}
}

func TestImportErrors(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	root := s.tree(t, ann).ID
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org"}).ID
	bobRoot := s.tree(t, bob).ID
	bobCategory := s.noteCategory(t, bob, "Bob", "")
	karakeep := testdata(t, "karakeep.json")

	for _, tt := range []struct {
		name   string
		fields map[string]string
		file   string
		status int
		code   string
	}{
		{"wrong format", map[string]string{"format": "karakeep", "folderId": root}, testdata(t, "browser.html"), 400, "invalid_import_file"},
		{"unknown format", map[string]string{"format": "pocket", "folderId": root}, karakeep, 400, "invalid_request"},
		{"no format", map[string]string{"folderId": root}, karakeep, 400, "invalid_request"},
		{"no file", map[string]string{"format": "karakeep", "folderId": root}, "", 400, "invalid_request"},
		{"no folder", map[string]string{"format": "karakeep"}, karakeep, 400, "invalid_folder"},
		{"link as folder", map[string]string{"format": "karakeep", "folderId": link}, karakeep, 400, "invalid_folder"},
		{"other user's folder", map[string]string{"format": "karakeep", "folderId": bobRoot}, karakeep, 400, "invalid_folder"},
		{"invalid skipDuplicates", map[string]string{"format": "karakeep", "folderId": root, "skipDuplicates": "yes"}, karakeep, 400, "invalid_request"},
		{"too large", map[string]string{"format": "netscape", "folderId": root}, strings.Repeat("x", maxImportBytes+1), 413, "file_too_large"},
		{"other user's category", map[string]string{"format": "karakeep", "folderId": root, "categoryId": bobCategory}, karakeep, 400, "invalid_category"},
		{"unknown category", map[string]string{"format": "netscape", "folderId": root, "categoryId": "nope"}, testdata(t, "browser.html"), 400, "invalid_category"},
	} {
		rec := s.importFile(t, ann, tt.fields, tt.file)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), `"code":"`+tt.code+`"`) {
			t.Errorf("%s: %d %s, want %d %s", tt.name, rec.Code, rec.Body, tt.status, tt.code)
		}
	}
	if got := childTitles(s.tree(t, ann)); !reflect.DeepEqual(got, []string{"https://example.org"}) {
		t.Fatalf("tree after failed imports = %v", got)
	}
	if got := childTitles(s.tree(t, bob)); len(got) != 0 {
		t.Fatalf("bob's tree = %v", got)
	}
	if got := s.noteTitles(t, ann); len(got) != 0 {
		t.Fatalf("notes after failed imports = %v", got)
	}
}

// noteCategory creates a note category below parentID ("" for the top level).
func (s *server) noteCategory(t *testing.T, cookie, title, parentID string) string {
	t.Helper()
	body := `{"title":"` + title + `"}`
	if parentID != "" {
		body = `{"title":"` + title + `","parentId":"` + parentID + `"}`
	}
	rec := s.req("POST", "/api/note-categories", body, withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: %d %s", rec.Code, rec.Body)
	}
	return decodeJSONBody[notes.Category](t, rec).ID
}

// noteTitles returns the titles of the user's notes with their category's
// title ("-" for none), sorted.
func (s *server) noteTitles(t *testing.T, cookie string) []string {
	t.Helper()
	categories := map[string]string{}
	for _, c := range decodeJSONBody[[]notes.Category](t, s.req("GET", "/api/note-categories", "", withCookie(cookie))) {
		categories[c.ID] = c.Title
	}
	var out []string
	for _, n := range decodeJSONBody[[]notes.Note](t, s.req("GET", "/api/notes", "", withCookie(cookie))) {
		category := "-"
		if n.CategoryID != nil {
			category = categories[*n.CategoryID]
		}
		out = append(out, category+"/"+*n.Title)
	}
	slices.Sort(out)
	return out
}

func TestImportNotes(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	root := s.tree(t, ann).ID
	imported := s.noteCategory(t, ann, "Imported", "")
	file := testdata(t, "karakeep.json")

	rec := s.importFile(t, ann, map[string]string{"format": "karakeep", "folderId": root, "categoryId": imported}, file)
	if rec.Code != http.StatusOK {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	if got := s.noteTitles(t, ann); !reflect.DeepEqual(got, []string{"Linux/Text note"}) {
		t.Fatalf("notes = %v", got)
	}
	for _, c := range decodeJSONBody[[]notes.Category](t, s.req("GET", "/api/note-categories", "", withCookie(ann))) {
		if c.Title == "Linux" && (c.ParentID == nil || *c.ParentID != imported) {
			t.Fatalf("Linux is not below Imported: %+v", c)
		}
	}

	rec = s.importFile(t, ann, map[string]string{"format": "karakeep", "folderId": root, "categoryId": imported}, file)
	if got := decodeJSONBody[importReport](t, rec); got.Notes != 0 || got.Categories != 0 || got.Skipped.DuplicateNotes != 1 {
		t.Fatalf("second import: report = %+v, want 1 duplicate note", got)
	}
	rec = s.importFile(t, ann, map[string]string{"format": "karakeep", "folderId": root, "categoryId": imported, "skipDuplicates": "false"}, file)
	if got := decodeJSONBody[importReport](t, rec); got.Notes != 1 || got.Skipped.DuplicateNotes != 0 {
		t.Fatalf("duplicates allowed: report = %+v, want 1 note", got)
	}
}

func TestImportRollsBackWhenNotesFail(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	root := s.tree(t, ann).ID
	if _, err := s.db.Exec(`CREATE TRIGGER fail_notes BEFORE INSERT ON notes BEGIN SELECT RAISE(ABORT, 'disk full'); END`); err != nil {
		t.Fatal(err)
	}
	rec := s.importFile(t, ann, map[string]string{"format": "karakeep", "folderId": root}, testdata(t, "karakeep.json"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("import: %d %s, want 500", rec.Code, rec.Body)
	}
	if got := childTitles(s.tree(t, ann)); len(got) != 0 {
		t.Fatalf("tree after failed import = %v", got)
	}
	if categories := decodeJSONBody[[]notes.Category](t, s.req("GET", "/api/note-categories", "", withCookie(ann))); len(categories) != 0 {
		t.Fatalf("categories after failed import = %+v", categories)
	}
}

func TestImportSkipDuplicates(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	root := s.tree(t, ann).ID
	file := testdata(t, "browser.html")
	for _, tt := range []struct {
		skip       string
		links, dup int
	}{
		{"", 4, 0},
		{"", 0, 4},
		{"true", 0, 4},
		{"false", 4, 0},
	} {
		rec := s.importFile(t, ann, map[string]string{"format": "netscape", "folderId": root, "skipDuplicates": tt.skip}, file)
		if rec.Code != http.StatusOK {
			t.Fatalf("skipDuplicates=%q: %d %s", tt.skip, rec.Code, rec.Body)
		}
		if got := decodeJSONBody[importReport](t, rec); got.Links != tt.links || got.Skipped.Duplicates != tt.dup {
			t.Fatalf("skipDuplicates=%q: report = %+v, want %d links and %d duplicates", tt.skip, got, tt.links, tt.dup)
		}
	}
}

func TestImportRejectsFormPosts(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	rec := s.req("POST", "/api/favourites/import?format=netscape&folderId="+s.tree(t, ann).ID, testdata(t, "browser.html"), withCookie(ann), func(r *http.Request) {
		r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("form post: %d, want 403", rec.Code)
	}
}

func TestImportRequiresSignIn(t *testing.T) {
	s := newServer(t)
	if rec := s.importFile(t, "", map[string]string{"format": "karakeep", "folderId": "x"}, "{}"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("signed out: %d, want 401", rec.Code)
	}
}
