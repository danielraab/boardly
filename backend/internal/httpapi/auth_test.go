package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/dashboards"
	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/jobs"
	"boardly/internal/mail"
	"boardly/internal/notes"
	"boardly/internal/users"
	"boardly/internal/widgets"
)

const testOrigin = "https://dash.example.org"

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type outbox struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (o *outbox) Send(_ context.Context, m mail.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sent = append(o.sent, m)
	return nil
}

func (o *outbox) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.sent)
}

func (o *outbox) last(t *testing.T) mail.Message {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.sent) == 0 {
		t.Fatal("no mail sent")
	}
	return o.sent[len(o.sent)-1]
}

type server struct {
	h        http.Handler
	db       *sql.DB
	auth     *auth.Service
	clock    *clock
	mail     *outbox
	images   *images.Store
	favicons *images.Favicons
	previews *images.Previews
	jobs     *jobs.Queue
}

func newServer(t *testing.T) *server {
	t.Helper()
	database := dbtest.Open(t)
	c := &clock{now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	box := &outbox{}
	base, _ := url.Parse(testOrigin)

	store := users.NewStore(database)
	store.Now = c.Now
	svc := auth.NewService(database, store, box, base)
	svc.Now = c.Now
	var err error
	if svc.WebAuthn, err = auth.NewWebAuthn(base.Hostname(), testOrigin); err != nil {
		t.Fatal(err)
	}
	svc.Providers = testProviders(t)

	imageStore, err := images.NewStore(database, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	imageStore.Now = c.Now
	queue := jobs.New(database)
	queue.PollInterval = 20 * time.Millisecond
	favicons := images.NewFavicons(imageStore, false)
	favicons.Now = c.Now
	favicons.Jobs = queue
	previews, err := images.NewPreviews(imageStore, queue, favicons.Client, 1<<20, images.Converter{MaxPixels: 25_000_000, Width: 800})
	if err != nil {
		t.Fatal(err)
	}
	favouriteStore, dashboardStore := favourites.NewStore(database), dashboards.NewStore(database)
	favouriteStore.Jobs = queue
	noteStore := notes.NewStore(database)
	widgets.Register(dashboardStore, favouriteStore, noteStore, favicons)

	appTokens := apptokens.NewStore(database)
	appTokens.Now = c.Now

	h := New(Deps{
		DB: database, Web: fakeWeb, Auth: svc, Users: store, Favourites: favouriteStore, Dashboards: dashboardStore, Notes: noteStore,
		Images: imageStore, Favicons: favicons, Previews: previews, AppTokens: appTokens,
		Origin: testOrigin, SecureCookies: true,
	})
	return &server{h: h, db: database, auth: svc, clock: c, mail: box, images: imageStore, favicons: favicons, previews: previews, jobs: queue}
}

// req sends a same-origin JSON request; opts can adjust it.
func (s *server) req(method, path, body string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.1:4000"
	if method != http.MethodGet {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", testOrigin)
	}
	for _, o := range opts {
		o(r)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)
	return rec
}

func withCookie(value string) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: SessionCookie, Value: value}) }
}

func fromIP(ip string) func(*http.Request) {
	return func(r *http.Request) { r.RemoteAddr = ip + ":4000" }
}

var tokenRE = regexp.MustCompile(`/login/verify#token=([A-Za-z0-9_-]+)`)

// requestToken requests a link for email and returns the token from the mail.
func (s *server) requestToken(t *testing.T, email string) string {
	t.Helper()
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"`+email+`","locale":"en"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("request link: %d %s", rec.Code, rec.Body)
	}
	m := tokenRE.FindStringSubmatch(s.mail.last(t).Text)
	if m == nil {
		t.Fatalf("no link in mail %q", s.mail.last(t).Text)
	}
	return m[1]
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie {
			return c
		}
	}
	return nil
}

// signIn requests and redeems a link and returns the session cookie value.
func (s *server) signIn(t *testing.T, email string) string {
	t.Helper()
	token := s.requestToken(t, email)
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	c := sessionCookie(rec)
	if c == nil {
		t.Fatal("no session cookie")
	}
	s.clock.Advance(61 * time.Second) // keep later link requests clear of the per-email limit
	return c.Value
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) users.User {
	t.Helper()
	var u users.User
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("invalid user JSON %q: %v", rec.Body, err)
	}
	return u
}

// --- 3.1 request a link ---

