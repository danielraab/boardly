package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"boardly/internal/aaguid"
	"boardly/internal/auth"
	"boardly/internal/webauthntest"
)

// gpmAAGUID is Google Password Manager in the aaguid test fixture.
var gpmAAGUID = [16]byte(uuid.MustParse("ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4"))

func testProviders(t *testing.T) *aaguid.Directory {
	t.Helper()
	data, err := os.ReadFile("../aaguid/testdata/providers.json")
	if err != nil {
		t.Fatal(err)
	}
	d := aaguid.New("", filepath.Join(t.TempDir(), "aaguids.json"))
	if err := os.WriteFile(d.Path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.Load(); err != nil {
		t.Fatal(err)
	}
	return d
}

func newAuthenticator(aaguid [16]byte) *webauthntest.Authenticator {
	return webauthntest.New(testOrigin, aaguid)
}

func withCookies(cookies ...*http.Cookie) func(*http.Request) {
	return func(r *http.Request) {
		for _, c := range cookies {
			if c != nil {
				r.AddCookie(c)
			}
		}
	}
}

func webAuthnCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == WebAuthnCookie {
			return c
		}
	}
	return nil
}

func session(value string) *http.Cookie { return &http.Cookie{Name: SessionCookie, Value: value} }

// registrationOptions starts adding a passkey and returns the response.
func (s *server) registrationOptions(cookie string) *httptest.ResponseRecorder {
	return s.req("POST", "/api/passkeys/registration/options", `{}`, withCookie(cookie))
}

// finishRegistration lets authn answer the options and posts the result with label.
func (s *server) finishRegistration(t *testing.T, cookie string, opts *httptest.ResponseRecorder, authn *webauthntest.Authenticator, label any) *httptest.ResponseRecorder {
	t.Helper()
	if opts.Code != http.StatusOK {
		t.Fatalf("registration options: %d %s", opts.Code, opts.Body)
	}
	cred, err := authn.Create(opts.Body.Bytes())
	if err != nil {
		t.Fatalf("authenticator: %v", err)
	}
	body := mustJSON(map[string]any{"credential": json.RawMessage(cred), "label": label})
	return s.req("POST", "/api/passkeys/registration/verify", body, withCookies(session(cookie), webAuthnCookie(opts)))
}

