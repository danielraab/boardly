package httpapi

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"famodash/internal/apptokens"
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

	if rec := s.req("PATCH", "/api/app-tokens/"+created.ID, `{"name":"x"}`, withCookie(ann)); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH: %d, want 405", rec.Code)
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
	past := s.clock.Now().Add(-time.Hour).Format(time.RFC3339)
	for name, tc := range map[string]struct {
		body map[string]any
		code string
	}{
		"empty name":   {map[string]any{"name": " ", "app": "karakeep", "rootId": folder.ID}, "invalid_name"},
		"unknown app":  {map[string]any{"name": "x", "app": "pocket", "rootId": folder.ID}, "invalid_app"},
		"link as root": {map[string]any{"name": "x", "app": "karakeep", "rootId": link.ID}, "invalid_root"},
		"past expiry":  {map[string]any{"name": "x", "app": "karakeep", "rootId": folder.ID, "expiresAt": past}, "invalid_expiry"},
		"bad expiry":   {map[string]any{"name": "x", "app": "karakeep", "rootId": folder.ID, "expiresAt": "tomorrow"}, "invalid_expiry"},
	} {
		rec := s.req("POST", "/api/app-tokens", mustJSON(tc.body), withCookie(ann))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != tc.code {
			t.Errorf("%s: %d %s, want 400 %s", name, rec.Code, rec.Body, tc.code)
		}
	}
}
