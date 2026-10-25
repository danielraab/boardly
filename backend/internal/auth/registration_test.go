package auth

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/users"
)

var (
	tokenPattern = regexp.MustCompile(`#token=([A-Za-z0-9_-]+)`)
	codePattern  = regexp.MustCompile(`\b([A-Z0-9]{3}-[A-Z0-9]{3})\b`)
)

func newClosedTestService(t *testing.T) (*Service, *captured) {
	t.Helper()
	database := dbtest.Open(t)
	base, _ := url.Parse("http://localhost:3000")
	box := &captured{}
	svc := NewService(database, users.NewStore(database), box, base)
	svc.RegistrationClosed = true
	return svc, box
}

// requestLink requests a sign-in link and returns the token and code from its email.
func requestLink(t *testing.T, svc *Service, box *captured, email, browserSecret string) (token, code string) {
	t.Helper()
	if _, err := svc.RequestLink(context.Background(), email, "en", "192.0.2.1", browserSecret); err != nil {
		t.Fatal(err)
	}
	text := box.msgs[len(box.msgs)-1].Text
	tm, cm := tokenPattern.FindStringSubmatch(text), codePattern.FindStringSubmatch(text)
	if tm == nil || cm == nil {
		t.Fatalf("no link or code in mail:\n%s", text)
	}
	return tm[1], cm[1]
}

const testBrowserSecret = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"

func TestClosedRegistrationRefusesUnknownAddress(t *testing.T) {
	svc, box := newClosedTestService(t)
	ctx := context.Background()

	secret, err := svc.RequestLink(ctx, "new@example.org", "en", "192.0.2.1", testBrowserSecret)
	if err != nil {
		t.Fatal(err)
	}
	if secret != testBrowserSecret {
		t.Fatalf("browser secret = %q, want the given one", secret)
	}
	if len(box.msgs) != 1 {
		t.Fatalf("sent %d mails, want 1", len(box.msgs))
	}
	m := box.msgs[0]
	if m.To != "new@example.org" || m.Subject != "Registration for Boardly is closed" {
		t.Fatalf("sent %+v", m)
	}
	if all := m.Text + m.HTML; strings.Contains(all, "token=") || codePattern.MatchString(all) {
		t.Fatalf("registration-closed mail contains a link or code:\n%s", all)
	}
	if _, err := svc.Users.GetByEmail(ctx, svc.DB, "new@example.org"); !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("account lookup: err = %v, want ErrNotFound", err)
	}
}

func TestClosedRegistrationSignsInKnownAddress(t *testing.T) {
	svc, box := newClosedTestService(t)
	ctx := context.Background()
	ann, err := svc.Users.EnsureByEmail(ctx, svc.DB, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}

	token, _ := requestLink(t, svc, box, "ann@example.org", testBrowserSecret)
	if box.msgs[0].Subject != "Your Boardly sign-in link" {
		t.Fatalf("subject = %q", box.msgs[0].Subject)
	}
	user, _, err := svc.Redeem(ctx, token, "test", testBrowserSecret, false)
	if err != nil || user.ID != ann.ID {
		t.Fatalf("Redeem = %+v, %v", user, err)
	}
}

func TestClosedRegistrationIsRateLimited(t *testing.T) {
	svc, box := newClosedTestService(t)
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	ctx := context.Background()

	if _, err := svc.RequestLink(ctx, "new@example.org", "en", "192.0.2.1", ""); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	_, err := svc.RequestLink(ctx, "new@example.org", "en", "192.0.2.1", "")
	var limited *RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter <= 0 {
		t.Fatalf("second request: err = %v, want RateLimitedError", err)
	}
	if len(box.msgs) != 1 {
		t.Fatalf("sent %d mails, want 1", len(box.msgs))
	}
}

func TestClosedRegistrationRejectsLinkAndCodeIssuedWhileOpen(t *testing.T) {
	svc, box := newClosedTestService(t)
	svc.RegistrationClosed = false
	ctx := context.Background()
	token, _ := requestLink(t, svc, box, "new@example.org", testBrowserSecret)
	svc.Now = func() time.Time { return time.Now().Add(2 * time.Minute) } // past the per-email interval
	_, code := requestLink(t, svc, box, "new@example.org", testBrowserSecret)
	svc.RegistrationClosed = true

	if _, _, err := svc.Redeem(ctx, token, "test", testBrowserSecret, true); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Redeem: err = %v, want ErrInvalidToken", err)
	}
	if _, _, err := svc.RedeemCode(ctx, "new@example.org", code, "test", testBrowserSecret); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("RedeemCode: err = %v, want ErrInvalidCode", err)
	}
	if _, err := svc.Users.GetByEmail(ctx, svc.DB, "new@example.org"); !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("account lookup: err = %v, want ErrNotFound", err)
	}
}

func TestClosedRegistrationRedeemsCodeForKnownAddress(t *testing.T) {
	svc, box := newClosedTestService(t)
	ctx := context.Background()
	if _, err := svc.Users.EnsureByEmail(ctx, svc.DB, "ann@example.org"); err != nil {
		t.Fatal(err)
	}
	_, code := requestLink(t, svc, box, "ann@example.org", testBrowserSecret)
	if _, _, err := svc.RedeemCode(ctx, "ann@example.org", code, "test", testBrowserSecret); err != nil {
		t.Fatalf("RedeemCode: %v", err)
	}
}
