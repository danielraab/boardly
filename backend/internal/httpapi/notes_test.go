package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"boardly/internal/notes"
)

func TestNotesRequireSignIn(t *testing.T) {
	s := newServer(t)
	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/api/notes", ""},
		{"POST", "/api/notes", `{}`},
		{"GET", "/api/notes/x", ""},
		{"PATCH", "/api/notes/x", `{}`},
		{"DELETE", "/api/notes/x", ""},
		{"GET", "/api/note-categories", ""},
		{"POST", "/api/note-categories", `{}`},
		{"PATCH", "/api/note-categories/x", `{}`},
		{"DELETE", "/api/note-categories/x", ""},
	} {
		if rec := s.req(tt.method, tt.path, tt.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", tt.method, tt.path, rec.Code)
		}
	}
}

func TestNotesAPI(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	rec := s.req("POST", "/api/notes", `{"title":" Groceries ","color":"green","icon":{"type":"symbol","name":"shopping-cart"},
		"content":{"v":1,"blocks":[{"type":"list","style":"task","items":[{"content":[{"type":"text","text":"milk"}]}]}]}}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	n := decodeJSONBody[notes.Note](t, rec)
	if *n.Title != "Groceries" || *n.Color != "green" || n.Icon.Name != "shopping-cart" || n.Version != 1 {
		t.Errorf("created = %+v", n)
	}
	if !strings.Contains(rec.Body.String(), `"checked":false`) || !strings.Contains(rec.Body.String(), `"categoryId":null`) {
		t.Errorf("created body = %s", rec.Body)
	}

	rec = s.req("POST", "/api/notes", `{}`, withCookie(ann))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"content":{"v":1,"blocks":[]}`) {
		t.Fatalf("empty create: %d %s", rec.Code, rec.Body)
	}

	rec = s.req("POST", "/api/notes", `{"title":"Plan","text":"milk\neggs"}`, withCookie(ann))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(),
		`"content":{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"milk"},{"type":"break"},{"type":"text","text":"eggs"}]}]}`) {
		t.Fatalf("create from text: %d %s", rec.Code, rec.Body)
	}

	rec = s.req("GET", "/api/notes", "", withCookie(ann))
	if list := decodeJSONBody[[]notes.Note](t, rec); rec.Code != http.StatusOK || len(list) != 3 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/notes", "", withCookie(bob)); rec.Body.String() != "[]\n" && rec.Body.String() != "[]" {
		t.Errorf("Bob's list = %s", rec.Body)
	}

	rec = s.req("PATCH", "/api/notes/"+n.ID, `{"version":1,"title":null,"color":null}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	if p := decodeJSONBody[notes.Note](t, rec); p.Title != nil || p.Color != nil || p.Icon == nil || p.Version != 2 {
		t.Errorf("patched = %+v", p)
	}

	rec = s.req("PATCH", "/api/notes/"+n.ID, `{"version":2,"archived":true}`, withCookie(ann))
	if a := decodeJSONBody[notes.Note](t, rec); rec.Code != http.StatusOK || !a.Archived || a.Pinned || a.Version != 3 {
		t.Fatalf("archive: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("GET", "/api/notes", "", withCookie(ann))
	if list := decodeJSONBody[[]notes.Note](t, rec); len(list) != 3 || list[0].ID != n.ID || !list[0].Archived {
		t.Errorf("list after archiving = %s", rec.Body)
	}
	rec = s.req("POST", "/api/notes", `{"pinned":true}`, withCookie(ann))
	if c := decodeJSONBody[notes.Note](t, rec); rec.Code != http.StatusCreated || !c.Pinned || c.Archived {
		t.Errorf("create pinned: %d %s", rec.Code, rec.Body)
	}

	for name, tt := range map[string]struct {
		method, path, body, cookie string
		code                       int
		errCode                    string
	}{
		"stale version":        {"PATCH", "/api/notes/" + n.ID, `{"version":1,"title":"x"}`, ann, 409, "version_conflict"},
		"missing version":      {"PATCH", "/api/notes/" + n.ID, `{"title":"x"}`, ann, 400, "invalid_request"},
		"other user's get":     {"GET", "/api/notes/" + n.ID, "", bob, 404, "not_found"},
		"other user's patch":   {"PATCH", "/api/notes/" + n.ID, `{"version":2}`, bob, 404, "not_found"},
		"other user's delete":  {"DELETE", "/api/notes/" + n.ID, "", bob, 404, "not_found"},
		"long title":           {"POST", "/api/notes", `{"title":"` + strings.Repeat("x", 201) + `"}`, ann, 400, "invalid_note"},
		"unknown colour":       {"POST", "/api/notes", `{"color":"magenta"}`, ann, 400, "invalid_note"},
		"unknown symbol":       {"POST", "/api/notes", `{"icon":{"type":"symbol","name":"nope"}}`, ann, 400, "invalid_note"},
		"unknown category":     {"POST", "/api/notes", `{"categoryId":"nope"}`, ann, 400, "invalid_note"},
		"unknown field":        {"POST", "/api/notes", `{"starred":true}`, ann, 400, "invalid_note"},
		"pinned not a boolean": {"PATCH", "/api/notes/" + n.ID, `{"version":3,"pinned":"yes"}`, ann, 400, "invalid_note"},
		"archived null":        {"POST", "/api/notes", `{"archived":null}`, ann, 400, "invalid_note"},
		"wrong type":           {"POST", "/api/notes", `{"title":3}`, ann, 400, "invalid_note"},
		"invalid document":     {"POST", "/api/notes", `{"content":{"v":1,"blocks":[{"type":"iframe"}]}}`, ann, 400, "invalid_document"},
		"script link":          {"POST", "/api/notes", `{"content":{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"link","url":"javascript:alert(1)","content":[]}]}]}}`, ann, 400, "invalid_document"},
		"too large":            {"POST", "/api/notes", `{"content":{"v":1,"blocks":[{"type":"code","text":"` + strings.Repeat("x", 600<<10) + `"}]}}`, ann, 400, "invalid_document"},
		"not an object":        {"POST", "/api/notes", `[]`, ann, 400, "invalid_request"},
		"content and text":     {"POST", "/api/notes", `{"text":"a","content":{"v":1,"blocks":[]}}`, ann, 400, "invalid_request"},
		"text not a string":    {"POST", "/api/notes", `{"text":3}`, ann, 400, "invalid_request"},
		"text null":            {"POST", "/api/notes", `{"text":null}`, ann, 400, "invalid_request"},
		"text too large":       {"POST", "/api/notes", `{"text":"` + strings.Repeat("x", 512<<10-20) + `"}`, ann, 400, "invalid_document"},
		"unknown note":         {"GET", "/api/notes/nope", "", ann, 404, "not_found"},
		"category title blank": {"POST", "/api/note-categories", `{"title":"  "}`, ann, 400, "invalid_category"},
		"category no title":    {"POST", "/api/note-categories", `{}`, ann, 400, "invalid_category"},
		"category colour":      {"POST", "/api/note-categories", `{"title":"A","color":"magenta"}`, ann, 400, "invalid_category"},
		"category parent":      {"POST", "/api/note-categories", `{"title":"A","parentId":"nope"}`, ann, 400, "invalid_parent"},
		"unknown category id":  {"PATCH", "/api/note-categories/nope", `{"title":"A"}`, ann, 404, "not_found"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := s.req(tt.method, tt.path, tt.body, withCookie(tt.cookie))
			if rec.Code != tt.code || !strings.Contains(rec.Body.String(), `"code":"`+tt.errCode+`"`) {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body, tt.code, tt.errCode)
			}
		})
	}

	// Categories: create, nest, reject cycles, and delete.
	rec = s.req("POST", "/api/note-categories", `{"title":"Home","color":"blue","icon":{"type":"symbol","name":"home"}}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: %d %s", rec.Code, rec.Body)
	}
	home := decodeJSONBody[notes.Category](t, rec)
	rec = s.req("POST", "/api/note-categories", `{"title":"Garden","parentId":"`+home.ID+`"}`, withCookie(ann))
	garden := decodeJSONBody[notes.Category](t, rec)
	if rec.Code != http.StatusCreated || *garden.ParentID != home.ID {
		t.Fatalf("create subcategory: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/note-categories/"+home.ID, `{"parentId":"`+garden.ID+`"}`, withCookie(ann)); rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_parent") {
		t.Errorf("cycle: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/notes/"+n.ID, `{"version":3,"categoryId":"`+home.ID+`"}`, withCookie(ann)); rec.Code != 200 {
		t.Fatalf("set category: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/notes/"+n.ID, `{"version":4,"categoryId":"`+home.ID+`"}`, withCookie(bob)); rec.Code != 404 {
		t.Errorf("Bob sets category: %d", rec.Code)
	}
	if rec := s.req("DELETE", "/api/note-categories/"+home.ID, "", withCookie(bob)); rec.Code != 404 {
		t.Errorf("Bob deletes category: %d", rec.Code)
	}
	if rec := s.req("DELETE", "/api/note-categories/"+home.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete category: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("GET", "/api/notes/"+n.ID, "", withCookie(ann))
	if got := decodeJSONBody[notes.Note](t, rec); got.CategoryID != nil || got.Version != 4 {
		t.Errorf("note after category delete = %+v", got)
	}
	rec = s.req("GET", "/api/note-categories", "", withCookie(ann))
	if list := decodeJSONBody[[]notes.Category](t, rec); len(list) != 1 || list[0].ParentID != nil {
		t.Errorf("categories after delete = %s", rec.Body)
	}

	if rec := s.req("DELETE", "/api/notes/"+n.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/notes/"+n.ID, "", withCookie(ann)); rec.Code != http.StatusNotFound {
		t.Errorf("deleted note: %d", rec.Code)
	}
}
