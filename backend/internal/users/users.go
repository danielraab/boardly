// Package users stores user accounts and their profile data.
package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"famodash/internal/db"
)

// User is an account as the API exposes it.
type User struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
}

// ErrNotFound is returned when no user matches.
var ErrNotFound = errors.New("user not found")

// ErrInvalidName is returned for a display name that is too long.
var ErrInvalidName = errors.New("invalid name")

// MaxNameLength is the longest display name, in characters, after trimming.
const MaxNameLength = 100

// Store reads and writes users.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

// EnsureByEmail returns the user with the (already normalized) email,
// creating the account first if none exists. q may be a transaction.
func (s *Store) EnsureByEmail(ctx context.Context, q db.Querier, email string) (User, error) {
	now := db.FormatTime(s.Now())
	if _, err := q.ExecContext(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, NULL, ?, ?)
		 ON CONFLICT (email) DO NOTHING`,
		uuid.NewString(), email, now, now,
	); err != nil {
		return User{}, err
	}
	return scan(q.QueryRowContext(ctx, `SELECT id, email, name FROM users WHERE email = ?`, email))
}

// Get returns the user with the given ID.
func (s *Store) Get(ctx context.Context, id string) (User, error) {
	return scan(s.DB.QueryRowContext(ctx, `SELECT id, email, name FROM users WHERE id = ?`, id))
}

// SetName sets or (with nil) clears the display name and returns the updated user.
// The name is normalized with NormalizeName.
func (s *Store) SetName(ctx context.Context, id string, name *string) (User, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return User{}, err
	}
	return scan(s.DB.QueryRowContext(ctx,
		`UPDATE users SET name = ?, updated_at = ? WHERE id = ? RETURNING id, email, name`,
		name, db.FormatTime(s.Now()), id,
	))
}

// NormalizeName trims the name. Nil, empty and whitespace-only names become nil;
// names longer than MaxNameLength characters return ErrInvalidName.
func NormalizeName(name *string) (*string, error) {
	if name == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxNameLength {
		return nil, ErrInvalidName
	}
	return &trimmed, nil
}

func scan(row *sql.Row) (User, error) {
	var u User
	var name sql.NullString
	if err := row.Scan(&u.ID, &u.Email, &name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	if name.Valid {
		u.Name = &name.String
	}
	return u, nil
}
