package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"boardly/internal/apptokens"
)

type createdToken struct {
	apptokens.Token
	Secret string `json:"secret"`
}

// createToken creates a Karakeep token rooted at a new folder "Reading".
func (s *server) createToken(t *testing.T, cookie string, expiresAt any) createdToken {
	t.Helper()
	folder := s.create(t, cookie, map[string]any{"kind": "folder", "title": "Reading"})
	rec := s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Phone", "app": "karakeep", "rootId": folder.ID, "expiresAt": expiresAt,
	}), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create token: %d %s", rec.Code, rec.Body)
	}
	return decodeJSONBody[createdToken](t, rec)
}

func withBearer(secret string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+secret) }
}

func TestAppTokensRequireSession(t *testing.T) {
	s := newServer(t)
	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/api/app-tokens", ""},
		{"POST", "/api/app-tokens", `{}`},
		{"POST", "/api/app-tokens/x/revoke", ""},
		{"PATCH", "/api/app-tokens/x", `{}`},
		{"DELETE", "/api/app-tokens/x", ""},
	} {
		if rec := s.req(tt.method, tt.path, tt.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d, want 401", tt.method, tt.path, rec.Code)
		}
	}
}

func TestAppTokensLifecycle(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	created := s.createToken(t, ann, nil)
	if created.Secret == "" || created.Status != apptokens.StatusActive || created.Prefix != created.Secret[:12] {
		t.Fatalf("created = %+v", created)
	}

	rec := s.req("GET", "/api/app-tokens", "", withCookie(ann))
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte(created.Secret)) || bytes.Contains(rec.Body.Bytes(), []byte(`"secret"`)) {
		t.Fatalf("list: %d %s; want 200 without the secret", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, rec); len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v", list)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(bob))); len(list) != 0 {
		t.Fatalf("bob's list = %+v, want empty", list)
	}

	if rec := s.req("PATCH", "/api/app-tokens/"+created.ID, `{"name":"x"}`, withCookie(ann)); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_request" {
		t.Errorf("rename: %d %s, want 400 invalid_request", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/app-tokens/"+created.ID, "", withCookie(ann)); rec.Code != http.StatusConflict || errorCode(t, rec) != "token_active" {
		t.Errorf("delete active: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"/api/app-tokens/" + created.ID + "/revoke", "/api/app-tokens/nope/revoke"} {
		if rec := s.req("POST", path, "", withCookie(bob)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Errorf("bob revokes %s: %d %s", path, rec.Code, rec.Body)
		}
	}

	rec = s.req("POST", "/api/app-tokens/"+created.ID+"/revoke", "", withCookie(ann))
	if rec.Code != http.StatusOK || decodeJSONBody[apptokens.Token](t, rec).Status != apptokens.StatusRevoked {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("POST", "/api/app-tokens/"+created.ID+"/revoke", "", withCookie(ann)); rec.Code != http.StatusConflict || errorCode(t, rec) != "token_inactive" {
		t.Errorf("revoke twice: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/app-tokens/"+created.ID, "", withCookie(bob)); rec.Code != http.StatusNotFound {
		t.Errorf("bob deletes: %d, want 404", rec.Code)
	}
	if rec := s.req("DELETE", "/api/app-tokens/"+created.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Errorf("delete revoked: %d %s", rec.Code, rec.Body)
	}
}

func TestCreateAppTokenErrors(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Reading"})
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org"})
	category := s.noteCategory(t, ann, "Inbox", "")
	bobCategory := s.noteCategory(t, s.signIn(t, "bob@example.org"), "Bob", "")
	past := s.clock.Now().Add(-time.Hour).Format(time.RFC3339)
	for name, tc := range map[string]struct {
		body map[string]any
		code string
	}{
		"empty name":              {map[string]any{"name": " ", "app": "karakeep", "rootId": folder.ID}, "invalid_name"},
		"unknown app":             {map[string]any{"name": "x", "app": "pocket", "rootId": folder.ID}, "invalid_app"},
		"link as root":            {map[string]any{"name": "x", "app": "karakeep", "rootId": link.ID}, "invalid_root"},
		"past expiry":             {map[string]any{"name": "x", "app": "karakeep", "rootId": folder.ID, "expiresAt": past}, "invalid_expiry"},
		"other user's category":   {map[string]any{"name": "x", "app": "karakeep", "rootId": folder.ID, "rootCategoryId": bobCategory}, "invalid_root_category"},
		"category for linkwarden": {map[string]any{"name": "x", "app": "linkwarden", "rootId": folder.ID, "rootCategoryId": category}, "invalid_root_category"},
		"bad expiry":              {map[string]any{"name": "x", "app": "karakeep", "rootId": folder.ID, "expiresAt": "tomorrow"}, "invalid_expiry"},
	} {
		rec := s.req("POST", "/api/app-tokens", mustJSON(tc.body), withCookie(ann))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != tc.code {
			t.Errorf("%s: %d %s, want 400 %s", name, rec.Code, rec.Body, tc.code)
		}
	}
}

func TestAppTokenNoteScope(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Reading"})
	inbox := s.noteCategory(t, ann, "Inbox", "")

	created := s.createToken(t, ann, nil)
	if created.NoteScope != apptokens.NoteScopeAll || created.RootCategoryID != nil {
		t.Fatalf("default scope = %s %v, want all", created.NoteScope, created.RootCategoryID)
	}
	rec := s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Inbox", "app": "karakeep", "rootId": folder.ID, "rootCategoryId": inbox,
	}), withCookie(ann))
	if tok := decodeJSONBody[createdToken](t, rec); rec.Code != http.StatusCreated || tok.NoteScope != apptokens.NoteScopeCategory || *tok.RootCategoryID != inbox {
		t.Fatalf("with category: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Tablet", "app": "linkwarden", "rootId": folder.ID,
	}), withCookie(ann))
	if tok := decodeJSONBody[createdToken](t, rec); tok.NoteScope != apptokens.NoteScopeNone {
		t.Fatalf("linkwarden: %s", rec.Body)
	}
}

