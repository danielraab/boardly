package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"famodash/internal/favourites"
)

func decodeJSONBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body, err)
	}
	return v
}

// create posts a node and returns it.
func (s *server) create(t *testing.T, cookie string, body map[string]any) favourites.Node {
	t.Helper()
	rec := s.req("POST", "/api/favourites", mustJSON(body), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %v: %d %s", body, rec.Code, rec.Body)
	}
	return decodeJSONBody[favourites.Node](t, rec)
}

func (s *server) tree(t *testing.T, cookie string) favourites.Node {
	t.Helper()
	rec := s.req("GET", "/api/favourites", "", withCookie(cookie))
	if rec.Code != http.StatusOK {
		t.Fatalf("tree: %d %s", rec.Code, rec.Body)
	}
	return decodeJSONBody[favourites.Node](t, rec)
}

func childTitles(n favourites.Node) []string {
	out := []string{}
	for _, c := range n.Children {
		if c.Title == nil {
			out = append(out, *c.URL)
			continue
		}
		out = append(out, *c.Title)
	}
	return out
}

func TestFavouritesRequireSignIn(t *testing.T) {
	s := newServer(t)
	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/api/favourites", ""},
		{"POST", "/api/favourites", `{}`},
		{"PATCH", "/api/favourites/x", `{}`},
		{"POST", "/api/favourites/x/move", `{}`},
		{"DELETE", "/api/favourites/x", ""},
		{"GET", "/api/tags", ""},
	} {
		if rec := s.req(tt.method, tt.path, tt.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d, want 401", tt.method, tt.path, rec.Code)
		}
	}
}

