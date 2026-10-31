package httpapi

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestLinkwardenFloccusFlow replays the calls of Floccus' Linkwarden adapter
// (floccusaddon/floccus src/lib/adapters/Linkwarden.ts) for a sync that
// creates, edits, moves and removes bookmarks and folders.
func TestLinkwardenFloccusFlow(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.linkwardenToken(t, ann)
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})

	id := func(n int64) string { return strconv.FormatInt(n, 10) }
	collections := func() map[string]lwCollection {
		t.Helper()
		out := map[string]lwCollection{}
		for _, col := range lwDecode[[]lwCollection](t, c.do("GET", lwV1+"/collections", ""), http.StatusOK) {
			out[col.Name] = col
		}
		return out
	}
	linksIn := func(collection int64) []string {
		t.Helper()
		var names []string
		for _, l := range lwDecode[lwLinks](t, c.do("GET", lwV1+"/search?searchQueryString=&collectionId="+id(collection), ""), http.StatusOK).Links {
			if l.CollectionID == collection {
				names = append(names, l.Name)
			}
		}
		return names
	}

	// Tree load: links page by page, then collections. Floccus looks for a
	// top-level collection with its folder's name and creates it when
	// there is none.
	serverFolder := func() lwCollection {
		t.Helper()
		page := lwDecode[lwLinks](t, c.do("GET", lwV1+"/search?searchQueryString=", ""), http.StatusOK)
		if page.NextCursor != nil {
			t.Fatalf("one page expected: %+v", page)
		}
		for _, col := range lwDecode[[]lwCollection](t, c.do("GET", lwV1+"/collections", ""), http.StatusOK) {
			if col.Name == "Floccus" && col.ParentID == nil {
				return col
			}
		}
		return lwDecode[lwCollection](t, c.do("POST", lwV1+"/collections", `{"name":"Floccus"}`), http.StatusOK)
	}
	root := collections()["Reading"]
	floccus := serverFolder()
	if floccus.ParentID != nil {
		t.Fatalf("floccus = %+v", floccus)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus"}) {
		t.Fatalf("token folder = %v", got)
	}
	// The next sync finds it again instead of creating another one.
	if again := serverFolder(); again.ID != floccus.ID {
		t.Fatalf("second sync: %+v, want %+v", again, floccus)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus"}) {
		t.Fatalf("token folder after second sync = %v", got)
	}

	// createFolder and createBookmark.
	bar := lwDecode[lwCollection](t, c.do("POST", lwV1+"/collections", mustJSON(map[string]any{"name": "Bookmarks bar", "parentId": floccus.ID})), http.StatusOK)
	link := lwDecode[lwLink](t, c.do("POST", lwV1+"/links", mustJSON(map[string]any{
		"url": "https://example.org/docs", "name": "Docs", "tags": []map[string]string{{"name": "dev"}},
		"collection": map[string]any{"id": bar.ID},
	})), http.StatusOK)
	if link.CollectionID != bar.ID || len(link.Tags) != 1 {
		t.Fatalf("link = %+v", link)
	}

	// updateBookmark: the parent collection first, then the link with its new
	// name, moved to "Floccus".
	parent := lwDecode[map[string]any](t, c.do("GET", lwV1+"/collections/"+id(floccus.ID), ""), http.StatusOK)
	updated := lwDecode[lwLink](t, c.do("PUT", lwV1+"/links/"+id(link.ID), mustJSON(map[string]any{
		"id": link.ID, "url": "https://example.org/docs", "name": "Docs (new)", "tags": []map[string]string{{"name": "dev"}},
		"collection": map[string]any{"id": parent["id"], "name": parent["name"], "ownerId": parent["ownerId"]},
	})), http.StatusOK)
	if updated.Name != "Docs (new)" || updated.CollectionID != floccus.ID {
		t.Fatalf("updated = %+v", updated)
	}

	// updateFolder: GET the collection and PUT it back renamed and moved to
	// the token's root.
	var body map[string]any
	if err := json.Unmarshal(decodeJSONBody[envelope](t, c.do("GET", lwV1+"/collections/"+id(bar.ID), "")).Response, &body); err != nil {
		t.Fatal(err)
	}
	body["name"] = "Toolbar"
	body["parentId"] = root.ID
	moved := lwDecode[lwCollection](t, c.do("PUT", lwV1+"/collections/"+id(bar.ID), mustJSON(body)), http.StatusOK)
	if moved.Name != "Toolbar" || moved.ParentID != nil {
		t.Fatalf("moved folder = %+v", moved)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus", "Toolbar"}) {
		t.Errorf("token folder after move = %v", got)
	}

	// A folder into its own subfolder, and out of scope.
	sub := lwDecode[lwCollection](t, c.do("POST", lwV1+"/collections", mustJSON(map[string]any{"name": "Sub", "parentId": moved.ID})), http.StatusOK)
	body["parentId"] = sub.ID
	if rec := c.do("PUT", lwV1+"/collections/"+id(bar.ID), mustJSON(body)); rec.Code != http.StatusBadRequest {
		t.Errorf("move into own subfolder: %d %s", rec.Code, rec.Body)
	}
	var workNum int64
	if err := s.db.QueryRow(`SELECT num FROM favourite_nodes WHERE id = ?`, work.ID).Scan(&workNum); err != nil {
		t.Fatal(err)
	}
	if rec := c.do("GET", lwV1+"/collections/"+id(workNum), ""); rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"response":"Collection not found."}` {
		t.Errorf("GET outside: %d %s", rec.Code, rec.Body)
	}
	body["parentId"] = workNum
	if rec := c.do("PUT", lwV1+"/collections/"+id(bar.ID), mustJSON(body)); rec.Code != http.StatusBadRequest {
		t.Errorf("move outside: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do("PUT", lwV1+"/collections/"+id(root.ID), `{"name":"Other"}`); rec.Code != http.StatusForbidden {
		t.Errorf("rename root: %d %s", rec.Code, rec.Body)
	}

	// removeBookmark, then removeFolder with a link inside: the link goes too.
	lwDecode[lwLink](t, c.do("DELETE", lwV1+"/links/"+id(updated.ID), ""), http.StatusOK)
	lwDecode[lwLink](t, c.do("POST", lwV1+"/links", mustJSON(map[string]any{
		"url": "https://example.org/inner", "name": "Inner", "tags": []any{}, "collection": map[string]any{"id": sub.ID},
	})), http.StatusOK)
	lwDecode[lwCollection](t, c.do("DELETE", lwV1+"/collections/"+id(moved.ID), ""), http.StatusOK)
	got := collections()
	if _, ok := got["Toolbar"]; ok {
		t.Error("Toolbar still exists")
	}
	if _, ok := got["Sub"]; ok {
		t.Error("Sub still exists")
	}
	if names := linksIn(root.ID); len(names) != 0 {
		t.Errorf("links in root after removing the folder = %v", names)
	}
	if names := linksIn(floccus.ID); len(names) != 0 {
		t.Errorf("links in Floccus = %v", names)
	}
}
