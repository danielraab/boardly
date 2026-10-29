// Package apptokens stores the API tokens that connected apps (such as
// Karakeep clients) use, each bound to one app, one folder of the owner's
// favourites tree (except Memos tokens) and, for Karakeep and Memos, a part
// of the owner's notes.
package apptokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// App is the kind of client a token is for. It decides which endpoints accept it.
type App string

// Apps.
const (
	AppKarakeep   App = "karakeep"
	AppLinkwarden App = "linkwarden"
	AppMemos      App = "memos"
)

// hasRoot reports whether tokens for app have a root folder. Memos has no
// bookmarks, so its tokens have none.
func (a App) hasRoot() bool { return a != AppMemos }

// hasNotes reports whether tokens for app see notes.
func (a App) hasNotes() bool { return a == AppKarakeep || a == AppMemos }

// NoteScope is which notes a token sees.
type NoteScope string

// Note scopes. Karakeep and Memos tokens see all notes or those of a root
// category with its subcategories; Linkwarden tokens see none.
const (
	NoteScopeAll      NoteScope = "all"
	NoteScopeCategory NoteScope = "category"
	NoteScopeNone     NoteScope = "none"
)

// Status is derived from a token's revocation and expiry dates.
type Status string

// Statuses. Only active tokens authenticate requests.
const (
	StatusActive  Status = "active"
	StatusExpired Status = "expired"
	StatusRevoked Status = "revoked"
)

// SecretPrefix starts every secret, so leaked tokens are easy to recognise.
const SecretPrefix = "bdly_"

// PrefixLength is how many characters of the secret are kept as the token's
// public prefix.
const PrefixLength = 12

// MaxNameLength is the longest token name, in characters, after trimming.
const MaxNameLength = 100

// touchInterval is how often a token's last use is recorded at most.
const touchInterval = time.Minute

// Errors returned by the store.
var (
	ErrNotFound    = errors.New("app token not found")
	ErrInvalidName = errors.New("invalid name")
	ErrInvalidApp  = errors.New("invalid app")
	ErrInvalidRoot = errors.New("root is not a folder of the user")
	// ErrInvalidRootCategory is returned for a root category that isn't a
	// category of the user, or one given for an app without notes.
	ErrInvalidRootCategory = errors.New("root category is not a category of the user")
	ErrInvalidExpiry       = errors.New("expiry is not in the future")
	ErrInactive            = errors.New("app token is not active")
	ErrActive              = errors.New("app token is active")
	// ErrInvalidToken is returned by Authenticate for every rejected secret.
	ErrInvalidToken = errors.New("invalid app token")
)

// Token is an app token as the API exposes it, without its secret.
type Token struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Name      string    `json:"name"`
	App       App       `json:"app"`
	Prefix    string    `json:"prefix"`
	RootID    *string   `json:"rootId"`
	NoteScope NoteScope `json:"noteScope"`
	// RootCategoryID is the root category for NoteScopeCategory, nil when
	// it was deleted, and nil for the other scopes.
	RootCategoryID *string    `json:"rootCategoryId"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastUsedAt     *time.Time `json:"lastUsedAt"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt"`
	Status         Status     `json:"status"`
}

// NewToken is a token to create. A nil ExpiresAt means it never expires.
// RootID must be empty for a Memos token. A nil RootCategoryID gives a
// Karakeep or Memos token all notes; Linkwarden tokens can't have one.
type NewToken struct {
	Name           string
	App            App
	RootID         string
	RootCategoryID *string
	ExpiresAt      *time.Time
}

// Patch lists what to change on a token; nil fields stay as they are. A nil
// *string RootCategoryID gives the token all notes. A Memos token only
// accepts an empty RootID.
type Patch struct {
	RootID         *string
	RootCategoryID **string
}

// Store reads and writes app tokens.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

const columns = `id, user_id, name, app, prefix, root_id, note_scope, root_category_id, created_at, last_used_at, expires_at, revoked_at`

