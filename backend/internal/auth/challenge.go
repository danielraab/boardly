package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"famodash/internal/db"
)

// ChallengeTTL is how long a WebAuthn ceremony may take from options to verify.
const ChallengeTTL = 5 * time.Minute

// Kinds of WebAuthn ceremony.
const (
	challengeLogin    = "login"
	challengeRegister = "register"
)

// ErrInvalidChallenge is returned when a ceremony's challenge is missing, used,
// expired, of another kind or bound to another session.
var ErrInvalidChallenge = errors.New("invalid challenge")

// storeChallenge saves the ceremony state and returns the secret for the
// challenge cookie. Registration challenges carry the user and the session
// (sessionSecret) that started them.
func (s *Service) storeChallenge(ctx context.Context, kind, userID, sessionSecret string, data *webauthn.SessionData) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encoding challenge: %w", err)
	}
	secret, idHash, err := newSecret()
	if err != nil {
		return "", err
	}
	var user, session any
	if userID != "" {
		user = userID
	}
	if sessionSecret != "" {
		session = hash(sessionSecret)
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO webauthn_challenges (id_hash, kind, user_id, session_id, data, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		idHash, kind, user, session, string(raw), db.FormatTime(s.Now().Add(ChallengeTTL)),
	); err != nil {
		return "", fmt.Errorf("storing challenge: %w", err)
	}
	return secret, nil
}

// takeChallenge uses up the challenge for the cookie secret and returns its
// ceremony state. A challenge can only be taken once. For registration
// challenges, sessionSecret must be the session that started it.
func (s *Service) takeChallenge(ctx context.Context, secret, kind, sessionSecret string) (webauthn.SessionData, error) {
	if secret == "" {
		return webauthn.SessionData{}, ErrInvalidChallenge
	}
	var sessionID []byte
	var raw string
	err := s.DB.QueryRowContext(ctx,
		`DELETE FROM webauthn_challenges WHERE id_hash = ? AND kind = ? AND expires_at > ?
		 RETURNING session_id, data`,
		hash(secret), kind, db.FormatTime(s.Now()),
	).Scan(&sessionID, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return webauthn.SessionData{}, ErrInvalidChallenge
	}
	if err != nil {
		return webauthn.SessionData{}, err
	}
	if kind == challengeRegister && (sessionSecret == "" || !bytes.Equal(sessionID, hash(sessionSecret))) {
		return webauthn.SessionData{}, ErrInvalidChallenge
	}
	var data webauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return webauthn.SessionData{}, fmt.Errorf("decoding challenge: %w", err)
	}
	return data, nil
}
