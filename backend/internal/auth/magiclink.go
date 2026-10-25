package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"boardly/internal/db"
	boardlymail "boardly/internal/mail"
	"boardly/internal/users"
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

// ErrInvalidCode is returned for a one-time code that doesn't sign in, whatever the reason.
var ErrInvalidCode = errors.New("invalid code")

// ErrConfirmationRequired is returned for a valid token that was redeemed without
// confirmation from a browser other than the one that requested it. The token stays unused.
var ErrConfirmationRequired = errors.New("confirmation required")

// RequestLink issues a sign-in link and its one-time code for the normalized
// email and mails both in the given locale. It does the same work whether or not an account exists. A failed
// delivery is logged but not returned, so the response reveals nothing.
//
// browserSecret is the requesting browser's login-request cookie value, or empty
// if it has none. The link is bound to that browser: RequestLink returns the
// secret to set as the cookie (the given one, or a new one).
func (s *Service) RequestLink(ctx context.Context, email, locale, clientIP, browserSecret string) (string, error) {
	if !supportedLocales[locale] {
		locale = boardlymail.DefaultLocale
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
	if s.RegistrationClosed {
		if _, err := s.Users.GetByEmail(ctx, s.DB, email); errors.Is(err, users.ErrNotFound) {
			return browserSecret, s.refuseRegistration(ctx, email, locale, clientIP, browserSecret, now)
		} else if err != nil {
			return "", err
		}
	}

	token, tokenHash, err := newSecret()
	if err != nil {
		return "", err
	}
	code, err := newCode()
	if err != nil {
		return "", err
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (token_hash, email, locale, created_at, expires_at, request_ip, browser_hash, code_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tokenHash, email, locale, db.FormatTime(now), db.FormatTime(now.Add(LinkTTL)), clientIP, hash(browserSecret), codeHash(tokenHash, code),
	); err != nil {
		return "", fmt.Errorf("storing token: %w", err)
	}

	msg, err := boardlymail.RenderMagicLink(locale, email, boardlymail.MagicLinkData{
		AppName:      s.AppTitle,
		Link:         s.BaseURL.String() + "/login/verify#token=" + token,
		Code:         FormatCode(code),
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

// refuseRegistration mails the registration-closed email to an address without
// an account. It stores the request as an already used token that nobody
// receives, so the per-email rate limits count it like a sign-in link request.
// A failed delivery is logged but not returned, as in RequestLink.
func (s *Service) refuseRegistration(ctx context.Context, email, locale, clientIP, browserSecret string, now time.Time) error {
	_, tokenHash, err := newSecret()
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (token_hash, email, locale, created_at, expires_at, used_at, request_ip, browser_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tokenHash, email, locale, db.FormatTime(now), db.FormatTime(now.Add(LinkTTL)), db.FormatTime(now), clientIP, hash(browserSecret),
	); err != nil {
		return fmt.Errorf("storing request: %w", err)
	}

	msg, err := boardlymail.RenderRegistrationClosed(locale, email, boardlymail.RegistrationClosedData{AppName: s.AppTitle})
	if err != nil {
		return fmt.Errorf("rendering mail: %w", err)
	}
	slog.InfoContext(ctx, "sign-in refused, registration is closed")
	if err := s.Mailer.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "sending registration-closed mail failed", "error", err)
	}
	return nil
}

// signInUser returns the account for email, creating it unless registration
// is closed. With registration closed, an address without an account returns
// users.ErrNotFound. q may be a transaction.
func (s *Service) signInUser(ctx context.Context, q db.Querier, email string) (users.User, error) {
	if s.RegistrationClosed {
		return s.Users.GetByEmail(ctx, q, email)
	}
	return s.Users.EnsureByEmail(ctx, q, email)
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

	user, err := s.signInUser(ctx, tx, email)
	if errors.Is(err, users.ErrNotFound) {
		return users.User{}, Session{}, ErrInvalidToken
	}
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

// RedeemCode signs in with the one-time code of a sign-in email. The code must
// belong to an unused, unexpired link for email that the browser with
// browserSecret requested, and that link must have fewer than maxCodeAttempts
// wrong codes. A match uses up the link, creates the account if needed and
// starts a session. A wrong code counts against every live code of that
// browser and address. Every failure returns ErrInvalidCode.
func (s *Service) RedeemCode(ctx context.Context, email, code, userAgent, browserSecret string) (users.User, Session, error) {
	code = NormalizeCode(code)
	now := s.Now()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return users.User{}, Session{}, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		`SELECT token_hash, code_hash FROM magic_link_tokens
		 WHERE email = ? AND browser_hash = ? AND used_at IS NULL AND expires_at > ?
		   AND code_hash IS NOT NULL AND code_attempts < ?`,
		email, hash(browserSecret), db.FormatTime(now), maxCodeAttempts,
	)
	if err != nil {
		return users.User{}, Session{}, err
	}
	var live [][]byte
	var match []byte
	for rows.Next() {
		var tokenHash, stored []byte
		if err := rows.Scan(&tokenHash, &stored); err != nil {
			rows.Close()
			return users.User{}, Session{}, err
		}
		live = append(live, tokenHash)
		if subtle.ConstantTimeCompare(codeHash(tokenHash, code), stored) == 1 {
			match = tokenHash
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return users.User{}, Session{}, err
	}

	if match == nil {
		for _, tokenHash := range live {
			if _, err := tx.ExecContext(ctx,
				`UPDATE magic_link_tokens SET code_attempts = code_attempts + 1 WHERE token_hash = ?`, tokenHash,
			); err != nil {
				return users.User{}, Session{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return users.User{}, Session{}, err
		}
		return users.User{}, Session{}, ErrInvalidCode
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE magic_link_tokens SET used_at = ? WHERE token_hash = ?`, db.FormatTime(now), match,
	); err != nil {
		return users.User{}, Session{}, err
	}
	user, err := s.signInUser(ctx, tx, email)
	if errors.Is(err, users.ErrNotFound) {
		return users.User{}, Session{}, ErrInvalidCode
	}
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
