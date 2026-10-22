package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"famodash/internal/apptokens"
)

const trpcPath = karakeepPrefix + "/api/trpc/"

// trpcQuery sends a batched tRPC query as httpBatchLink does.
func (s *server) trpcQuery(procedures, input string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	return s.req("GET", trpcPath+procedures+"?batch=1&input="+url.QueryEscape(input), "", opts...)
}

// trpcMutation sends a batched tRPC mutation from a browser extension.
func (s *server) trpcMutation(procedure, body string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	opts = append([]func(*http.Request){fromExtension}, opts...)
	return s.req("POST", trpcPath+procedure+"?batch=1", body, opts...)
}

func fromExtension(r *http.Request) { r.Header.Set("Origin", "chrome-extension://abc") }

type trpcEntry struct {
	Result *struct {
		Data struct {
			JSON json.RawMessage `json:"json"`
		} `json:"data"`
	} `json:"result"`
	Error *struct {
		JSON struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Data    struct {
				Code string `json:"code"`
			} `json:"data"`
		} `json:"json"`
	} `json:"error"`
}

// trpcEntries decodes a tRPC response and checks its CORS header.
func trpcEntries(t *testing.T, rec *httptest.ResponseRecorder) []trpcEntry {
	t.Helper()
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("no CORS header: %v", rec.Header())
	}
	return decodeJSONBody[[]trpcEntry](t, rec)
}

// trpcErrorCode returns the tRPC error code of a single-call response.
func trpcErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	entries := trpcEntries(t, rec)
	if len(entries) != 1 || entries[0].Error == nil {
		t.Fatalf("want one error entry, got %s", rec.Body)
	}
	return entries[0].Error.JSON.Data.Code
}