func TestFavouritesCRUD(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")

	root := s.tree(t, ann)
	if root.Kind != favourites.KindFolder || root.Title != nil || len(root.Children) != 0 {
		t.Fatalf("empty tree = %+v", root)
	}

	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	docs := s.create(t, ann, map[string]any{"parentId": work.ID, "kind": "link", "title": "Docs", "url": "https://example.org/docs", "tags": []string{"reading"}})
	if *docs.URL != "https://example.org/docs" || !reflect.DeepEqual(docs.Tags, []string{"reading"}) {
		t.Fatalf("created link = %+v", docs)
	}
	s.create(t, ann, map[string]any{"parentId": root.ID, "kind": "link", "url": "https://example.org"})

	root = s.tree(t, ann)
	if len(root.Children) != 2 || root.Children[0].ID != work.ID || root.Children[1].Title != nil {
		t.Fatalf("tree = %+v", root)
	}
	if got := childTitles(*root.Children[0]); !reflect.DeepEqual(got, []string{"Docs"}) {
		t.Fatalf("Work = %q", got)
	}

	rec := s.req("PATCH", "/api/favourites/"+docs.ID, `{"title":"Documentation","tags":["reading","new"]}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	if n := decodeJSONBody[favourites.Node](t, rec); *n.Title != "Documentation" || *n.URL != "https://example.org/docs" || !reflect.DeepEqual(n.Tags, []string{"new", "reading"}) {
		t.Fatalf("patched = %+v", n)
	}

	rec = s.req("GET", "/api/tags?q=RE", "", withCookie(ann))
	if tags := decodeJSONBody[[]favourites.Tag](t, rec); rec.Code != http.StatusOK || len(tags) != 1 || tags[0].Name != "reading" {
		t.Fatalf("tags: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("GET", "/api/tags?q=zzz", "", withCookie(ann))
	if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("no tags: %d %q, want an empty array", rec.Code, rec.Body)
	}

	if rec := s.req("POST", "/api/favourites/"+docs.ID+"/move", mustJSON(map[string]any{"parentId": root.ID, "position": 0}), withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("move: %d %s", rec.Code, rec.Body)
	}
	if got := childTitles(s.tree(t, ann))[:2]; !reflect.DeepEqual(got, []string{"Documentation", "Work"}) {
		t.Fatalf("root after move = %q", got)
	}

	if rec := s.req("DELETE", "/api/favourites/"+docs.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/favourites/"+docs.ID, "", withCookie(ann)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Fatalf("second delete: %d %s", rec.Code, rec.Body)
	}
}

func TestFavouritesDeleteModes(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	s.create(t, ann, map[string]any{"kind": "link", "title": "A", "url": "https://example.org/a"})
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	s.create(t, ann, map[string]any{"kind": "link", "title": "B", "url": "https://example.org/b"})
	tools := s.create(t, ann, map[string]any{"parentId": work.ID, "kind": "folder", "title": "Tools"})
	s.create(t, ann, map[string]any{"parentId": work.ID, "kind": "link", "title": "Docs", "url": "https://example.org/docs"})
	s.create(t, ann, map[string]any{"parentId": tools.ID, "kind": "link", "title": "Git", "url": "https://example.org/git"})

	rec := s.req("DELETE", "/api/favourites/"+work.ID, "", withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "folder_not_empty" {
		t.Fatalf("without mode: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/favourites/"+work.ID+"?children=all", "", withCookie(ann)); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown mode: %d %s", rec.Code, rec.Body)
	}

	if rec := s.req("DELETE", "/api/favourites/"+work.ID+"?children=move", "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("move mode: %d %s", rec.Code, rec.Body)
	}
	if got := childTitles(s.tree(t, ann)); !reflect.DeepEqual(got, []string{"A", "Tools", "Docs", "B"}) {
		t.Fatalf("after keeping contents = %q", got)
	}

	if rec := s.req("DELETE", "/api/favourites/"+tools.ID+"?children=delete", "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete mode: %d %s", rec.Code, rec.Body)
	}
	if got := childTitles(s.tree(t, ann)); !reflect.DeepEqual(got, []string{"A", "Docs", "B"}) {
		t.Fatalf("after deleting contents = %q", got)
	}
}

func TestFavouritesErrors(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	root := s.tree(t, ann)
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	tools := s.create(t, ann, map[string]any{"parentId": work.ID, "kind": "folder", "title": "Tools"})
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org"})

	for _, tt := range []struct {
		name, method, path, body, cookie string
		status                           int
		code                             string
	}{
		{"invalid JSON", "POST", "/api/favourites", `{`, ann, 400, "invalid_request"},
		{"empty folder name", "POST", "/api/favourites", `{"kind":"folder","title":" "}`, ann, 400, "invalid_title"},
		{"bad URL", "POST", "/api/favourites", `{"kind":"link","url":"javascript:alert(1)"}`, ann, 400, "invalid_url"},
		{"bad tag", "POST", "/api/favourites", `{"kind":"link","url":"https://example.org","tags":[""]}`, ann, 400, "invalid_tag"},
		{"bad kind", "POST", "/api/favourites", `{"kind":"note"}`, ann, 400, "invalid_kind"},
		{"url on folder", "POST", "/api/favourites", `{"kind":"folder","title":"F","url":"https://example.org"}`, ann, 400, "invalid_field"},
		{"parent is a link", "POST", "/api/favourites", `{"parentId":"` + link.ID + `","kind":"folder","title":"F"}`, ann, 400, "invalid_parent"},
		{"foreign parent", "POST", "/api/favourites", `{"parentId":"` + work.ID + `","kind":"folder","title":"F"}`, bob, 404, "not_found"},
		{"rename root", "PATCH", "/api/favourites/" + root.ID, `{"title":"Root"}`, ann, 400, "root_immutable"},
		{"foreign patch", "PATCH", "/api/favourites/" + work.ID, `{"title":"Mine"}`, bob, 404, "not_found"},
		{"move into subtree", "POST", "/api/favourites/" + work.ID + "/move", `{"parentId":"` + tools.ID + `","position":0}`, ann, 400, "invalid_move"},
		{"move root", "POST", "/api/favourites/" + root.ID + "/move", `{"parentId":"` + work.ID + `","position":0}`, ann, 400, "root_immutable"},
		{"foreign move", "POST", "/api/favourites/" + link.ID + "/move", `{"parentId":"` + work.ID + `","position":0}`, bob, 404, "not_found"},
		{"delete root", "DELETE", "/api/favourites/" + root.ID + "?children=delete", "", ann, 400, "root_immutable"},
		{"foreign delete", "DELETE", "/api/favourites/" + link.ID, "", bob, 404, "not_found"},
	} {
		rec := s.req(tt.method, tt.path, tt.body, withCookie(tt.cookie))
		if rec.Code != tt.status || errorCode(t, rec) != tt.code {
			t.Errorf("%s: %d %s, want %d %s", tt.name, rec.Code, rec.Body, tt.status, tt.code)
		}
	}

	if got := len(s.tree(t, bob).Children); got != 0 {
		t.Fatalf("bob sees %d nodes, want 0", got)
	}
}
