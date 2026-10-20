package auth

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"famodash/internal/dbtest"
	"famodash/internal/users"
)

func newTestService(t *testing.T) (*Service, *time.Time) {
	t.Helper()
	database := dbtest.Open(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store := users.NewStore(database)
	store.Now = clock
	base, _ := url.Parse("http://localhost:3000")
	svc := NewService(database, store, discard{}, base)
	svc.Now = clock
	return svc, &now
}

func TestChallengeSingleUse(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	secret, err := svc.storeChallenge(ctx, challengeLogin, "", "", &webauthn.SessionData{Challenge: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := svc.takeChallenge(ctx, secret, challengeLogin, "")
	if err != nil || data.Challenge != "abc" {
		t.Fatalf("take = %+v, %v", data, err)
	}
	if _, err := svc.takeChallenge(ctx, secret, challengeLogin, ""); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("reuse: err = %v, want ErrInvalidChallenge", err)
	}
	if _, err := svc.takeChallenge(ctx, "", challengeLogin, ""); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("no cookie: err = %v", err)
	}
}

func TestChallengeExpires(t *testing.T) {
	svc, now := newTestService(t)
	ctx := context.Background()

	secret, err := svc.storeChallenge(ctx, challengeLogin, "", "", &webauthn.SessionData{Challenge: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(ChallengeTTL)
	if _, err := svc.takeChallenge(ctx, secret, challengeLogin, ""); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("expired: err = %v", err)
	}
	if err := svc.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = svc.DB.QueryRow(`SELECT COUNT(*) FROM webauthn_challenges`).Scan(&n)
	if n != 0 {
		t.Fatalf("challenges after cleanup = %d", n)
	}
}

func TestChallengeWrongKind(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	secret, err := svc.storeChallenge(ctx, challengeLogin, "", "", &webauthn.SessionData{Challenge: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.takeChallenge(ctx, secret, challengeRegister, "session"); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("wrong kind: err = %v", err)
	}
	// The mismatched attempt doesn't use up the login challenge.
	if _, err := svc.takeChallenge(ctx, secret, challengeLogin, ""); err != nil {
		t.Fatalf("login after wrong kind: %v", err)
	}
}

func TestChallengeBoundToSession(t *testing.T) {
	svc, now := newTestService(t)
	ctx := context.Background()
	u, err := svc.Users.EnsureByEmail(ctx, svc.DB, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	mine, _ := svc.createSession(ctx, svc.DB, u.ID, MethodMagicLink, "", *now)
	other, _ := svc.createSession(ctx, svc.DB, u.ID, MethodMagicLink, "", *now)

	secret, err := svc.storeChallenge(ctx, challengeRegister, u.ID, mine.Secret, &webauthn.SessionData{Challenge: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.takeChallenge(ctx, secret, challengeRegister, other.Secret); !errors.Is(err, ErrInvalidChallenge) {
		t.Fatalf("other session: err = %v", err)
	}

	secret, _ = svc.storeChallenge(ctx, challengeRegister, u.ID, mine.Secret, &webauthn.SessionData{Challenge: "def"})
	if data, err := svc.takeChallenge(ctx, secret, challengeRegister, mine.Secret); err != nil || data.Challenge != "def" {
		t.Fatalf("own session: %+v, %v", data, err)
	}
}

func TestSessionEnrollmentWindow(t *testing.T) {
	svc, now := newTestService(t)
	ctx := context.Background()
	u, _ := svc.Users.EnsureByEmail(ctx, svc.DB, "ann@example.org")
	start := *now

	link, _ := svc.createSession(ctx, svc.DB, u.ID, MethodMagicLink, "", start)
	passkey, _ := svc.createSession(ctx, svc.DB, u.ID, MethodPasskey, "", start)

	got, err := svc.LookupSession(ctx, link.Secret)
	if err != nil || got.Method != MethodMagicLink || got.EnrollmentUntil == nil || !got.EnrollmentUntil.Equal(start.Add(EnrollmentWindow)) {
		t.Fatalf("magic-link session = %+v, %v", got, err)
	}
	if !got.EnrollmentOpen(start.Add(EnrollmentWindow-time.Second)) || got.EnrollmentOpen(start.Add(EnrollmentWindow)) {
		t.Fatal("window must be open for exactly 5 minutes")
	}

	got, err = svc.LookupSession(ctx, passkey.Secret)
	if err != nil || got.Method != MethodPasskey || got.EnrollmentUntil != nil || got.EnrollmentOpen(start) {
		t.Fatalf("passkey session = %+v, %v", got, err)
	}
}
