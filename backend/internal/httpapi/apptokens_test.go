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

func TestKarakeepBearerAuth(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	expires := s.clock.Now().Add(time.Hour)
	active := s.createToken(t, ann, expires.Format(time.RFC3339))
	revoked := s.createToken(t, ann, nil)
	s.req("POST", "/api/app-tokens/"+revoked.ID+"/revoke", "", withCookie(ann))
	const me = "/api/karakeep/api/v1/users/me"

	rec := s.req("GET", me, "", withBearer(active.Secret))
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("valid token: %d %s", rec.Code, rec.Body)
	}
	got := decodeJSONBody[struct {
		ID    string  `json:"id"`
		Name  *string `json:"name"`
		Email string  `json:"email"`
	}](t, rec)
	if got.Email != "ann@example.org" || got.ID == "" {
		t.Errorf("users/me = %+v", got)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); list[1].LastUsedAt == nil || !list[1].LastUsedAt.Equal(s.clock.Now()) {
		t.Errorf("last used = %v, want %v", list[1].LastUsedAt, s.clock.Now())
	}

	// Unknown paths answer 404 in the JSON format once authenticated.
	if rec := s.req("GET", "/api/karakeep/api/v1/nope", "", withBearer(active.Secret)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Errorf("unknown path: %d %s", rec.Code, rec.Body)
	}

	unknown := s.req("GET", me, "", withBearer(apptokens.SecretPrefix+"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	if unknown.Code != http.StatusUnauthorized || errorCode(t, unknown) != "unauthenticated" {
		t.Fatalf("unknown token: %d %s", unknown.Code, unknown.Body)
	}
	for name, opts := range map[string][]func(*http.Request){
		"cookie only":    {withCookie(ann)},
		"nothing":        nil,
		"revoked":        {withBearer(revoked.Secret)},
		"basic scheme":   {func(r *http.Request) { r.Header.Set("Authorization", "Basic "+active.Secret) }},
		"unknown path":   {withCookie(ann), func(r *http.Request) { r.URL.Path = "/api/karakeep/api/v1/nope" }},
		"cookie and bad": {withCookie(ann), withBearer("famo_nope")},
	} {
		rec := s.req("GET", me, "", opts...)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
			t.Errorf("%s: %d %s, want 401 like an unknown token", name, rec.Code, rec.Body)
		}
	}

	s.clock.Advance(time.Hour)
	if rec := s.req("GET", me, "", withBearer(active.Secret)); rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
		t.Errorf("expired: %d %s", rec.Code, rec.Body)
	}
}

func TestKarakeepSkipsCSRFAndAllowsCORS(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)

	rec := s.req("OPTIONS", "/api/karakeep/api/v1/users/me", "", func(r *http.Request) {
		r.Header.Set("Origin", "chrome-extension://abc")
		r.Header.Set("Access-Control-Request-Method", "GET")
		r.Header.Set("Access-Control-Request-Headers", "authorization")
	})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" ||
		rec.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}

	// A cross-origin multipart request isn't rejected by the cross-site check:
	// it is authenticated and then routed (no such endpoint yet, so 404/405).
	rec = s.req("POST", "/api/karakeep/api/v1/users/me", "x", withBearer(token.Secret), func(r *http.Request) {
		r.Header.Set("Origin", "chrome-extension://abc")
		r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	})
	if rec.Code != http.StatusMethodNotAllowed || errorCode(t, rec) != "method_not_allowed" {
		t.Errorf("cross-origin POST: %d %s, want 405 method_not_allowed", rec.Code, rec.Body)
	}

	// The web app's API keeps its cross-site check and has no CORS headers.
	rec = s.req("POST", "/api/app-tokens", `{}`, withCookie(ann), func(r *http.Request) { r.Header.Set("Origin", "chrome-extension://abc") })
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "forbidden_origin" || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("web API cross-origin: %d %s", rec.Code, rec.Body)
	}
}
