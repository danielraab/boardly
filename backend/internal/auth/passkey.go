package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"boardly/internal/aaguid"
	"boardly/internal/db"
	"boardly/internal/users"
)

// MaxPasskeyLabelLength is the longest passkey label, in characters, after trimming.
const MaxPasskeyLabelLength = 64

// Passkey errors.
var (
	// ErrPasskeyUnknown is returned when a sign-in uses a credential that isn't registered.
	ErrPasskeyUnknown = errors.New("passkey unknown")
	// ErrPasskeyInvalid is returned for every other failed passkey ceremony.
	ErrPasskeyInvalid = errors.New("passkey invalid")
	// ErrEnrollmentClosed is returned when the session's enrollment window isn't open.
	ErrEnrollmentClosed = errors.New("enrollment window closed")
	// ErrInvalidLabel is returned for a label longer than MaxPasskeyLabelLength.
	ErrInvalidLabel = errors.New("invalid label")
	// ErrPasskeyExists is returned when the credential is already registered.
	ErrPasskeyExists = errors.New("passkey already registered")
	// ErrPasskeyNotFound is returned when the user has no passkey with the ID.
	ErrPasskeyNotFound = errors.New("passkey not found")
)

// Passkey is a registered passkey as the API lists it.
type Passkey struct {
	ID         string           `json:"id"`
	Label      *string          `json:"label"`
	Provider   *aaguid.Provider `json:"provider"`
	CreatedAt  time.Time        `json:"createdAt"`
	LastUsedAt *time.Time       `json:"lastUsedAt"`
}

// passkeyUser adapts a user and their credentials to webauthn.User.
type passkeyUser struct {
	handle      []byte
	user        users.User
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte   { return u.handle }
func (u *passkeyUser) WebAuthnName() string { return u.user.Email }
func (u *passkeyUser) WebAuthnDisplayName() string {
	if u.user.Name != nil {
		return *u.user.Name
	}
	return u.user.Email
}
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// userHandle is the WebAuthn user handle: the 16 bytes of the user's UUID.
func userHandle(userID string) ([]byte, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("user ID %q: %w", userID, err)
	}
	return id[:], nil
}

// credentialColumns are the passkeys columns scanCredential reads.
const credentialColumns = `credential_id, public_key, aaguid, sign_count, transports, backup_eligible, backup_state`

func scanCredential(scan func(...any) error, extra ...any) (webauthn.Credential, error) {
	var c webauthn.Credential
	var transports string
	dest := append([]any{&c.ID, &c.PublicKey, &c.Authenticator.AAGUID, &c.Authenticator.SignCount, &transports,
		&c.Flags.BackupEligible, &c.Flags.BackupState}, extra...)
	if err := scan(dest...); err != nil {
		return c, err
	}
	if err := json.Unmarshal([]byte(transports), &c.Transport); err != nil {
		return c, fmt.Errorf("passkey transports: %w", err)
	}
	c.Flags.UserPresent = true
	return c, nil
}

func (s *Service) passkeyUser(ctx context.Context, user users.User) (*passkeyUser, error) {
	handle, err := userHandle(user.ID)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+credentialColumns+` FROM passkeys WHERE user_id = ?`, user.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	u := &passkeyUser{handle: handle, user: user}
	for rows.Next() {
		c, err := scanCredential(rows.Scan)
		if err != nil {
			return nil, err
		}
		u.credentials = append(u.credentials, c)
	}
	return u, rows.Err()
}

// PasskeyLoginOptions starts a discoverable sign-in. It returns the options for
// the browser and the secret for the challenge cookie.
func (s *Service) PasskeyLoginOptions(ctx context.Context) (*protocol.PublicKeyCredentialRequestOptions, string, error) {
	assertion, data, err := s.WebAuthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return nil, "", fmt.Errorf("starting passkey sign-in: %w", err)
	}
	secret, err := s.storeChallenge(ctx, challengeLogin, "", "", data)
	if err != nil {
		return nil, "", err
	}
	return &assertion.Response, secret, nil
}