// addPasskey registers a passkey from authn for the (magic-link) session cookie.
func (s *server) addPasskey(t *testing.T, cookie string, authn *webauthntest.Authenticator, label any) auth.Passkey {
	t.Helper()
	rec := s.finishRegistration(t, cookie, s.registrationOptions(cookie), authn, label)
	if rec.Code != http.StatusCreated {
		t.Fatalf("registration verify: %d %s", rec.Code, rec.Body)
	}
	var p auth.Passkey
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// passkeyLogin signs in with cred of authn (nil: its newest) and returns the verify response.
func (s *server) passkeyLogin(t *testing.T, authn *webauthntest.Authenticator, cred *webauthntest.Credential) *httptest.ResponseRecorder {
	t.Helper()
	opts := s.req("POST", "/api/auth/passkey/login/options", `{}`)
	if opts.Code != http.StatusOK {
		t.Fatalf("login options: %d %s", opts.Code, opts.Body)
	}
	assertion, err := authn.Get(opts.Body.Bytes(), cred)
	if err != nil {
		t.Fatalf("authenticator: %v", err)
	}
	return s.req("POST", "/api/auth/passkey/login/verify", string(assertion), withCookies(webAuthnCookie(opts)))
}

func (s *server) listPasskeys(t *testing.T, cookie string) []auth.Passkey {
	t.Helper()
	rec := s.req("GET", "/api/passkeys", "", withCookie(cookie))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var list []auth.Passkey
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func (s *server) countPasskeys(t *testing.T) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM passkeys`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// --- 3.3 registration ---

func TestRegisterPasskey(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")

	opts := s.registrationOptions(cookie)
	var options struct {
		RP                     struct{ ID string }
		AuthenticatorSelection struct {
			ResidentKey        string `json:"residentKey"`
			RequireResidentKey bool   `json:"requireResidentKey"`
		} `json:"authenticatorSelection"`
		Attestation string `json:"attestation"`
	}
	_ = json.Unmarshal(opts.Body.Bytes(), &options)
	if options.RP.ID != "dash.example.org" || options.AuthenticatorSelection.ResidentKey != "required" || options.Attestation != "none" {
		t.Fatalf("options = %s", opts.Body)
	}
	c := webAuthnCookie(opts)
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api" || c.MaxAge != 300 {
		t.Fatalf("challenge cookie = %+v", c)
	}

	authn := newAuthenticator(gpmAAGUID)
	rec := s.finishRegistration(t, cookie, opts, authn, "  Work laptop  ")
	if rec.Code != http.StatusCreated {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	if c := webAuthnCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Fatalf("challenge cookie not cleared: %+v", c)
	}

	list := s.listPasskeys(t, cookie)
	if len(list) != 1 || list[0].Label == nil || *list[0].Label != "Work laptop" {
		t.Fatalf("list = %+v", list)
	}
	if list[0].Provider == nil || list[0].Provider.Name != "Google Password Manager" || list[0].Provider.IconDark == nil {
		t.Fatalf("provider = %+v", list[0].Provider)
	}
	if list[0].LastUsedAt != nil || !list[0].CreatedAt.Equal(s.clock.Now()) {
		t.Fatalf("dates = %v, %v", list[0].CreatedAt, list[0].LastUsedAt)
	}
}

func TestRegisterPasskeyWithoutLabelUnknownProvider(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	p := s.addPasskey(t, cookie, newAuthenticator([16]byte{}), "   ")
	if p.Label != nil || p.Provider != nil {
		t.Fatalf("passkey = %+v; want no label and no provider", p)
	}
	raw := s.req("GET", "/api/passkeys", "", withCookie(cookie)).Body.String()
	if !strings.Contains(raw, `"label":null`) || !strings.Contains(raw, `"provider":null`) {
		t.Fatalf("list = %s", raw)
	}
}

func TestRegisterPasskeyWindowClosed(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	s.clock.Advance(6 * time.Minute)

	rec := s.registrationOptions(cookie)
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "enrollment_window_closed" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if webAuthnCookie(rec) != nil {
		t.Fatal("challenge issued for a closed window")
	}
}

func TestRegisterPasskeyWindowClosesDuringPrompt(t *testing.T) {
	s := newServer(t)
	start := s.clock.Now()
	cookie := s.signIn(t, "ann@example.org")
	s.clock.Advance(start.Add(4*time.Minute + 50*time.Second).Sub(s.clock.Now()))

	opts := s.registrationOptions(cookie)
	s.clock.Advance(30 * time.Second)
	rec := s.finishRegistration(t, cookie, opts, newAuthenticator(gpmAAGUID), nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if s.countPasskeys(t) != 1 {
		t.Fatal("passkey not saved")
	}
}

func TestRegisterPasskeyLabelTooLong(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")

	rec := s.finishRegistration(t, cookie, s.registrationOptions(cookie), newAuthenticator(gpmAAGUID), strings.Repeat("ä", 65))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_label" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if s.countPasskeys(t) != 0 {
		t.Fatal("passkey saved despite invalid label")
	}

	// 64 characters are fine.
	s.addPasskey(t, cookie, newAuthenticator(gpmAAGUID), strings.Repeat("ä", 64))
}

func TestRegisterPasskeyExcludesExisting(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	authn := newAuthenticator(gpmAAGUID)
	s.addPasskey(t, cookie, authn, nil)

	if _, err := authn.Create(s.registrationOptions(cookie).Body.Bytes()); !errors.Is(err, webauthntest.ErrExcluded) {
		t.Fatalf("second registration on the same authenticator: err = %v, want ErrExcluded", err)
	}
}

func TestRegisterPasskeyChallengeBoundToSession(t *testing.T) {
	s := newServer(t)
	first := s.signIn(t, "ann@example.org")
	second := s.signIn(t, "ann@example.org")

	opts := s.registrationOptions(first)
	cred, err := newAuthenticator(gpmAAGUID).Create(opts.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	body := mustJSON(map[string]any{"credential": json.RawMessage(cred)})
	rec := s.req("POST", "/api/passkeys/registration/verify", body, withCookies(session(second), webAuthnCookie(opts)))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_challenge" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	// Without a challenge at all.
	rec = s.req("POST", "/api/passkeys/registration/verify", body, withCookie(first))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_challenge" {
		t.Fatalf("no challenge: %d %s", rec.Code, rec.Body)
	}
	if s.countPasskeys(t) != 0 {
		t.Fatal("passkey saved")
	}
}

func TestPasskeyEndpointsRequireSignIn(t *testing.T) {
	s := newServer(t)
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/passkeys"},
		{"POST", "/api/passkeys/registration/options"},
		{"POST", "/api/passkeys/registration/verify"},
		{"DELETE", "/api/passkeys/x"},
	} {
		if rec := s.req(tc.method, tc.path, `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d", tc.method, tc.path, rec.Code)
		}
	}
}

