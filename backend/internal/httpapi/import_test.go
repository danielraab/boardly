package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"boardly/internal/favourites"
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
		want         favourites.ImportReport
		top          []string
	}{
		{"karakeep", "karakeep.json", favourites.ImportReport{Links: 4, Folders: 7, Skipped: favourites.ImportSkipped{InvalidURLs: 1, NotLinks: 1, SmartLists: 1}}, []string{"Dev", "Linux", "Floccus", "Empty", "In a smart list only"}},
		{"linkwarden", "linkwarden.json", favourites.ImportReport{Links: 2, Folders: 2, Skipped: favourites.ImportSkipped{NotLinks: 1}}, []string{"Reading"}},
		{"netscape", "browser.html", favourites.ImportReport{Links: 4, Folders: 2}, []string{"Toolbar", "Top"}},
	} {
		t.Run(tt.format, func(t *testing.T) {
			s := newServer(t)
			ann := s.signIn(t, "ann@example.org")
			root := s.tree(t, ann)
			rec := s.importFile(t, ann, map[string]string{"format": tt.format, "folderId": root.ID}, testdata(t, tt.file))
			if rec.Code != http.StatusOK {
				t.Fatalf("import: %d %s", rec.Code, rec.Body)
			}
			if got := decodeJSONBody[favourites.ImportReport](t, rec); got != tt.want {
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
		{"too large", map[string]string{"format": "netscape", "folderId": root}, strings.Repeat("x", maxImportBytes+1), 413, "file_too_large"},
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
