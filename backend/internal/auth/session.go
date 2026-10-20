package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"famodash/internal/db"
)

// Session lifetime: it expires after SessionTTL without use, and use extends
// it at most once per touchInterval.
const (
	SessionTTL    = 30 * 24 * time.Hour
	touchInterval = time.Hour
	maxUserAgent  = 512
)

// MethodMagicLink is the auth_method of sessions started by a magic link.
const MethodMagicLink = "magic_link"

// ErrNoSession is returned for unknown and expired sessions.
var ErrNoSession = errors.New("no session")

// Session is a server-side session.
type Session struct {
	// Secret is the cookie value. It is only known when the session was just
	// created; lookups leave it as passed in.
	Secret    string
	UserID    string
	ExpiresAt time.Time
	// Renewed reports that a lookup extended the expiry, so the cookie should be renewed.
	Renewed bool
}

func (s *Service) createSession(ctx context.Context, q db.Querier, userID, method, userAgent string, now time.Time) (Session, error) {
	secret, idHash, err := newSecret()
	if err != nil {
		return Session{}, err
	}
	if len(userAgent) > maxUserAgent {
		userAgent = userAgent[:maxUserAgent]
	}
	expires := now.Add(SessionTTL)
	if _, err := q.ExecContext(ctx,
		`INSERT INTO sessions (id_hash, user_id, auth_method, created_at, last_used_at, expires_at, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		idHash, userID, method, db.FormatTime(now), db.FormatTime(now), db.FormatTime(expires), userAgent,
	); err != nil {
		return Session{}, err
	}
	return Session{Secret: secret, UserID: userID, ExpiresAt: expires}, nil
}

// LookupSession returns the valid session for the cookie secret and extends
// its expiry when the last extension is older than touchInterval.
func (s *Service) LookupSession(ctx context.Context, secret string) (Session, error) {
	if secret == "" {
		return Session{}, ErrNoSession
	}
	now := s.Now()
	idHash := hash(secret)

	var userID, lastUsed, expires string
	err := s.DB.QueryRowContext(ctx,
		`SELECT user_id, last_used_at, expires_at FROM sessions WHERE id_hash = ? AND expires_at > ?`,
		idHash, db.FormatTime(now),
	).Scan(&userID, &lastUsed, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, err
	}
	sess := Session{Secret: secret, UserID: userID}
	if sess.ExpiresAt, err = db.ParseTime(expires); err != nil {
		return Session{}, err
	}
	last, err := db.ParseTime(lastUsed)
	if err != nil {
		return Session{}, err
	}

	if now.Sub(last) >= touchInterval {
		sess.ExpiresAt = now.Add(SessionTTL)
		sess.Renewed = true
		if _, err := s.DB.ExecContext(ctx,
			`UPDATE sessions SET last_used_at = ?, expires_at = ? WHERE id_hash = ?`,
			db.FormatTime(now), db.FormatTime(sess.ExpiresAt), idHash,
		); err != nil {
			return Session{}, err
		}
	}
	return sess, nil
}

// DeleteSession ends the session with the cookie secret. Unknown secrets are ignored.
func (s *Service) DeleteSession(ctx context.Context, secret string) error {
	if secret == "" {
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id_hash = ?`, hash(secret))
	return err
}

// tokenRetention is how long expired sign-in tokens are kept; the per-email
// rate limit counts tokens from the last hour, so they must outlive that.
const tokenRetention = 24 * time.Hour

// Cleanup deletes expired sessions and sign-in tokens that expired more than a day ago.
func (s *Service) Cleanup(ctx context.Context) error {
	now := s.Now()
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, db.FormatTime(now)); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM magic_link_tokens WHERE expires_at <= ?`, db.FormatTime(now.Add(-tokenRetention)))
	return err
}

// RunJanitor calls Cleanup every interval until ctx is done.
func (s *Service) RunJanitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Cleanup(ctx); err != nil && ctx.Err() == nil {
				slog.ErrorContext(ctx, "cleaning up expired sessions failed", "error", err)
			}
		}
	}
}