func TestRequestLinkValid(t *testing.T) {
	s := newServer(t)
	rec := s.req("POST", "/api/auth/magic-link", `{"email":" Ann@Example.org ","locale":"de"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d %s", rec.Code, rec.Body)
	}
	m := s.mail.last(t)
	if m.To != "ann@example.org" {
		t.Fatalf("to = %q", m.To)
	}
	if m.Subject != "Dein Anmeldelink für Boardly" {
		t.Fatalf("subject = %q, want the German one", m.Subject)
	}
	if !strings.Contains(m.Text, testOrigin+"/login/verify#token=") {
		t.Fatalf("link format wrong in %q", m.Text)
	}
}

func TestRequestLinkUnsupportedLocaleFallsBackToEnglish(t *testing.T) {
	s := newServer(t)
	s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org","locale":"fr"}`)
	if got := s.mail.last(t).Subject; got != "Your Boardly sign-in link" {
		t.Fatalf("subject = %q", got)
	}
}

func TestRequestLinkInvalidEmail(t *testing.T) {
	s := newServer(t)
	for _, email := range []string{"not-an-email", "", "Ann <ann@example.org>", "ann@localhost", "a@b@example.org"} {
		rec := s.req("POST", "/api/auth/magic-link", `{"email":`+mustJSON(email)+`}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%q: status = %d", email, rec.Code)
		}
		if code := errorCode(t, rec); code != "invalid_email" {
			t.Fatalf("%q: code = %q", email, code)
		}
	}
	if n := s.mail.count(); n != 0 {
		t.Fatalf("sent %d mails, want 0", n)
	}
}

func TestRequestLinkDoesNotRevealAccounts(t *testing.T) {
	s := newServer(t)
	s.signIn(t, "known@example.org")

	known := s.req("POST", "/api/auth/magic-link", `{"email":"known@example.org"}`)
	unknown := s.req("POST", "/api/auth/magic-link", `{"email":"unknown@example.org"}`)
	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
		t.Fatalf("status = %d / %d", known.Code, unknown.Code)
	}
	if known.Body.String() != unknown.Body.String() {
		t.Fatalf("bodies differ: %q vs %q", known.Body, unknown.Body)
	}
}

func TestClosedRegistrationDoesNotRevealAccounts(t *testing.T) {
	s := newServer(t)
	s.signIn(t, "known@example.org")
	s.auth.RegistrationClosed = true
	const browser = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"

	known := s.req("POST", "/api/auth/magic-link", `{"email":"known@example.org"}`, withLoginRequest(browser))
	if s.mail.last(t).To != "known@example.org" || !tokenRE.MatchString(s.mail.last(t).Text) {
		t.Fatalf("known address got %+v", s.mail.last(t))
	}
	unknown := s.req("POST", "/api/auth/magic-link", `{"email":"unknown@example.org"}`, withLoginRequest(browser))
	if m := s.mail.last(t); m.To != "unknown@example.org" || tokenRE.MatchString(m.Text) {
		t.Fatalf("unknown address got %+v", m)
	}
	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
		t.Fatalf("status = %d / %d", known.Code, unknown.Code)
	}
	if known.Body.String() != unknown.Body.String() {
		t.Fatalf("bodies differ: %q vs %q", known.Body, unknown.Body)
	}
	if k, u := known.Header().Get("Set-Cookie"), unknown.Header().Get("Set-Cookie"); k == "" || k != u {
		t.Fatalf("Set-Cookie differs: %q vs %q", k, u)
	}
}

func TestTokenNotStoredInPlainText(t *testing.T) {
	s := newServer(t)
	token := s.requestToken(t, "ann@example.org")

	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM magic_link_tokens
		WHERE instr(CAST(token_hash AS TEXT), ?) > 0 OR email = ? OR locale = ? OR request_ip = ?`,
		token, token, token, token).Scan(&n)
	if err != nil || n != 0 {
		t.Fatalf("plain token found in %d rows (err %v)", n, err)
	}
	var hashLen int
	if err := s.db.QueryRow(`SELECT length(token_hash) FROM magic_link_tokens`).Scan(&hashLen); err != nil || hashLen != 32 {
		t.Fatalf("hash length = %d (err %v), want 32", hashLen, err)
	}
	if len(token) < 43 { // 32 bytes base64url
		t.Fatalf("token %q shorter than 256 bits", token)
	}
}

// --- 3.2 rate limits ---

func TestRateLimitImmediateRepeat(t *testing.T) {
	s := newServer(t)
	s.requestToken(t, "ann@example.org")
	s.clock.Advance(10 * time.Second)

	rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "rate_limited" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Retry-After"); got != "50" {
		t.Fatalf("Retry-After = %q, want 50", got)
	}
	if n := s.mail.count(); n != 1 {
		t.Fatalf("sent %d mails, want 1", n)
	}
}