// Create stores a new token for the user and returns it with its secret.
// The secret can't be read again later.
func (s *Store) Create(ctx context.Context, userID string, n NewToken) (Token, string, error) {
	name := strings.TrimSpace(n.Name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength {
		return Token{}, "", ErrInvalidName
	}
	if n.App != AppKarakeep && n.App != AppLinkwarden && n.App != AppMemos {
		return Token{}, "", ErrInvalidApp
	}
	now := s.Now()
	if n.ExpiresAt != nil && !n.ExpiresAt.After(now) {
		return Token{}, "", ErrInvalidExpiry
	}
	if err := checkRoot(ctx, s.DB, userID, n.App, n.RootID); err != nil {
		return Token{}, "", err
	}
	var rootID *string
	if n.App.hasRoot() {
		rootID = &n.RootID
	}
	scope, err := noteScope(ctx, s.DB, userID, n.App, n.RootCategoryID)
	if err != nil {
		return Token{}, "", err
	}

	secret := NewSecret()
	var expires *string
	if n.ExpiresAt != nil {
		v := db.FormatTime(*n.ExpiresAt)
		expires = &v
	}
	id := uuid.NewString()
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO app_tokens (id, user_id, name, app, root_id, note_scope, root_category_id, token_hash, prefix, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, name, n.App, rootID, scope, n.RootCategoryID, Hash(secret), secret[:PrefixLength], db.FormatTime(now), expires,
	); err != nil {
		return Token{}, "", err
	}
	t, err := s.get(ctx, s.DB, userID, id)
	return t, secret, err
}

// checkRoot returns ErrInvalidRoot unless id is a folder of the user, or
// empty for an app without a root folder.
func checkRoot(ctx context.Context, q db.Querier, userID string, app App, id string) error {
	if !app.hasRoot() {
		if id != "" {
			return ErrInvalidRoot
		}
		return nil
	}
	var isFolder bool
	if err := q.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM favourite_nodes WHERE id = ? AND user_id = ? AND kind = 'folder')`,
		id, userID,
	).Scan(&isFolder); err != nil {
		return err
	}
	if !isFolder {
		return ErrInvalidRoot
	}
	return nil
}

// noteScope returns the note scope of a token for app with the root category
// categoryID, or ErrInvalidRootCategory.
func noteScope(ctx context.Context, q db.Querier, userID string, app App, categoryID *string) (NoteScope, error) {
	if !app.hasNotes() {
		if categoryID != nil {
			return "", ErrInvalidRootCategory
		}
		return NoteScopeNone, nil
	}
	if categoryID == nil {
		return NoteScopeAll, nil
	}
	var ok bool
	if err := q.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM note_categories WHERE id = ? AND user_id = ?)`, *categoryID, userID,
	).Scan(&ok); err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidRootCategory
	}
	return NoteScopeCategory, nil
}

