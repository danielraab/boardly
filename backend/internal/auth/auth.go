// Package auth implements passwordless sign-in by magic link and server-side sessions.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"famodash/internal/aaguid"
	famomail "famodash/internal/mail"
	"famodash/internal/ratelimit"
	"famodash/internal/users"
)

// Service holds the dependencies of sign-in and sessions.
type Service struct {
	DB      *sql.DB
	Users   *users.Store
	Mailer  famomail.Sender
	BaseURL *url.URL
	// IPLimiter limits link requests per client IP.
	IPLimiter *ratelimit.Limiter
	Now       func() time.Time

	// WebAuthn is the relying party for passkeys.
	WebAuthn *webauthn.WebAuthn
	// Providers names passkey providers; nil knows none.
	Providers *aaguid.Directory
}

// NewService returns a service with the real clock and the default per-IP limit (20/h, burst 5).
func NewService(database *sql.DB, userStore *users.Store, mailer famomail.Sender, baseURL *url.URL) *Service {
	return &Service{
		DB:        database,
		Users:     userStore,
		Mailer:    mailer,
		BaseURL:   baseURL,
		IPLimiter: ratelimit.New(20, 5, 10_000),
		Now:       time.Now,
	}
}

// ErrInvalidEmail is returned for an address that isn't a plain, valid email address.
var ErrInvalidEmail = errors.New("invalid email")

// NormalizeEmail trims and lowercases the address and checks that it is a plain
// address without a display name, such as "ann@example.org".
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return "", ErrInvalidEmail
	}
	_, domain, _ := strings.Cut(email, "@")
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// newSecret returns a random 256-bit secret (base64url) and its SHA-256 hash.
func newSecret() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generating secret: %w", err)
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	return s, hash(s), nil
}

func hash(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}
