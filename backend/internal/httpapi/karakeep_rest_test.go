package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

// karakeepRESTClient calls the Karakeep REST API with a Karakeep token.
type karakeepRESTClient struct {
	s      *server
	t      *testing.T
	secret string
	rootID string
}

func (s *server) karakeepRESTClient(t *testing.T, cookie string) karakeepRESTClient {
	t.Helper()
	token := s.createToken(t, cookie, nil)
	return karakeepRESTClient{s: s, t: t, secret: token.Secret, rootID: *token.RootID}
}

func (c karakeepRESTClient) do(method, path, body string) *httptest.ResponseRecorder {
	return c.s.req(method, karakeepV1+path, body, withBearer(c.secret), fromExtension)
}

// json sends a request and decodes the response, which must have the status.
func kkJSON[T any](c karakeepRESTClient, method, path string, body any, wantStatus int) T {
	c.t.Helper()
	raw := ""
	if body != nil {
		raw = mustJSON(body)
	}
	rec := c.do(method, path, raw)
	if rec.Code != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body)
	}
	var v T
	if wantStatus != http.StatusNoContent {
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			c.t.Fatalf("%s %s: decode %s: %v", method, path, rec.Body, err)
		}
	}
	return v
}

// children returns the titles of a folder's children in order.
func (s *server) children(t *testing.T, folderID string) []string {
	t.Helper()
	rows, err := s.db.Query(`SELECT title FROM favourite_nodes WHERE parent_id = ? ORDER BY position`, folderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var title sql.NullString
		if err := rows.Scan(&title); err != nil {
			t.Fatal(err)
		}
		out = append(out, title.String)
	}
	return out
}

type kkBookmark struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	AlreadyExists bool   `json:"alreadyExists"`
	Tags          []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tags"`
	Content struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"content"`
}

type kkList struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
}

type kkLists struct {
	Lists []kkList `json:"lists"`
}

type kkPage struct {
	Bookmarks  []kkBookmark `json:"bookmarks"`
	NextCursor *string      `json:"nextCursor"`
}

func TestKarakeepRESTAuthAndUnknownPaths(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.karakeepRESTClient(t, ann)

	rec := s.req("GET", karakeepV1+"/lists", "")
	if rec.Code != http.StatusUnauthorized || decodeJSONBody[map[string]string](t, rec)["error"] != "Unauthorized" {
		t.Errorf("without token: %d %s", rec.Code, rec.Body)
	}
	lw := s.linkwardenToken(t, ann)
	if rec := s.req("GET", karakeepV1+"/lists", "", withBearer(lw.secret)); rec.Code != http.StatusUnauthorized {
		t.Errorf("Linkwarden token: %d", rec.Code)
	}
	for _, r := range [][2]string{{"GET", "/highlights"}, {"PUT", "/bookmarks/1"}, {"GET", "/lists/1/bookmarks/2"}} {
		if rec := c.do(r[0], r[1], ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s: %d %s", r[0], r[1], rec.Code, rec.Body)
		}
	}
	// Floccus' connection test.
	me := kkJSON[map[string]any](c, "GET", "/users/me", nil, http.StatusOK)
	if me["email"] != "ann@example.org" || me["id"] == nil {
		t.Errorf("users/me = %v", me)
	}
	if rec := s.req("GET", karakeepV1+"/users/me", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("users/me without token: %d", rec.Code)
	}
	if rec := c.do("POST", "/bookmarks", "{"); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON: %d", rec.Code)
	}
}

func TestKarakeepRESTEndpoints(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.karakeepRESTClient(t, ann)
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	outside := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/work", "parentId": work.ID})

	// Bookmarks: create, create again, get, rename, out of scope.
	created := kkJSON[kkBookmark](c, "POST", "/bookmarks", map[string]any{"type": "link", "url": "https://example.org", "title": "Example"}, http.StatusCreated)
	if created.AlreadyExists || created.Title != "Example" || created.Content.URL != "https://example.org" {
		t.Fatalf("created = %+v", created)
	}
	again := kkJSON[kkBookmark](c, "POST", "/bookmarks", map[string]any{"type": "link", "url": "https://example.org/"}, http.StatusOK)
	if !again.AlreadyExists || again.ID != created.ID {
		t.Errorf("again = %+v", again)
	}
	if got := kkJSON[kkBookmark](c, "GET", "/bookmarks/"+created.ID+"?includeContent=false", nil, http.StatusOK); got.ID != created.ID {
		t.Errorf("get = %+v", got)
	}
	renamed := kkJSON[kkBookmark](c, "PATCH", "/bookmarks/"+created.ID, map[string]any{"url": "https://example.org", "title": "Example site"}, http.StatusOK)
	if renamed.Title != "Example site" {
		t.Errorf("renamed = %+v", renamed)
	}
	if rec := c.do("GET", "/bookmarks/"+outside.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("outside: %d", rec.Code)
	}
	if rec := c.do("POST", "/bookmarks", mustJSON(map[string]any{"type": "link", "url": "javascript:alert(1)"})); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid URL: %d", rec.Code)
	}

	// Tags: attach by name, then detach.
	attached := kkJSON[map[string][]string](c, "POST", "/bookmarks/"+created.ID+"/tags", map[string]any{"tags": []map[string]string{{"tagName": "dev"}}}, http.StatusOK)
	tagged := kkJSON[kkBookmark](c, "GET", "/bookmarks/"+created.ID, nil, http.StatusOK)
	if len(attached["attached"]) != 1 || len(tagged.Tags) != 1 || tagged.Tags[0].ID != attached["attached"][0] || tagged.Tags[0].Name != "dev" {
		t.Fatalf("attached = %v, tags = %+v", attached, tagged.Tags)
	}
	detached := kkJSON[map[string][]string](c, "DELETE", "/bookmarks/"+created.ID+"/tags", map[string]any{"tags": []map[string]string{{"tagName": "dev"}}}, http.StatusOK)
	if !reflect.DeepEqual(detached["detached"], attached["attached"]) {
		t.Errorf("detached = %v", detached)
	}
	if got := kkJSON[kkBookmark](c, "GET", "/bookmarks/"+created.ID, nil, http.StatusOK); len(got.Tags) != 0 {
		t.Errorf("tags after detach = %+v", got.Tags)
	}

	// Lists: create (as Floccus does, with an icon and type manual), nested,
	// rename and move, add and remove a bookmark, lists of a bookmark.
	top := kkJSON[kkList](c, "POST", "/lists", map[string]any{"name": "Floccus", "icon": "📔", "type": "manual"}, http.StatusCreated)
	bar := kkJSON[kkList](c, "POST", "/lists", map[string]any{"name": "Bookmarks bar", "icon": "📔", "type": "manual", "parentId": top.ID}, http.StatusCreated)
	if bar.ParentID == nil || *bar.ParentID != top.ID {
		t.Fatalf("bar = %+v", bar)
	}
	if got := kkJSON[kkLists](c, "GET", "/lists", nil, http.StatusOK); len(got.Lists) != 2 {
		t.Errorf("lists = %+v", got)
	}
	kkJSON[any](c, "PUT", "/lists/"+bar.ID+"/bookmarks/"+created.ID, nil, http.StatusNoContent)
	if got := s.children(t, bar.ID); !reflect.DeepEqual(got, []string{"Example site"}) {
		t.Errorf("bar = %v", got)
	}
	if got := kkJSON[kkLists](c, "GET", "/bookmarks/"+created.ID+"/lists", nil, http.StatusOK); len(got.Lists) != 1 || got.Lists[0].ID != bar.ID {
		t.Errorf("lists of bookmark = %+v", got)
	}
	edited := kkJSON[kkList](c, "PATCH", "/lists/"+bar.ID, map[string]any{"name": "Toolbar", "parentId": nil}, http.StatusOK)
	if edited.Name != "Toolbar" || edited.ParentID != nil {
		t.Errorf("edited = %+v", edited)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus", "Toolbar"}) {
		t.Errorf("root = %v", got)
	}
	kkJSON[any](c, "DELETE", "/lists/"+bar.ID+"/bookmarks/"+created.ID, nil, http.StatusNoContent)
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Example site", "Floccus", "Toolbar"}) {
		t.Errorf("root after remove from list = %v", got)
	}
	// Not in that list (any more): nothing changes.
	kkJSON[any](c, "DELETE", "/lists/"+bar.ID+"/bookmarks/"+created.ID, nil, http.StatusNoContent)
	if got := s.children(t, c.rootID); got[0] != "Example site" {
		t.Errorf("root after removing again = %v", got)
	}
	if rec := c.do("POST", "/lists", mustJSON(map[string]any{"name": "Smart", "type": "smart", "query": "#dev"})); rec.Code != http.StatusBadRequest {
		t.Errorf("smart list: %d", rec.Code)
	}
	kkJSON[any](c, "DELETE", "/lists/"+bar.ID, nil, http.StatusNoContent)
	if rec := c.do("PATCH", "/lists/"+bar.ID, `{"name":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("edit deleted list: %d", rec.Code)
	}

	kkJSON[any](c, "DELETE", "/bookmarks/"+created.ID, nil, http.StatusNoContent)
	if rec := c.do("GET", "/bookmarks/"+created.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("deleted bookmark: %d", rec.Code)
	}
}