func TestRateLimitSixthPerHour(t *testing.T) {
	s := newServer(t)
	for i := range 5 {
		rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`, fromIP("192.0.2."+string(rune('1'+i))))
		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
		s.clock.Advance(61 * time.Second)
	}
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`, fromIP("192.0.2.9"))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("sixth request: %d", rec.Code)
	}
	// The first request was 5*61s ago, so it ages out after another 60min-305s.
	if got := rec.Header().Get("Retry-After"); got != "3295" {
		t.Fatalf("Retry-After = %q, want 3295", got)
	}
}

func TestRateLimitPerIP(t *testing.T) {
	s := newServer(t)
	for i := range 5 {
		rec := s.req("POST", "/api/auth/magic-link", `{"email":"user`+string(rune('a'+i))+`@example.org"}`)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
	}
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"userz@example.org"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("sixth request from the same IP: %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	if rec := s.req("POST", "/api/auth/magic-link", `{"email":"userz@example.org"}`, fromIP("198.51.100.1")); rec.Code != http.StatusAccepted {
		t.Fatalf("other IP: %d", rec.Code)
	}
}

// --- 3.3 redeem ---

func TestVerifyFirstSignInCreatesAccount(t *testing.T) {
	s := newServer(t)
	token := s.requestToken(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`, func(r *http.Request) { r.Header.Set("User-Agent", "test-agent") })
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rec.Code, rec.Body)
	}
	u := decodeUser(t, rec)
	if u.ID == "" || u.Email != "ann@example.org" || u.Name != nil {
		t.Fatalf("user = %+v", u)
	}
	if !strings.Contains(rec.Body.String(), `"name":null`) {
		t.Fatalf("name not null in %s", rec.Body)
	}
	if sessionCookie(rec) == nil {
		t.Fatal("no session cookie")
	}
	var method, ua string
	if err := s.db.QueryRow(`SELECT auth_method, user_agent FROM sessions`).Scan(&method, &ua); err != nil || method != "magic_link" || ua != "test-agent" {
		t.Fatalf("session row: %q %q %v", method, ua, err)
	}
}

func TestVerifyExistingAccount(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	first := decodeUser(t, s.req("GET", "/api/me", "", withCookie(cookie)))

	token := s.requestToken(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
	if rec.Code != http.StatusOK || decodeUser(t, rec).ID != first.ID {
		t.Fatalf("got %d %s, want user %s", rec.Code, rec.Body, first.ID)
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n != 1 {
		t.Fatalf("users = %d, want 1", n)
	}
}

func TestVerifyInvalidTokens(t *testing.T) {
	s := newServer(t)
	used := s.requestToken(t, "used@example.org")
	if rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+used+`","confirm":true}`); rec.Code != http.StatusOK {
		t.Fatalf("first use: %d", rec.Code)
	}
	expired := s.requestToken(t, "expired@example.org")
	s.clock.Advance(16 * time.Minute)

	for name, token := range map[string]string{"reused": used, "expired": expired, "unknown": "nope", "empty": ""} {
		t.Run(name, func(t *testing.T) {
			rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
			if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_token" {
				t.Fatalf("got %d %s", rec.Code, rec.Body)
			}
			if sessionCookie(rec) != nil {
				t.Fatal("session cookie set")
			}
		})
	}
}

func TestVerifyConcurrentRedemptionsOnlyOneSucceeds(t *testing.T) {
	s := newServer(t)
	token := s.requestToken(t, "ann@example.org")

	const n = 8
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			codes <- s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`).Code
		})
	}
	wg.Wait()
	close(codes)
	ok := 0
	for c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusBadRequest:
		default:
			t.Fatalf("unexpected status %d", c)
		}
	}
	if ok != 1 {
		t.Fatalf("%d redemptions succeeded, want 1", ok)
	}
}

// --- 4.1 sessions ---

func TestSessionCookieAttributes(t *testing.T) {
	s := newServer(t)
	token := s.requestToken(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
	raw := rec.Header().Get("Set-Cookie")
	for _, want := range []string{SessionCookie + "=", "Path=/", "HttpOnly", "Secure", "SameSite=Lax", "Max-Age="} {
		if !strings.Contains(raw, want) {
			t.Fatalf("Set-Cookie %q lacks %q", raw, want)
		}
	}
	var stored int
	c := sessionCookie(rec)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE CAST(id_hash AS TEXT) = ?`, c.Value).Scan(&stored)
	if stored != 0 {
		t.Fatal("session secret stored in plain text")
	}
}

