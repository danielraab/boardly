package auth

import (
	"context"
	"net/url"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/mail"
	"boardly/internal/users"
)

type discard struct{}

func (discard) Send(context.Context, mail.Message) error { return nil }

func TestNormalizeEmail(t *testing.T) {
	valid := map[string]string{
		" Ann@Example.org ":      "ann@example.org",
		"a.b+tag@sub.example.at": "a.b+tag@sub.example.at",
	}
	for in, want := range valid {
		if got, err := NormalizeEmail(in); err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "  ", "not-an-email", "ann@", "@example.org", "ann@localhost", "ann@example.", "Ann <ann@example.org>", "a b@example.org", "a@b@example.org"} {
		if got, err := NormalizeEmail(in); err == nil {
			t.Errorf("NormalizeEmail(%q) = %q, want an error", in, got)
		}
	}
}

func TestCleanupDeletesExpiredRows(t *testing.T) {
	database := dbtest.Open(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store := users.NewStore(database)
	store.Now = clock
	base, _ := url.Parse("http://localhost:3000")
	svc := NewService(database, store, discard{}, base)
	svc.Now = clock
	ctx := context.Background()

	// t0: an old session and an old token.
	if _, err := svc.RequestLink(ctx, "old@example.org", "en", "192.0.2.1", ""); err != nil {
		t.Fatal(err)
	}
	u, err := store.EnsureByEmail(ctx, database, "old@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.createSession(ctx, database, u.ID, MethodMagicLink, "", now); err != nil {
		t.Fatal(err)
	}

	// 31 days later: a fresh session and a fresh token.
	now = now.Add(31 * 24 * time.Hour)
	if _, err := svc.RequestLink(ctx, "new@example.org", "en", "192.0.2.1", ""); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.createSession(ctx, database, u.ID, MethodMagicLink, "", now)
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}

	var sessions, tokens int
	_ = database.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessions)
	_ = database.QueryRow(`SELECT COUNT(*) FROM magic_link_tokens`).Scan(&tokens)
	if sessions != 1 || tokens != 1 {
		t.Fatalf("sessions = %d, tokens = %d; want 1 each", sessions, tokens)
	}
	if _, err := svc.LookupSession(ctx, fresh.Secret); err != nil {
		t.Fatalf("fresh session deleted: %v", err)
	}
	var email string
	_ = database.QueryRow(`SELECT email FROM magic_link_tokens`).Scan(&email)
	if email != "new@example.org" {
		t.Fatalf("kept token for %q", email)
	}
}

func TestCleanupKeepsRecentlyExpiredTokens(t *testing.T) {
	database := dbtest.Open(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	base, _ := url.Parse("http://localhost:3000")
	svc := NewService(database, users.NewStore(database), discard{}, base)
	svc.Now = func() time.Time { return now }
	ctx := context.Background()

	if _, err := svc.RequestLink(ctx, "ann@example.org", "en", "192.0.2.1", ""); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour) // expired, but not for a day yet
	if err := svc.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var tokens int
	_ = database.QueryRow(`SELECT COUNT(*) FROM magic_link_tokens`).Scan(&tokens)
	if tokens != 1 {
		t.Fatalf("tokens = %d, want 1", tokens)
	}
}

type captured struct{ msgs []mail.Message }

func (c *captured) Send(_ context.Context, m mail.Message) error {
	c.msgs = append(c.msgs, m)
	return nil
}

func TestRequestLinkUsesAppTitle(t *testing.T) {
	database := dbtest.Open(t)
	base, _ := url.Parse("http://localhost:3000")
	box := &captured{}
	svc := NewService(database, users.NewStore(database), box, base)
	svc.AppTitle = "Boardly Staging"

	if _, err := svc.RequestLink(context.Background(), "ann@example.org", "en", "192.0.2.1", "secret"); err != nil {
		t.Fatal(err)
	}
	if len(box.msgs) != 1 || box.msgs[0].Subject != "Your Boardly Staging sign-in link" {
		t.Fatalf("sent %+v", box.msgs)
	}
}
