package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"famodash/internal/db"
	famomail "famodash/internal/mail"
	"famodash/internal/users"
)

// LinkTTL is how long a sign-in link stays valid.
const LinkTTL = 15 * time.Minute

// Per-email limits on link requests. The per-IP limit is Service.IPLimiter.
const (
	emailMinInterval = 60 * time.Second
	emailPerHour     = 5
)

// supportedLocales are the languages sign-in emails are written in.
var supportedLocales = map[string]bool{"en": true, "de": true}

// RateLimitedError is returned when a link request exceeds a limit.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limited, retry after %v", e.RetryAfter)
}

// ErrInvalidToken is returned for unknown, expired and already used tokens alike.
var ErrInvalidToken = errors.New("invalid token")

// ErrConfirmationRequired is returned for a valid token that was redeemed without
// confirmation from a browser other than the one that requested it. The token stays unused.
var ErrConfirmationRequired = errors.New("confirmation required")

// RequestLink issues a sign-in link for the normalized email and mails it in the
// given locale. It does the same work whether or not an account exists. A failed
// delivery is logged but not returned, so the response reveals nothing.
//
// browserSecret is the requesting browser's login-request cookie value, or empty
// if it has none. The link is bound to that browser: RequestLink returns the
// secret to set as the cookie (the given one, or a new one).
func (s *Service) RequestLink(ctx context.Context, email, locale, clientIP, browserSecret string) (string, error) {
	if !supportedLocales[locale] {
		locale = famomail.DefaultLocale
	}
	now := s.Now()

	if retry, err := s.emailRetryAfter(ctx, email, now); err != nil {
		return "", err
	} else if retry > 0 {
		return "", &RateLimitedError{RetryAfter: retry}
	}
	if ok, retry := s.IPLimiter.Reserve(clientIP, now); !ok {
		return "", &RateLimitedError{RetryAfter: retry}
	}

	if !validBrowserSecret(browserSecret) {
		var err error
		if browserSecret, _, err = newSecret(); err != nil {
			return "", err
		}
	}
	token, tokenHash, err := newSecret()
	if err != nil {
		return "", err
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (token_hash, email, locale, created_at, expires_at, request_ip, browser_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tokenHash, email, locale, db.FormatTime(now), db.FormatTime(now.Add(LinkTTL)), clientIP, hash(browserSecret),
	); err != nil {
		return "", fmt.Errorf("storing token: %w", err)
	}

	msg, err := famomail.RenderMagicLink(locale, email, famomail.MagicLinkData{
		Link:         s.BaseURL.String() + "/login/verify#token=" + token,
		ValidMinutes: int(LinkTTL / time.Minute),
	})
	if err != nil {
		return "", fmt.Errorf("rendering mail: %w", err)
	}
	if err := s.Mailer.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "sending sign-in link failed", "error", err)
	}
	return browserSecret, nil
}

// validBrowserSecret reports whether a login-request cookie value looks like one
// this server issued (base64url of 32 bytes), so it can be reused.
func validBrowserSecret(s string) bool {
	if len(s) != 43 {
		return false
	}
	for _, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// emailRetryAfter returns how long until the next link request for email is allowed (0 = now).
func (s *Service) emailRetryAfter(ctx context.Context, email string, now time.Time) (time.Duration, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT created_at FROM magic_link_tokens WHERE email = ? AND created_at > ? ORDER BY created_at`,
		email, db.FormatTime(now.Add(-time.Hour)),
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var recent []time.Time
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		t, err := db.ParseTime(v)
		if err != nil {
			return 0, err
		}
		recent = append(recent, t)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var retry time.Duration
	if n := len(recent); n > 0 {
		retry = max(retry, recent[n-1].Add(emailMinInterval).Sub(now))
	}
	if n := len(recent); n >= emailPerHour {
		// Allowed again once enough of the requests in the window have aged out.
		retry = max(retry, recent[n-emailPerHour].Add(time.Hour).Sub(now))
	}
	return max(retry, 0), nil
}

// Redeem uses up a valid token, creates the account for its email if needed and
// starts a session. It returns the user and the new session's secret.
// Unknown, expired and used tokens all return ErrInvalidToken.
//
// Without confirmed, the token is only redeemed when browserSecret is the
// login-request cookie of the browser that requested it; otherwise Redeem
// returns ErrConfirmationRequired and leaves the token unused.
func (s *Service) Redeem(ctx context.Context, token, userAgent, browserSecret string, confirmed bool) (users.User, Session, error) {
	now := s.Now()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return users.User{}, Session{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var email string
	err = tx.QueryRowContext(ctx,
		`UPDATE magic_link_tokens SET used_at = ?
		 WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?
		   AND (? OR browser_hash = ?)
		 RETURNING email`,
		db.FormatTime(now), hash(token), db.FormatTime(now), confirmed, hash(browserSecret),
	).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		if !confirmed {
			// Tell a valid token from the browser that didn't request it apart from an invalid one.
			var valid bool
			if err := tx.QueryRowContext(ctx,
				`SELECT EXISTS (SELECT 1 FROM magic_link_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?)`,
				hash(token), db.FormatTime(now),
			).Scan(&valid); err != nil {
				return users.User{}, Session{}, err
			}
			if valid {
				return users.User{}, Session{}, ErrConfirmationRequired
			}
		}
		return users.User{}, Session{}, ErrInvalidToken
	}
	if err != nil {
		return users.User{}, Session{}, err
	}

	user, err := s.Users.EnsureByEmail(ctx, tx, email)
	if err != nil {
		return users.User{}, Session{}, err
	}
	sess, err := s.createSession(ctx, tx, user.ID, MethodMagicLink, userAgent, now)
	if err != nil {
		return users.User{}, Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return users.User{}, Session{}, err
	}
	return user, sess, nil
}