// PasskeyLogin verifies a discoverable sign-in for the challenge cookie secret
// and starts a passkey session, which never gets an enrollment window.
func (s *Service) PasskeyLogin(ctx context.Context, challengeSecret string, credential []byte, userAgent string) (users.User, Session, error) {
	data, err := s.takeChallenge(ctx, challengeSecret, challengeLogin, "")
	if errors.Is(err, ErrInvalidChallenge) {
		return users.User{}, Session{}, ErrPasskeyInvalid
	}
	if err != nil {
		return users.User{}, Session{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credential)
	if err != nil {
		return users.User{}, Session{}, ErrPasskeyInvalid
	}

	var unknown bool
	var owner *passkeyUser
	var lookupErr error
	handler := func(rawID, handle []byte) (webauthn.User, error) {
		var userID string
		c, err := scanCredential(
			s.DB.QueryRowContext(ctx, `SELECT `+credentialColumns+`, user_id FROM passkeys WHERE credential_id = ?`, rawID).Scan,
			&userID,
		)
		if errors.Is(err, sql.ErrNoRows) {
			unknown = true
			return nil, ErrPasskeyUnknown
		}
		if err != nil {
			lookupErr = err
			return nil, err
		}
		user, err := s.Users.Get(ctx, userID)
		if err != nil {
			lookupErr = err
			return nil, err
		}
		h, err := userHandle(userID)
		if err != nil {
			lookupErr = err
			return nil, err
		}
		owner = &passkeyUser{handle: h, user: user, credentials: []webauthn.Credential{c}}
		return owner, nil
	}
	_, cred, err := s.WebAuthn.ValidatePasskeyLogin(handler, data, parsed)
	switch {
	case lookupErr != nil:
		return users.User{}, Session{}, lookupErr
	case unknown:
		return users.User{}, Session{}, ErrPasskeyUnknown
	case err != nil:
		slog.InfoContext(ctx, "passkey sign-in rejected", "error", err)
		return users.User{}, Session{}, ErrPasskeyInvalid
	}
	if cred.Authenticator.CloneWarning {
		// Synced passkeys report a sign count of 0, so a regression is only logged.
		slog.WarnContext(ctx, "passkey sign count went backwards", "user_id", owner.user.ID)
	}

	now := s.Now()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return users.User{}, Session{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`UPDATE passkeys SET sign_count = ?, backup_state = ?, last_used_at = ? WHERE credential_id = ?`,
		cred.Authenticator.SignCount, cred.Flags.BackupState, db.FormatTime(now), cred.ID,
	); err != nil {
		return users.User{}, Session{}, err
	}
	sess, err := s.createSession(ctx, tx, owner.user.ID, MethodPasskey, userAgent, now)
	if err != nil {
		return users.User{}, Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return users.User{}, Session{}, err
	}
	return owner.user, sess, nil
}

// PasskeyRegistrationOptions starts adding a passkey for user from sess, which
// must have an open enrollment window. It returns the options for the browser
// and the secret for the challenge cookie.
func (s *Service) PasskeyRegistrationOptions(ctx context.Context, user users.User, sess Session) (*protocol.PublicKeyCredentialCreationOptions, string, error) {
	if !sess.EnrollmentOpen(s.Now()) {
		return nil, "", ErrEnrollmentClosed
	}
	u, err := s.passkeyUser(ctx, user)
	if err != nil {
		return nil, "", err
	}
	exclude := make([]protocol.CredentialDescriptor, 0, len(u.credentials))
	for _, c := range u.credentials {
		exclude = append(exclude, c.Descriptor())
	}
	creation, data, err := s.WebAuthn.BeginRegistration(u,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(exclude),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil {
		return nil, "", fmt.Errorf("starting passkey registration: %w", err)
	}
	secret, err := s.storeChallenge(ctx, challengeRegister, user.ID, sess.Secret, data)
	if err != nil {
		return nil, "", err
	}
	return &creation.Response, secret, nil
}

// NormalizePasskeyLabel trims the label. Nil and blank labels become nil;
// labels longer than MaxPasskeyLabelLength characters return ErrInvalidLabel.
func NormalizePasskeyLabel(label *string) (*string, error) {
	if label == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*label)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxPasskeyLabelLength {
		return nil, ErrInvalidLabel
	}
	return &trimmed, nil
}

// RegisterPasskey completes adding a passkey. The challenge must have been
// started by sess; the enrollment window isn't checked again, so a ceremony
// that started in time can finish after the window closed.
func (s *Service) RegisterPasskey(ctx context.Context, user users.User, sess Session, challengeSecret string, credential []byte, label *string) (Passkey, error) {
	label, err := NormalizePasskeyLabel(label)
	if err != nil {
		return Passkey{}, err
	}
	data, err := s.takeChallenge(ctx, challengeSecret, challengeRegister, sess.Secret)
	if err != nil {
		return Passkey{}, err
	}
	u, err := s.passkeyUser(ctx, user)
	if err != nil {
		return Passkey{}, err
	}
	if !bytes.Equal(data.UserID, u.handle) {
		return Passkey{}, ErrInvalidChallenge
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(credential)
	if err != nil {
		return Passkey{}, ErrPasskeyInvalid
	}
	cred, err := s.WebAuthn.CreateCredential(u, data, parsed)
	if err != nil {
		slog.InfoContext(ctx, "passkey registration rejected", "error", err)
		return Passkey{}, ErrPasskeyInvalid
	}

	transports, err := json.Marshal(cred.Transport)
	if err != nil {
		return Passkey{}, err
	}
	if cred.Transport == nil {
		transports = []byte("[]")
	}
	aaguidBytes := cred.Authenticator.AAGUID
	if len(aaguidBytes) != 16 {
		aaguidBytes = make([]byte, 16)
	}
	now := s.Now()
	p := Passkey{ID: uuid.NewString(), Label: label, Provider: s.Providers.Lookup(aaguidBytes), CreatedAt: now.UTC().Truncate(time.Millisecond)}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO passkeys (id, user_id, credential_id, public_key, aaguid, sign_count, transports, backup_eligible, backup_state, label, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (credential_id) DO NOTHING`,
		p.ID, user.ID, cred.ID, cred.PublicKey, aaguidBytes, cred.Authenticator.SignCount, string(transports),
		cred.Flags.BackupEligible, cred.Flags.BackupState, label, db.FormatTime(now),
	)
	if err != nil {
		return Passkey{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Passkey{}, err
	} else if n == 0 {
		return Passkey{}, ErrPasskeyExists
	}
	return p, nil
}

// ListPasskeys returns the user's passkeys, newest first.
func (s *Service) ListPasskeys(ctx context.Context, userID string) ([]Passkey, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, label, aaguid, created_at, last_used_at FROM passkeys
		 WHERE user_id = ? ORDER BY created_at DESC, rowid DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Passkey{}
	for rows.Next() {
		var p Passkey
		var aaguidBytes []byte
		var created string
		var lastUsed sql.NullString
		if err := rows.Scan(&p.ID, &p.Label, &aaguidBytes, &created, &lastUsed); err != nil {
			return nil, err
		}
		if p.CreatedAt, err = db.ParseTime(created); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			t, err := db.ParseTime(lastUsed.String)
			if err != nil {
				return nil, err
			}
			p.LastUsedAt = &t
		}
		p.Provider = s.Providers.Lookup(aaguidBytes)
		list = append(list, p)
	}
	return list, rows.Err()
}

// DeletePasskey deletes the user's passkey with the ID. Passkeys of other
// users and unknown IDs return ErrPasskeyNotFound.
func (s *Service) DeletePasskey(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM passkeys WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}