// Update changes the root folder and the root category of the user's active
// token and returns it. Errors are ErrNotFound, ErrInactive, ErrInvalidRoot
// and ErrInvalidRootCategory.
func (s *Store) Update(ctx context.Context, userID, id string, p Patch) (Token, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Token{}, err
	}
	defer func() { _ = tx.Rollback() }()
	t, err := s.get(ctx, tx, userID, id)
	if err != nil {
		return Token{}, err
	}
	if t.Status != StatusActive {
		return Token{}, ErrInactive
	}
	if p.RootID != nil {
		if err := checkRoot(ctx, tx, userID, t.App, *p.RootID); err != nil {
			return Token{}, err
		}
	}
	if p.RootID != nil && t.App.hasRoot() {
		if _, err := tx.ExecContext(ctx, `UPDATE app_tokens SET root_id = ? WHERE id = ?`, *p.RootID, id); err != nil {
			return Token{}, err
		}
	}
	if p.RootCategoryID != nil {
		scope, err := noteScope(ctx, tx, userID, t.App, *p.RootCategoryID)
		if err != nil {
			return Token{}, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE app_tokens SET note_scope = ?, root_category_id = ? WHERE id = ?`, scope, *p.RootCategoryID, id,
		); err != nil {
			return Token{}, err
		}
	}
	if t, err = s.get(ctx, tx, userID, id); err != nil {
		return Token{}, err
	}
	return t, tx.Commit()
}

// List returns the user's tokens, newest first.
func (s *Store) List(ctx context.Context, userID string) ([]Token, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+columns+` FROM app_tokens WHERE user_id = ? ORDER BY created_at DESC, rowid DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := s.Now()
	list := []Token{}
	for rows.Next() {
		t, err := scan(rows, now)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// Revoke revokes the user's active token and returns it. Revoking is final.
func (s *Store) Revoke(ctx context.Context, userID, id string) (Token, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Token{}, err
	}
	defer func() { _ = tx.Rollback() }()
	t, err := s.get(ctx, tx, userID, id)
	if err != nil {
		return Token{}, err
	}
	if t.Status != StatusActive {
		return Token{}, ErrInactive
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE app_tokens SET revoked_at = ? WHERE id = ?`, db.FormatTime(s.Now()), id,
	); err != nil {
		return Token{}, err
	}
	if t, err = s.get(ctx, tx, userID, id); err != nil {
		return Token{}, err
	}
	return t, tx.Commit()
}

// Delete removes the user's revoked or expired token.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	t, err := s.get(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	if t.Status == StatusActive {
		return ErrActive
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM app_tokens WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Authenticate returns the active token for app with the given secret and
// records its use at most once per touchInterval. Every rejection, whatever
// the reason, is ErrInvalidToken.
func (s *Store) Authenticate(ctx context.Context, secret string, app App) (Token, error) {
	if !strings.HasPrefix(secret, SecretPrefix) {
		return Token{}, ErrInvalidToken
	}
	now := s.Now()
	t, err := scan(s.DB.QueryRowContext(ctx,
		`SELECT `+columns+` FROM app_tokens WHERE token_hash = ?`, Hash(secret)), now)
	if errors.Is(err, ErrNotFound) {
		return Token{}, ErrInvalidToken
	}
	if err != nil {
		return Token{}, err
	}
	// An active token always has its root and root category: deleting them
	// revokes the token.
	if t.Status != StatusActive || t.App != app || (app.hasRoot() && t.RootID == nil) ||
		(t.NoteScope == NoteScopeCategory && t.RootCategoryID == nil) {
		return Token{}, ErrInvalidToken
	}
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) >= touchInterval {
		if _, err := s.DB.ExecContext(ctx,
			`UPDATE app_tokens SET last_used_at = ? WHERE id = ?`, db.FormatTime(now), t.ID,
		); err != nil {
			return Token{}, err
		}
		used := now.UTC().Truncate(time.Millisecond)
		t.LastUsedAt = &used
	}
	return t, nil
}

// NewSecret returns a new random secret: SecretPrefix and 32 base32
// characters (160 bits).
func NewSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b) // never fails
	return SecretPrefix + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// Hash returns the stored form of a secret.
func Hash(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

func (s *Store) get(ctx context.Context, q db.Querier, userID, id string) (Token, error) {
	return scan(q.QueryRowContext(ctx,
		`SELECT `+columns+` FROM app_tokens WHERE id = ? AND user_id = ?`, id, userID), s.Now())
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner, now time.Time) (Token, error) {
	var t Token
	var rootID, rootCategoryID, lastUsed, expires, revoked sql.NullString
	var created string
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.App, &t.Prefix, &rootID, &t.NoteScope, &rootCategoryID, &created, &lastUsed, &expires, &revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Token{}, ErrNotFound
		}
		return Token{}, err
	}
	var err error
	if t.CreatedAt, err = db.ParseTime(created); err != nil {
		return Token{}, err
	}
	if rootID.Valid {
		t.RootID = &rootID.String
	}
	if rootCategoryID.Valid {
		t.RootCategoryID = &rootCategoryID.String
	}
	for _, f := range []struct {
		src sql.NullString
		dst **time.Time
	}{{lastUsed, &t.LastUsedAt}, {expires, &t.ExpiresAt}, {revoked, &t.RevokedAt}} {
		if !f.src.Valid {
			continue
		}
		v, err := db.ParseTime(f.src.String)
		if err != nil {
			return Token{}, err
		}
		*f.dst = &v
	}
	switch {
	case t.RevokedAt != nil:
		t.Status = StatusRevoked
	case t.ExpiresAt != nil && !now.Before(*t.ExpiresAt):
		t.Status = StatusExpired
	default:
		t.Status = StatusActive
	}
	return t, nil
}