func TestKarakeepLogin(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)
	revoked := s.createToken(t, ann, nil)
	s.req("POST", "/api/app-tokens/"+revoked.ID+"/revoke", "", withCookie(ann))

	// The extension validates the key before it has a bearer token.
	rec := s.trpcMutation("apiKeys.validate", `{"0":{"json":{"apiKey":"`+token.Secret+`"}}}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `[{"result":{"data":{"json":{"success":true}}}}]`+"\n" {
		t.Fatalf("validate: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); list[1].LastUsedAt == nil {
		t.Error("validate didn't record the last use")
	}
	for name, key := range map[string]string{"revoked": revoked.Secret, "unknown": "famo_nope", "empty": ""} {
		rec := s.trpcMutation("apiKeys.validate", `{"0":{"json":{"apiKey":"`+key+`"}}}`)
		if rec.Code != http.StatusUnauthorized || trpcErrorCode(t, rec) != "UNAUTHORIZED" {
			t.Errorf("validate %s: %d %s", name, rec.Code, rec.Body)
		}
	}

	rec = s.trpcMutation("apiKeys.exchange", `{"0":{"json":{"email":"ann@example.org","password":"x","keyName":"ext"}}}`)
	if rec.Code != http.StatusUnauthorized || trpcErrorCode(t, rec) != "UNAUTHORIZED" {
		t.Errorf("exchange: %d %s", rec.Code, rec.Body)
	}

	// After login, the apps query their user and the configuration in one batch.
	rec = s.trpcQuery("users.whoami,config.clientConfig", `{"0":{"json":null},"1":{"json":null}}`, withBearer(token.Secret))
	entries := trpcEntries(t, rec)
	if rec.Code != http.StatusOK || len(entries) != 2 || entries[0].Result == nil || entries[1].Result == nil {
		t.Fatalf("batch: %d %s", rec.Code, rec.Body)
	}
	var me struct {
		ID        string  `json:"id"`
		Email     string  `json:"email"`
		Image     *string `json:"image"`
		LocalUser bool    `json:"localUser"`
	}
	if err := json.Unmarshal(entries[0].Result.Data.JSON, &me); err != nil || me.ID == "" || me.Email != "ann@example.org" || !me.LocalUser {
		t.Errorf("whoami = %+v (%v)", me, err)
	}
	var config struct {
		PublicURL string `json:"publicUrl"`
		Auth      struct {
			DisablePasswordAuth bool `json:"disablePasswordAuth"`
		} `json:"auth"`
		Inference struct {
			IsConfigured bool `json:"isConfigured"`
		} `json:"inference"`
	}
	if err := json.Unmarshal(entries[1].Result.Data.JSON, &config); err != nil || config.PublicURL != testOrigin || !config.Auth.DisablePasswordAuth || config.Inference.IsConfigured {
		t.Errorf("clientConfig = %+v (%v)", config, err)
	}
}

func TestKarakeepBearerAuth(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	expires := s.clock.Now().Add(time.Hour)
	active := s.createToken(t, ann, expires.Format(time.RFC3339))
	revoked := s.createToken(t, ann, nil)
	s.req("POST", "/api/app-tokens/"+revoked.ID+"/revoke", "", withCookie(ann))

	if rec := s.trpcQuery("users.whoami", `{}`, withBearer(active.Secret)); rec.Code != http.StatusOK {
		t.Fatalf("valid token: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); list[1].LastUsedAt == nil || !list[1].LastUsedAt.Equal(s.clock.Now()) {
		t.Errorf("last used = %v, want %v", list[1].LastUsedAt, s.clock.Now())
	}

	unknown := s.trpcQuery("users.whoami", `{}`, withBearer(apptokens.SecretPrefix+"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	if unknown.Code != http.StatusUnauthorized || trpcErrorCode(t, unknown) != "UNAUTHORIZED" {
		t.Fatalf("unknown token: %d %s", unknown.Code, unknown.Body)
	}
	for name, opts := range map[string][]func(*http.Request){
		"cookie only":    {withCookie(ann)},
		"nothing":        nil,
		"revoked":        {withBearer(revoked.Secret)},
		"basic scheme":   {func(r *http.Request) { r.Header.Set("Authorization", "Basic "+active.Secret) }},
		"cookie and bad": {withCookie(ann), withBearer("famo_nope")},
	} {
		rec := s.trpcQuery("users.whoami", `{}`, opts...)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
			t.Errorf("%s: %d %s, want 401 like an unknown token", name, rec.Code, rec.Body)
		}
	}

	// Public procedures and endpoints work without a token.
	if rec := s.trpcQuery("config.clientConfig", `{}`); rec.Code != http.StatusOK {
		t.Errorf("clientConfig without token: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/health", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"message":"Famodash is working","status":"ok"}`+"\n" {
		t.Errorf("health: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/version", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"version":"Famodash"}`+"\n" {
		t.Errorf("version: %d %s", rec.Code, rec.Body)
	}

	// Anything else of Karakeep's API doesn't exist, the REST API included.
	if rec := s.trpcQuery("bookmarks.summarizeBookmark", `{}`, withBearer(active.Secret)); rec.Code != http.StatusNotFound || trpcErrorCode(t, rec) != "NOT_FOUND" {
		t.Errorf("unknown procedure: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/v1/users/me", "", withBearer(active.Secret)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Errorf("REST path: %d %s", rec.Code, rec.Body)
	}

	s.clock.Advance(time.Hour)
	if rec := s.trpcQuery("users.whoami", `{}`, withBearer(active.Secret)); rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
		t.Errorf("expired: %d %s", rec.Code, rec.Body)
	}
}

func TestKarakeepSkipsCSRFAndAllowsCORS(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)

	rec := s.req("OPTIONS", trpcPath+"users.whoami?batch=1", "", fromExtension, func(r *http.Request) {
		r.Header.Set("Access-Control-Request-Method", "GET")
		r.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-proxy-auth")
	})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" ||
		rec.Header().Get("Access-Control-Allow-Headers") != "authorization,content-type,x-proxy-auth" {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}

	// A cross-origin request with a non-JSON body isn't rejected by the
	// cross-site check: it reaches the tRPC layer, which rejects the batch.
	rec = s.trpcMutation("apiKeys.validate", "x", withBearer(token.Secret), func(r *http.Request) {
		r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	})
	if rec.Code != http.StatusBadRequest || trpcErrorCode(t, rec) != "BAD_REQUEST" {
		t.Errorf("cross-origin POST: %d %s, want 400 BAD_REQUEST", rec.Code, rec.Body)
	}

	// The web app's API keeps its cross-site check and has no CORS headers.
	rec = s.req("POST", "/api/app-tokens", `{}`, withCookie(ann), fromExtension)
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "forbidden_origin" || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("web API cross-origin: %d %s", rec.Code, rec.Body)
	}
}