func TestSessionExpiresAfter30DaysWithoutUse(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	s.clock.Advance(31 * 24 * time.Hour)
	rec := s.req("GET", "/api/me", "", withCookie(cookie))
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "unauthenticated" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestSessionUseExtendsExpiry(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	for range 3 {
		s.clock.Advance(20 * 24 * time.Hour)
		if rec := s.req("GET", "/api/me", "", withCookie(cookie)); rec.Code != http.StatusOK {
			t.Fatalf("status = %d after regular use", rec.Code)
		}
	}
}

func TestSessionTouchIsThrottled(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	lastUsed := func() string {
		var v string
		_ = s.db.QueryRow(`SELECT last_used_at FROM sessions`).Scan(&v)
		return v
	}
	before := lastUsed()

	s.clock.Advance(30 * time.Minute)
	rec := s.req("GET", "/api/me", "", withCookie(cookie))
	if lastUsed() != before || sessionCookie(rec) != nil {
		t.Fatal("session touched within the hour")
	}

	s.clock.Advance(31 * time.Minute)
	rec = s.req("GET", "/api/me", "", withCookie(cookie))
	if lastUsed() == before {
		t.Fatal("session not touched after an hour")
	}
	if c := sessionCookie(rec); c == nil || c.Value != cookie || c.MaxAge <= 0 {
		t.Fatalf("cookie not renewed: %+v", c)
	}
}

// --- 4.2 CSRF ---

func TestCSRFForeignOrigin(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/logout", "", withCookie(cookie), func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") })
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "forbidden_origin" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(cookie)); rec.Code != http.StatusOK {
		t.Fatalf("session no longer valid: %d", rec.Code)
	}
}

func TestCSRFRequiresJSONContentType(t *testing.T) {
	s := newServer(t)
	for _, ct := range []string{"", "text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x"} {
		rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`, func(r *http.Request) {
			if ct == "" {
				r.Header.Del("Content-Type")
			} else {
				r.Header.Set("Content-Type", ct)
			}
		})
		if rec.Code != http.StatusForbidden || errorCode(t, rec) != "forbidden_origin" {
			t.Fatalf("content type %q: %d %s", ct, rec.Code, rec.Body)
		}
	}
	if n := s.mail.count(); n != 0 {
		t.Fatalf("sent %d mails", n)
	}
}

func TestCSRFSameOriginPasses(t *testing.T) {
	s := newServer(t)
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	// Without an Origin header (non-browser clients) the request passes too.
	s.clock.Advance(time.Minute)
	rec = s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`, func(r *http.Request) { r.Header.Del("Origin") })
	if rec.Code != http.StatusAccepted {
		t.Fatalf("without Origin: %d %s", rec.Code, rec.Body)
	}
}

// --- 4.3 me and logout ---

