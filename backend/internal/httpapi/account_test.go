package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestDeleteMe(t *testing.T) {
	s := newServer(t)
	phone := s.signIn(t, "ann@example.org")
	tok := s.createToken(t, phone, nil)
	laptop := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	if rec := s.req("DELETE", "/api/me", `{"email":"bob@example.org","locale":"en"}`, withCookie(laptop)); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "email_mismatch" {
		t.Fatalf("wrong email: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("POST", "/api/me/restore", "", withCookie(laptop)); rec.Code != http.StatusConflict || errorCode(t, rec) != "deletion_not_scheduled" {
		t.Fatalf("restore without schedule: %d %s", rec.Code, rec.Body)
	}

	rec := s.req("DELETE", "/api/me", `{"email":"Ann@Example.org","locale":"de"}`, withCookie(laptop))
	if rec.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", rec.Code, rec.Body)
	}
	if u := decodeUser(t, rec); u.DeletionScheduledAt == nil || !u.DeletionScheduledAt.Equal(s.clock.Now().Add(7*24*time.Hour)) {
		t.Fatalf("deletionScheduledAt = %v", u.DeletionScheduledAt)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(laptop)); rec.Code != http.StatusOK || decodeUser(t, rec).DeletionScheduledAt == nil {
		t.Fatalf("current session: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(phone)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("other session: %d, want 401", rec.Code)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Fatalf("other user's session: %d", rec.Code)
	}
	if rec := s.trpcQuery("lists.list", "{}", withBearer(tok.Secret)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token: %d, want 401", rec.Code)
	}
	if rec := s.req("DELETE", "/api/me", `{"email":"ann@example.org"}`, withCookie(laptop)); rec.Code != http.StatusConflict || errorCode(t, rec) != "deletion_scheduled" {
		t.Fatalf("twice: %d %s", rec.Code, rec.Body)
	}

	// Cancelling doesn't need the window.
	s.clock.Advance(time.Hour)
	rec = s.req("POST", "/api/me/restore", "", withCookie(laptop))
	if rec.Code != http.StatusOK || decodeUser(t, rec).DeletionScheduledAt != nil {
		t.Fatalf("restore: %d %s", rec.Code, rec.Body)
	}
	if rec := s.trpcQuery("lists.list", "{}", withBearer(tok.Secret)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token after restore: %d, want 401", rec.Code)
	}

	// The window has closed by now.
	rec = s.req("DELETE", "/api/me", `{"email":"ann@example.org"}`, withCookie(laptop))
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "deletion_window_closed" {
		t.Fatalf("window closed: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/me", "", withCookie(laptop)); decodeUser(t, rec).DeletionScheduledAt != nil {
		t.Fatal("a rejected request scheduled the deletion")
	}
}

func TestDeletionWindowInMe(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	var body me
	rec := s.req("GET", "/api/me", "", withCookie(ann))
	body = decodeJSONBody[me](t, rec)
	if body.DeletionWindowUntil == nil || body.DeletionWindowUntil.Sub(s.clock.Now()) != 3*time.Minute-61*time.Second {
		t.Fatalf("deletionWindowUntil = %v at %v", body.DeletionWindowUntil, s.clock.Now())
	}
	s.clock.Advance(2 * time.Minute)
	if body = decodeJSONBody[me](t, s.req("GET", "/api/me", "", withCookie(ann))); body.DeletionWindowUntil != nil {
		t.Fatalf("after the window: %v", body.DeletionWindowUntil)
	}
}

func TestAccountEndpointsRequireSession(t *testing.T) {
	s := newServer(t)
	cookie := s.signIn(t, "ann@example.org")
	tok := s.createToken(t, cookie, nil)
	for _, opt := range []func(*http.Request){nil, withBearer(tok.Secret)} {
		var opts []func(*http.Request)
		if opt != nil {
			opts = append(opts, opt)
		}
		if rec := s.req("DELETE", "/api/me", `{"email":"ann@example.org"}`, opts...); rec.Code != http.StatusUnauthorized {
			t.Errorf("DELETE /api/me: %d", rec.Code)
		}
		if rec := s.req("POST", "/api/me/restore", "", opts...); rec.Code != http.StatusUnauthorized {
			t.Errorf("POST /api/me/restore: %d", rec.Code)
		}
	}
}

func TestPublicDashboardHiddenWhileDeletionScheduled(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, ann, "ann")
	s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home", "isPublic": true})

	if rec := s.req("DELETE", "/api/me", `{"email":"ann@example.org"}`, withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", rec.Code, rec.Body)
	}
	for name, opts := range map[string][]func(*http.Request){"signed out": nil, "other user": {withCookie(bob)}} {
		if rec := s.req("GET", "/api/u/ann/home", "", opts...); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	if rec := s.req("GET", "/api/u/ann/home", "", withCookie(ann)); rec.Code != http.StatusOK {
		t.Errorf("owner: %d %s", rec.Code, rec.Body)
	}

	if rec := s.req("POST", "/api/me/restore", "", withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("restore: %d", rec.Code)
	}
	if rec := s.req("GET", "/api/u/ann/home", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Errorf("after cancelling: %d %s", rec.Code, rec.Body)
	}
}