// --- 3.4 sign-in ---

func TestPasskeyLogin(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	s.addPasskey(t, s.signIn(t, "ann@example.org"), authn, nil)
	s.clock.Advance(time.Hour)

	rec := s.passkeyLogin(t, authn, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	if u := decodeUser(t, rec); u.Email != "ann@example.org" {
		t.Fatalf("user = %+v", u)
	}
	c := sessionCookie(rec)
	if c == nil || c.Value == "" {
		t.Fatal("no session cookie")
	}

	// A passkey session never opens the enrollment window.
	me := s.req("GET", "/api/me", "", withCookie(c.Value))
	if me.Code != http.StatusOK || enrollmentUntil(t, me) != nil {
		t.Fatalf("me: %d %s", me.Code, me.Body)
	}
	if rec := s.registrationOptions(c.Value); rec.Code != http.StatusForbidden {
		t.Fatalf("registration from passkey session: %d", rec.Code)
	}
	var method string
	_ = s.db.QueryRow(`SELECT auth_method FROM sessions ORDER BY created_at DESC LIMIT 1`).Scan(&method)
	if method != "passkey" {
		t.Fatalf("auth_method = %q", method)
	}

	var signCount int
	var lastUsed *string
	_ = s.db.QueryRow(`SELECT sign_count, last_used_at FROM passkeys`).Scan(&signCount, &lastUsed)
	if signCount != 1 || lastUsed == nil {
		t.Fatalf("sign_count = %d, last_used_at = %v", signCount, lastUsed)
	}
}

func TestPasskeyLoginSecondUser(t *testing.T) {
	s := newServer(t)
	ann, bob := newAuthenticator(gpmAAGUID), newAuthenticator(gpmAAGUID)
	s.addPasskey(t, s.signIn(t, "ann@example.org"), ann, nil)
	s.addPasskey(t, s.signIn(t, "bob@example.org"), bob, nil)

	rec := s.passkeyLogin(t, bob, nil)
	if rec.Code != http.StatusOK || decodeUser(t, rec).Email != "bob@example.org" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestPasskeyLoginDeletedCredential(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	cookie := s.signIn(t, "ann@example.org")
	p := s.addPasskey(t, cookie, authn, nil)
	if rec := s.req("DELETE", "/api/passkeys/"+p.ID, "", withCookie(cookie)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}

	rec := s.passkeyLogin(t, authn, nil)
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "passkey_unknown" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if sessionCookie(rec) != nil {
		t.Fatal("session started")
	}
}

func TestPasskeyLoginBadSignature(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	s.addPasskey(t, s.signIn(t, "ann@example.org"), authn, nil)
	authn.BadSignature = true

	rec := s.passkeyLogin(t, authn, nil)
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "passkey_invalid" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if sessionCookie(rec) != nil {
		t.Fatal("session started")
	}
}

func TestPasskeyLoginChallengeIsSingleUse(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	s.addPasskey(t, s.signIn(t, "ann@example.org"), authn, nil)

	opts := s.req("POST", "/api/auth/passkey/login/options", `{}`)
	assertion, err := authn.Get(opts.Body.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec := s.req("POST", "/api/auth/passkey/login/verify", string(assertion), withCookies(webAuthnCookie(opts))); rec.Code != http.StatusOK {
		t.Fatalf("first: %d %s", rec.Code, rec.Body)
	}
	rec := s.req("POST", "/api/auth/passkey/login/verify", string(assertion), withCookies(webAuthnCookie(opts)))
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "passkey_invalid" {
		t.Fatalf("replay: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("POST", "/api/auth/passkey/login/verify", string(assertion))
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "passkey_invalid" {
		t.Fatalf("no challenge cookie: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("POST", "/api/auth/passkey/login/verify", `not json`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("garbage: %d", rec.Code)
	}
}

// --- 3.5 list and delete ---

func TestListPasskeysOnlyOwnNewestFirst(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	first := s.addPasskey(t, ann, newAuthenticator(gpmAAGUID), "first")
	s.clock.Advance(time.Second)
	second := s.addPasskey(t, ann, newAuthenticator(gpmAAGUID), "second")
	bobs := s.addPasskey(t, bob, newAuthenticator(gpmAAGUID), "bob")

	list := s.listPasskeys(t, ann)
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("ann's list = %+v", list)
	}
	list = s.listPasskeys(t, bob)
	if len(list) != 1 || list[0].ID != bobs.ID {
		t.Fatalf("bob's list = %+v", list)
	}
}

func TestDeletePasskeyOutsideWindow(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	s.addPasskey(t, s.signIn(t, "ann@example.org"), authn, nil)
	other := s.addPasskey(t, s.signIn(t, "ann@example.org"), newAuthenticator([16]byte{}), nil)

	login := s.passkeyLogin(t, authn, nil)
	cookie := sessionCookie(login).Value
	s.clock.Advance(time.Hour)

	// Delete the passkey this session was created with.
	list := s.listPasskeys(t, cookie)
	var used string
	for _, p := range list {
		if p.ID != other.ID {
			used = p.ID
		}
	}
	if rec := s.req("DELETE", "/api/passkeys/"+used, "", withCookie(cookie)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if list := s.listPasskeys(t, cookie); len(list) != 1 || list[0].ID != other.ID {
		t.Fatalf("list after delete = %+v", list)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(cookie)); rec.Code != http.StatusOK {
		t.Fatalf("session ended by deleting its passkey: %d", rec.Code)
	}
	if rec := s.passkeyLogin(t, authn, nil); errorCode(t, rec) != "passkey_unknown" {
		t.Fatalf("deleted passkey still signs in: %d %s", rec.Code, rec.Body)
	}
}

func TestDeleteForeignOrMissingPasskey(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	bobs := s.addPasskey(t, bob, newAuthenticator(gpmAAGUID), nil)

	for _, id := range []string{bobs.ID, uuid.NewString()} {
		rec := s.req("DELETE", "/api/passkeys/"+id, "", withCookie(ann))
		if rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Fatalf("delete %s: %d %s", id, rec.Code, rec.Body)
		}
	}
	if len(s.listPasskeys(t, bob)) != 1 {
		t.Fatal("foreign passkey deleted")
	}
}

func TestPasskeyCeremonyFromOtherOriginRejected(t *testing.T) {
	s := newServer(t)
	authn := newAuthenticator(gpmAAGUID)
	cookie := s.signIn(t, "ann@example.org")
	s.addPasskey(t, cookie, authn, nil)

	authn.Origin = "https://evil.example.org"
	if rec := s.passkeyLogin(t, authn, nil); rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "passkey_invalid" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	evil := newAuthenticator(gpmAAGUID)
	evil.Origin = authn.Origin
	rec := s.finishRegistration(t, cookie, s.registrationOptions(cookie), evil, nil)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "passkey_invalid" {
		t.Fatalf("registration: %d %s", rec.Code, rec.Body)
	}
}