func TestMe(t *testing.T) {
	s := newServer(t)
	if rec := s.req("GET", "/api/me", ""); rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "unauthenticated" {
		t.Fatalf("signed out: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/me", "", withCookie("garbage")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown session: %d", rec.Code)
	}

	cookie := s.signIn(t, "ann@example.org")
	rec := s.req("GET", "/api/me", "", withCookie(cookie))
	if rec.Code != http.StatusOK {
		t.Fatalf("signed in: %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if slug, _ := body["slug"].(string); len(body) != 6 || body["shareFolderId"] != nil || body["email"] != "ann@example.org" || body["name"] != nil || body["id"] == "" || len(slug) != 10 {
		t.Fatalf("body = %v", body)
	}
}

func TestPatchMeName(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	patch := func(body string) *httptest.ResponseRecorder {
		return s.req("PATCH", "/api/me", body, withCookie(cookie))
	}

	rec := patch(`{"name":"  Ann Smith  "}`)
	if rec.Code != http.StatusOK || decodeUser(t, rec).Name == nil || *decodeUser(t, rec).Name != "Ann Smith" {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}

	long := strings.Repeat("x", 101)
	rec = patch(`{"name":"` + long + `"}`)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_name" {
		t.Fatalf("too long: %d %s", rec.Code, rec.Body)
	}
	if u := decodeUser(t, s.req("GET", "/api/me", "", withCookie(cookie))); u.Name == nil || *u.Name != "Ann Smith" {
		t.Fatalf("name changed by rejected request: %+v", u)
	}

	if rec := patch(`{"name":42}`); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_name" {
		t.Fatalf("number: %d %s", rec.Code, rec.Body)
	}
	if rec := patch(`{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing name: %d", rec.Code)
	}

	for _, clear := range []string{`{"name":"   "}`, `{"name":""}`, `{"name":null}`} {
		patch(`{"name":"Ann"}`)
		rec = patch(clear)
		if rec.Code != http.StatusOK || decodeUser(t, rec).Name != nil {
			t.Fatalf("%s: %d %s", clear, rec.Code, rec.Body)
		}
	}

	if rec := s.req("PATCH", "/api/me", `{"name":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("signed out patch: %d", rec.Code)
	}
}

func TestPatchMeSlug(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	patch := func(cookie, body string) *httptest.ResponseRecorder {
		return s.req("PATCH", "/api/me", body, withCookie(cookie))
	}

	if rec := patch(bob, `{"slug":"Bob"}`); rec.Code != http.StatusOK || decodeUser(t, rec).Slug != "bob" {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	for body, want := range map[string]string{
		`{"slug":"a_nn"}`: "invalid_slug",
		`{"slug":42}`:     "invalid_slug",
		`{"slug":"API"}`:  "slug_reserved",
	} {
		if rec := patch(cookie, body); rec.Code != http.StatusBadRequest || errorCode(t, rec) != want {
			t.Errorf("%s: %d %s, want %s", body, rec.Code, rec.Body, want)
		}
	}
	rec := patch(cookie, `{"name":"Ann","slug":"bob"}`)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "slug_taken" {
		t.Fatalf("taken: %d %s", rec.Code, rec.Body)
	}
	if u := decodeUser(t, s.req("GET", "/api/me", "", withCookie(cookie))); u.Name != nil || u.Slug == "bob" {
		t.Fatalf("rejected request changed the user: %+v", u)
	}
	rec = patch(cookie, `{"name":"Ann","slug":"ann"}`)
	if u := decodeUser(t, rec); rec.Code != http.StatusOK || u.Slug != "ann" || u.Name == nil || *u.Name != "Ann" {
		t.Fatalf("both: %d %s", rec.Code, rec.Body)
	}
}

func TestPatchMeShareFolder(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	patch := func(body string) *httptest.ResponseRecorder {
		return s.req("PATCH", "/api/me", body, withCookie(cookie))
	}
	inbox := s.create(t, cookie, map[string]any{"kind": "folder", "title": "Inbox"})
	link := s.create(t, cookie, map[string]any{"kind": "link", "parentId": inbox.ID, "url": "https://example.org"})
	bobs := s.create(t, bob, map[string]any{"kind": "folder", "title": "Bob"})

	rec := patch(`{"shareFolderId":"` + inbox.ID + `"}`)
	if u := decodeUser(t, rec); rec.Code != http.StatusOK || u.ShareFolderID == nil || *u.ShareFolderID != inbox.ID {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{
		`{"shareFolderId":"` + link.ID + `"}`,
		`{"shareFolderId":"` + bobs.ID + `"}`,
		`{"shareFolderId":"unknown"}`,
		`{"shareFolderId":42}`,
	} {
		if rec := patch(body); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_folder" {
			t.Errorf("%s: %d %s, want invalid_folder", body, rec.Code, rec.Body)
		}
	}
	if u := decodeUser(t, s.req("GET", "/api/me", "", withCookie(cookie))); u.ShareFolderID == nil || *u.ShareFolderID != inbox.ID {
		t.Fatalf("rejected request changed the folder: %+v", u)
	}

	root := s.tree(t, cookie)
	if rec := patch(`{"shareFolderId":"` + root.ID + `"}`); rec.Code != http.StatusOK || decodeUser(t, rec).ShareFolderID != nil {
		t.Fatalf("root: %d %s", rec.Code, rec.Body)
	}
	patch(`{"shareFolderId":"` + inbox.ID + `"}`)
	if rec := patch(`{"shareFolderId":null}`); rec.Code != http.StatusOK || decodeUser(t, rec).ShareFolderID != nil {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}
}

func TestLogout(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")

	rec := s.req("POST", "/api/auth/logout", "", withCookie(cookie))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body)
	}
	if c := sessionCookie(rec); c == nil || c.MaxAge >= 0 || c.Value != "" {
		t.Fatalf("cookie not cleared: %+v", c)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(cookie)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old cookie still valid: %d", rec.Code)
	}
	if rec := s.req("POST", "/api/auth/logout", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("logout without session: %d", rec.Code)
	}
}

func TestMalformedJSON(t *testing.T) {
	s := newServer(t)
	rec := s.req("POST", "/api/auth/magic-link", `{"email":`)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_request" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// --- 9 same-browser sign-in ---

func loginRequestCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == LoginRequestCookie {
			return c
		}
	}
	return nil
}

func withLoginRequest(value string) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: LoginRequestCookie, Value: value}) }
}