func TestEditAppToken(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	tok := s.createToken(t, ann, nil)
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	s.create(t, ann, map[string]any{"kind": "folder", "title": "Projects", "parentId": work.ID})
	inbox := s.noteCategory(t, ann, "Inbox", "")
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org"})
	patch := func(cookie, body string) *httptest.ResponseRecorder {
		return s.req("PATCH", "/api/app-tokens/"+tok.ID, body, withCookie(cookie))
	}

	rec := patch(ann, `{"rootId":"`+work.ID+`","rootCategoryId":"`+inbox+`"}`)
	got := decodeJSONBody[apptokens.Token](t, rec)
	if rec.Code != http.StatusOK || *got.RootID != work.ID || got.NoteScope != apptokens.NoteScopeCategory || *got.RootCategoryID != inbox {
		t.Fatalf("edit: %d %s", rec.Code, rec.Body)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"secret"`)) || got.Prefix != tok.Prefix {
		t.Fatalf("edit changed the secret or returned it: %s", rec.Body)
	}
	// The next request with the token sees the new folder.
	if res := s.trpcQuery("lists.list", "{}", withBearer(tok.Secret)); !bytes.Contains(res.Body.Bytes(), []byte(`"Projects"`)) {
		t.Fatalf("lists after the edit: %s", res.Body)
	}

	if rec := patch(ann, `{"rootCategoryId":null}`); decodeJSONBody[apptokens.Token](t, rec).NoteScope != apptokens.NoteScopeAll {
		t.Fatalf("back to all notes: %s", rec.Body)
	}
	for name, tc := range map[string]struct {
		cookie, body string
		status       int
		code         string
	}{
		"link as root":   {ann, `{"rootId":"` + link.ID + `"}`, 400, "invalid_root"},
		"null root":      {ann, `{"rootId":null}`, 400, "invalid_root"},
		"other category": {ann, `{"rootCategoryId":"nope"}`, 400, "invalid_root_category"},
		"rename":         {ann, `{"name":"x"}`, 400, "invalid_request"},
		"expiry":         {ann, `{"expiresAt":null}`, 400, "invalid_request"},
		"other user":     {bob, `{"rootId":"` + work.ID + `"}`, 404, "not_found"},
	} {
		if rec := patch(tc.cookie, tc.body); rec.Code != tc.status || errorCode(t, rec) != tc.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, tc.status, tc.code)
		}
	}

	s.req("POST", "/api/app-tokens/"+tok.ID+"/revoke", "", withCookie(ann))
	if rec := patch(ann, `{"rootId":"`+work.ID+`"}`); rec.Code != http.StatusConflict || errorCode(t, rec) != "token_inactive" {
		t.Errorf("revoked: %d %s", rec.Code, rec.Body)
	}
}

func TestMemosAppToken(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Reading"})
	inbox := s.noteCategory(t, ann, "Inbox", "")

	rec := s.req("POST", "/api/app-tokens", `{"name":"MoeMemos","app":"memos"}`, withCookie(ann))
	tok := decodeJSONBody[createdToken](t, rec)
	if rec.Code != http.StatusCreated || tok.App != apptokens.AppMemos || tok.RootID != nil || tok.NoteScope != apptokens.NoteScopeAll {
		t.Fatalf("memos token: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("POST", "/api/app-tokens", mustJSON(map[string]any{"name": "Inbox", "app": "memos", "rootCategoryId": inbox}), withCookie(ann))
	if got := decodeJSONBody[createdToken](t, rec); rec.Code != http.StatusCreated || got.NoteScope != apptokens.NoteScopeCategory {
		t.Fatalf("with a category: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("POST", "/api/app-tokens", mustJSON(map[string]any{"name": "x", "app": "memos", "rootId": folder.ID}), withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_root" {
		t.Errorf("with a root folder: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); len(list) != 2 || list[1].RootID != nil {
		t.Errorf("list = %+v", list)
	}

	patch := func(body string) *httptest.ResponseRecorder {
		return s.req("PATCH", "/api/app-tokens/"+tok.ID, body, withCookie(ann))
	}
	if rec := patch(`{"rootCategoryId":"` + inbox + `","rootId":null}`); rec.Code != http.StatusOK || decodeJSONBody[apptokens.Token](t, rec).NoteScope != apptokens.NoteScopeCategory {
		t.Errorf("change the category: %d %s", rec.Code, rec.Body)
	}
	if rec := patch(`{"rootCategoryId":null}`); decodeJSONBody[apptokens.Token](t, rec).NoteScope != apptokens.NoteScopeAll {
		t.Errorf("back to all notes: %s", rec.Body)
	}
	if rec := patch(`{"rootId":"` + folder.ID + `"}`); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_root" {
		t.Errorf("set a root folder: %d %s", rec.Code, rec.Body)
	}
}