func TestKarakeepRESTListPaging(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.karakeepRESTClient(t, ann)
	later := kkJSON[kkList](c, "POST", "/lists", map[string]any{"name": "Later"}, http.StatusCreated)
	for i := range 25 {
		s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/" + strconv.Itoa(i), "title": "L" + strconv.Itoa(i), "parentId": later.ID})
	}

	// Floccus always appends "&", and adds the cursor without encoding it.
	path := "/lists/" + later.ID + "/bookmarks?includeContent=false&"
	first := kkJSON[kkPage](c, "GET", path, nil, http.StatusOK)
	if len(first.Bookmarks) != 20 || first.NextCursor == nil {
		t.Fatalf("first page: %d, %v", len(first.Bookmarks), first.NextCursor)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(*first.NextCursor) {
		t.Errorf("cursor %q is not URL-safe", *first.NextCursor)
	}
	second := kkJSON[kkPage](c, "GET", path+"cursor="+*first.NextCursor, nil, http.StatusOK)
	if len(second.Bookmarks) != 5 || second.NextCursor != nil {
		t.Errorf("second page: %d, %v", len(second.Bookmarks), second.NextCursor)
	}
	seen := map[string]bool{}
	for _, b := range append(first.Bookmarks, second.Bookmarks...) {
		seen[b.ID] = true
	}
	if len(seen) != 25 {
		t.Errorf("distinct bookmarks = %d, want 25", len(seen))
	}

	if got := kkJSON[kkPage](c, "GET", "/lists/"+later.ID+"/bookmarks?limit=100&sortOrder=asc", nil, http.StatusOK); len(got.Bookmarks) != 25 || got.NextCursor != nil {
		t.Errorf("limit 100 asc: %d, next %v", len(got.Bookmarks), got.NextCursor)
	}
	for _, q := range []string{"cursor=!!", "cursor=e30", "limit=x", "limit=101"} {
		if rec := c.do("GET", "/lists/"+later.ID+"/bookmarks?"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", q, rec.Code, rec.Body)
		}
	}
	if rec := c.do("GET", "/lists/nope/bookmarks", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown list: %d", rec.Code)
	}
}