// requestFromBrowser requests a link and returns the token and the login-request cookie value.
func (s *server) requestFromBrowser(t *testing.T, email string, opts ...func(*http.Request)) (string, string) {
	t.Helper()
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"`+email+`"}`, opts...)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("request link: %d %s", rec.Code, rec.Body)
	}
	c := loginRequestCookie(rec)
	if c == nil {
		t.Fatal("no login-request cookie")
	}
	m := tokenRE.FindStringSubmatch(s.mail.last(t).Text)
	return m[1], c.Value
}

func TestLoginRequestCookieAttributes(t *testing.T) {
	s := newServer(t)
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`)
	raw := rec.Header().Get("Set-Cookie")
	for _, want := range []string{LoginRequestCookie + "=", "Path=/api/auth/magic-link", "Max-Age=900", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("Set-Cookie %q lacks %q", raw, want)
		}
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM magic_link_tokens WHERE length(browser_hash) = 32`).Scan(&n)
	if n != 1 {
		t.Fatalf("tokens with browser hash = %d, want 1", n)
	}
}

func TestLoginRequestCookieIsReused(t *testing.T) {
	s := newServer(t)
	_, first := s.requestFromBrowser(t, "ann@example.org")
	_, second := s.requestFromBrowser(t, "bob@example.org", withLoginRequest(first))
	if second != first {
		t.Fatalf("cookie changed from %q to %q", first, second)
	}
	// A value this server can't have issued is replaced.
	_, third := s.requestFromBrowser(t, "cid@example.org", withLoginRequest("forged"))
	if third == "forged" || third == first {
		t.Fatalf("got %q", third)
	}
}

func TestNoLoginRequestCookieOnRejectedRequests(t *testing.T) {
	s := newServer(t)
	if rec := s.req("POST", "/api/auth/magic-link", `{"email":"nope"}`); loginRequestCookie(rec) != nil {
		t.Fatal("cookie set on 400")
	}
	s.requestFromBrowser(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/magic-link", `{"email":"ann@example.org"}`)
	if rec.Code != http.StatusTooManyRequests || loginRequestCookie(rec) != nil {
		t.Fatalf("got %d with cookie %v", rec.Code, loginRequestCookie(rec))
	}
}

func TestSameBrowserSignsInWithoutConfirmation(t *testing.T) {
	s := newServer(t)
	token, browser := s.requestFromBrowser(t, "ann@example.org")
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`"}`, withLoginRequest(browser))
	if rec.Code != http.StatusOK || sessionCookie(rec) == nil {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if c := loginRequestCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Fatalf("login-request cookie not cleared: %+v", c)
	}
}

func TestOtherBrowserNeedsConfirmation(t *testing.T) {
	s := newServer(t)
	token, _ := s.requestFromBrowser(t, "ann@example.org")
	_, otherBrowser := s.requestFromBrowser(t, "bob@example.org")

	for name, opts := range map[string][]func(*http.Request){
		"no cookie":      nil,
		"foreign cookie": {withLoginRequest(otherBrowser)},
	} {
		t.Run(name, func(t *testing.T) {
			rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`"}`, opts...)
			if rec.Code != http.StatusForbidden || errorCode(t, rec) != "confirmation_required" {
				t.Fatalf("got %d %s", rec.Code, rec.Body)
			}
			if sessionCookie(rec) != nil {
				t.Fatal("session cookie set")
			}
		})
	}

	// The link is still usable after the unconfirmed attempts (e.g. by a mail scanner).
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
	if rec.Code != http.StatusOK || sessionCookie(rec) == nil {
		t.Fatalf("confirmed: %d %s", rec.Code, rec.Body)
	}
}

func TestUnconfirmedInvalidTokenIsStillInvalid(t *testing.T) {
	s := newServer(t)
	token, browser := s.requestFromBrowser(t, "ann@example.org")
	s.clock.Advance(16 * time.Minute)
	for _, body := range []string{`{"token":"nope"}`, `{"token":"` + token + `"}`} {
		rec := s.req("POST", "/api/auth/magic-link/verify", body, withLoginRequest(browser))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_token" {
			t.Fatalf("%s: got %d %s", body, rec.Code, rec.Body)
		}
	}
}

// --- passkeys 3.2: enrollment window ---

func enrollmentUntil(t *testing.T, rec *httptest.ResponseRecorder) any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	v, ok := body["passkeyEnrollmentUntil"]
	if !ok {
		t.Fatalf("no passkeyEnrollmentUntil in %s", rec.Body)
	}
	return v
}

func TestMeEnrollmentWindowAfterMagicLink(t *testing.T) {
	s := newServer(t)
	signedInAt := s.clock.Now()
	want := signedInAt.Add(5 * time.Minute).Format(time.RFC3339)

	// The verify response already carries the window, so the client cache has it.
	verify := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+s.requestToken(t, "ann@example.org")+`","confirm":true}`)
	if got := enrollmentUntil(t, verify); got != want {
		t.Fatalf("verify passkeyEnrollmentUntil = %v, want %s", got, want)
	}
	cookie := sessionCookie(verify).Value
	s.clock.Advance(61 * time.Second)

	rec := s.req("GET", "/api/me", "", withCookie(cookie))
	if got := enrollmentUntil(t, rec); got != want {
		t.Fatalf("passkeyEnrollmentUntil = %v, want %s", got, want)
	}
	var mode string
	_ = s.db.QueryRow(`SELECT auth_method FROM sessions`).Scan(&mode)
	if mode != "magic_link" {
		t.Fatalf("auth_method = %q", mode)
	}

	// PATCH returns the same shape, so the client cache keeps the window.
	rec = s.req("PATCH", "/api/me", `{"name":"Ann"}`, withCookie(cookie))
	if got := enrollmentUntil(t, rec); got != want {
		t.Fatalf("PATCH passkeyEnrollmentUntil = %v, want %s", got, want)
	}

	s.clock.Advance(4 * time.Minute)
	if got := enrollmentUntil(t, s.req("GET", "/api/me", "", withCookie(cookie))); got != nil {
		t.Fatalf("after the window: passkeyEnrollmentUntil = %v, want null", got)
	}
}

// --- magic-link-code: one-time codes ---

var codeRE = regexp.MustCompile(`\n    ([A-Z0-9]{3}-[A-Z0-9]{3})\n`)

// requestCode requests an email from a browser and returns the token, the
// code as printed in the mail, and the login-request cookie value.
func (s *server) requestCode(t *testing.T, email string, opts ...func(*http.Request)) (token, code, browser string) {
	t.Helper()
	token, browser = s.requestFromBrowser(t, email, opts...)
	m := codeRE.FindStringSubmatch(s.mail.last(t).Text)
	if m == nil {
		t.Fatalf("no code in mail %q", s.mail.last(t).Text)
	}
	return token, m[1], browser
}

func (s *server) postCode(email, code string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	return s.req("POST", "/api/auth/magic-link/code", `{"email":`+mustJSON(email)+`,"code":`+mustJSON(code)+`}`, opts...)
}

func wantInvalidCode(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_code" || sessionCookie(rec) != nil {
		t.Fatalf("got %d %s, want 400 invalid_code without session", rec.Code, rec.Body)
	}
}

func TestCodeInMail(t *testing.T) {
	s := newServer(t)
	_, code, _ := s.requestCode(t, "ann@example.org")
	for _, r := range strings.ReplaceAll(code, "-", "") {
		if !strings.ContainsRune(auth.CodeAlphabet, r) {
			t.Fatalf("code %q contains %q", code, r)
		}
	}
}

func TestCodeNotStoredInPlainText(t *testing.T) {
	s := newServer(t)
	_, code, _ := s.requestCode(t, "ann@example.org")
	plain := strings.ReplaceAll(code, "-", "")

	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM magic_link_tokens
		WHERE instr(CAST(code_hash AS TEXT), ?) > 0 OR instr(CAST(token_hash AS TEXT), ?) > 0
		   OR email = ? OR locale = ? OR request_ip = ?`,
		plain, plain, plain, plain, plain).Scan(&n)
	if err != nil || n != 0 {
		t.Fatalf("plain code found in %d rows (err %v)", n, err)
	}
	var hashLen int
	if err := s.db.QueryRow(`SELECT length(code_hash) FROM magic_link_tokens`).Scan(&hashLen); err != nil || hashLen != 32 {
		t.Fatalf("code hash length = %d (err %v), want 32", hashLen, err)
	}
}

func TestCodeSignsIn(t *testing.T) {
	s := newServer(t)
	_, code, browser := s.requestCode(t, "ann@example.org")
	// Lower case and without the hyphen: still the same code.
	entered := strings.ToLower(strings.ReplaceAll(code, "-", ""))

	rec := s.postCode(" Ann@Example.org ", entered, withLoginRequest(browser))
	if rec.Code != http.StatusOK || sessionCookie(rec) == nil {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if u := decodeUser(t, rec); u.Email != "ann@example.org" {
		t.Fatalf("user = %+v", u)
	}
	if c := loginRequestCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Fatalf("login-request cookie not cleared: %+v", c)
	}
	var method string
	if err := s.db.QueryRow(`SELECT auth_method FROM sessions`).Scan(&method); err != nil || method != "magic_link" {
		t.Fatalf("auth_method = %q (err %v)", method, err)
	}
}

func TestCodeRejected(t *testing.T) {
	s := newServer(t)
	_, code, browser := s.requestCode(t, "ann@example.org")
	_, _, otherBrowser := s.requestCode(t, "bob@example.org")

	for name, rec := range map[string]func() *httptest.ResponseRecorder{
		"no cookie": func() *httptest.ResponseRecorder { return s.postCode("ann@example.org", code) },
		"other browser": func() *httptest.ResponseRecorder {
			return s.postCode("ann@example.org", code, withLoginRequest(otherBrowser))
		},
		"wrong address": func() *httptest.ResponseRecorder {
			return s.postCode("cid@example.org", code, withLoginRequest(browser))
		},
		"invalid email": func() *httptest.ResponseRecorder { return s.postCode("nope", code, withLoginRequest(browser)) },
	} {
		t.Run(name, func(t *testing.T) { wantInvalidCode(t, rec()) })
	}
	// None of these reached Ann's code in her browser, so it still works there.
	if rec := s.postCode("ann@example.org", code, withLoginRequest(browser)); rec.Code != http.StatusOK {
		t.Fatalf("code no longer works in the requesting browser: %d %s", rec.Code, rec.Body)
	}
}

func TestCodeExpires(t *testing.T) {
	s := newServer(t)
	_, code, browser := s.requestCode(t, "ann@example.org")
	s.clock.Advance(16 * time.Minute)
	wantInvalidCode(t, s.postCode("ann@example.org", code, withLoginRequest(browser)))
}

func TestCodeAndLinkUseEachOtherUp(t *testing.T) {
	t.Run("link first", func(t *testing.T) {
		s := newServer(t)
		token, code, browser := s.requestCode(t, "ann@example.org")
		if rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`"}`, withLoginRequest(browser)); rec.Code != http.StatusOK {
			t.Fatalf("verify: %d %s", rec.Code, rec.Body)
		}
		wantInvalidCode(t, s.postCode("ann@example.org", code, withLoginRequest(browser)))
	})
	t.Run("code first", func(t *testing.T) {
		s := newServer(t)
		token, code, browser := s.requestCode(t, "ann@example.org")
		if rec := s.postCode("ann@example.org", code, withLoginRequest(browser)); rec.Code != http.StatusOK {
			t.Fatalf("code: %d %s", rec.Code, rec.Body)
		}
		rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`","confirm":true}`)
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_token" {
			t.Fatalf("verify after code: %d %s", rec.Code, rec.Body)
		}
	})
}

func TestCodeLockedAfterFiveWrongAttempts(t *testing.T) {
	s := newServer(t)
	token, code, browser := s.requestCode(t, "ann@example.org")
	wrong := "AAAAAA"
	if strings.ReplaceAll(code, "-", "") == wrong {
		wrong = "BBBBBB"
	}
	for range 5 {
		wantInvalidCode(t, s.postCode("ann@example.org", wrong, withLoginRequest(browser)))
	}
	wantInvalidCode(t, s.postCode("ann@example.org", code, withLoginRequest(browser)))

	// The link still works.
	rec := s.req("POST", "/api/auth/magic-link/verify", `{"token":"`+token+`"}`, withLoginRequest(browser))
	if rec.Code != http.StatusOK || sessionCookie(rec) == nil {
		t.Fatalf("link after locked code: %d %s", rec.Code, rec.Body)
	}
}

func TestWrongCodeCountsAgainstEveryLiveCode(t *testing.T) {
	s := newServer(t)
	_, first, browser := s.requestCode(t, "ann@example.org")
	for range 4 {
		wantInvalidCode(t, s.postCode("ann@example.org", "AAAAAA", withLoginRequest(browser)))
	}
	s.clock.Advance(61 * time.Second)
	_, second, _ := s.requestCode(t, "ann@example.org", withLoginRequest(browser))
	// The fifth wrong attempt locks the first code and is the second code's first.
	wantInvalidCode(t, s.postCode("ann@example.org", "AAAAAA", withLoginRequest(browser)))
	wantInvalidCode(t, s.postCode("ann@example.org", first, withLoginRequest(browser)))
	if rec := s.postCode("ann@example.org", second, withLoginRequest(browser)); rec.Code != http.StatusOK {
		t.Fatalf("second code: %d %s", rec.Code, rec.Body)
	}
}

func TestCodeMalformedBody(t *testing.T) {
	s := newServer(t)
	rec := s.req("POST", "/api/auth/magic-link/code", `{"email":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}
