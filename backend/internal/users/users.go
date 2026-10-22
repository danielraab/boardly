// Package users stores user accounts and their profile data.
package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"famodash/internal/db"
	"famodash/internal/slugs"
)

// User is an account as the API exposes it.
type User struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
	Slug  string  `json:"slug"`
	// CreatedAt and UpdatedAt are for connected apps; the web API leaves them out.
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// ErrNotFound is returned when no user matches.
var ErrNotFound = errors.New("user not found")

// ErrInvalidName is returned for a display name that is too long.
var ErrInvalidName = errors.New("invalid name")

// ErrInvalidSlug, ErrSlugReserved and ErrSlugTaken are returned by SetSlug.
var (
	ErrInvalidSlug  = errors.New("invalid slug")
	ErrSlugReserved = errors.New("slug is reserved")
	ErrSlugTaken    = errors.New("slug is taken")
)

// MaxNameLength is the longest display name, in characters, after trimming.
const MaxNameLength = 100

// Store reads and writes users.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
	// GenerateSlug returns a random slug for new accounts.
	GenerateSlug func() string
}

// NewStore returns a store that uses the real clock and random slugs.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now, GenerateSlug: slugs.Generate}
}

const columns = `id, email, name, slug, created_at, updated_at`

// EnsureByEmail returns the user with the (already normalized) email,
// creating the account first if none exists. q may be a transaction.
// A new account gets a generated slug; if it's taken, "-2", "-3", … is
// appended until it is free.
func (s *Store) EnsureByEmail(ctx context.Context, q db.Querier, email string) (User, error) {
	u, err := scan(q.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE email = ?`, email))
	if !errors.Is(err, ErrNotFound) {
		return u, err
	}
	slug, err := s.freeSlug(ctx, q)
	if err != nil {
		return User{}, err
	}
	now := db.FormatTime(s.Now())
	if _, err := q.ExecContext(ctx,
		`INSERT INTO users (id, email, name, slug, created_at, updated_at) VALUES (?, ?, NULL, ?, ?, ?)
		 ON CONFLICT (email) DO NOTHING`,
		uuid.NewString(), email, slug, now, now,
	); err != nil {
		return User{}, err
	}
	return scan(q.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE email = ?`, email))
}

// freeSlug generates a slug that is neither reserved nor used by another user.
func (s *Store) freeSlug(ctx context.Context, q db.Querier) (string, error) {
	base := s.GenerateSlug()
	for slugs.IsReserved(base) {
		base = s.GenerateSlug()
	}
	slug := base
	for n := 2; ; n++ {
		var taken bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE slug = ?)`, slug).Scan(&taken); err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
}

// Get returns the user with the given ID.
func (s *Store) Get(ctx context.Context, id string) (User, error) {
	return scan(s.DB.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE id = ?`, id))
}

// GetBySlug returns the user with the given slug, ignoring case.
func (s *Store) GetBySlug(ctx context.Context, slug string) (User, error) {
	return scan(s.DB.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE slug = ?`, strings.ToLower(slug)))
}

// SetName sets or (with nil) clears the display name and returns the updated user.
// The name is normalized with NormalizeName.
func (s *Store) SetName(ctx context.Context, id string, name *string) (User, error) {
	return s.Update(ctx, id, Patch{Name: &name})
}

// SetSlug changes the user's slug and returns the updated user.
func (s *Store) SetSlug(ctx context.Context, id string, slug string) (User, error) {
	return s.Update(ctx, id, Patch{Slug: &slug})
}

// Patch lists the profile fields to change; nil fields stay as they are.
type Patch struct {
	Name **string // a nil *string clears the name
	Slug *string
}

// Update validates every field of p before writing any of them, then writes
// them in one statement. Errors are ErrInvalidName, ErrInvalidSlug,
// ErrSlugReserved, ErrSlugTaken and ErrNotFound.
func (s *Store) Update(ctx context.Context, id string, p Patch) (User, error) {
	sets := []string{"updated_at = ?"}
	args := []any{db.FormatTime(s.Now())}
	if p.Name != nil {
		name, err := NormalizeName(*p.Name)
		if err != nil {
			return User{}, err
		}
		sets = append(sets, "name = ?")
		args = append(args, name)
	}
	if p.Slug != nil {
		slug, err := slugs.Normalize(*p.Slug)
		if err != nil {
			return User{}, ErrInvalidSlug
		}
		if slugs.IsReserved(slug) {
			return User{}, ErrSlugReserved
		}
		sets = append(sets, "slug = ?")
		args = append(args, slug)
	}
	args = append(args, id)
	u, err := scan(s.DB.QueryRowContext(ctx,
		`UPDATE users SET `+strings.Join(sets, ", ")+` WHERE id = ? RETURNING `+columns, args...))
	if db.IsUniqueViolation(err) {
		return User{}, ErrSlugTaken
	}
	return u, err
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
	var name, slug sql.NullString
	var created, updated string
	if err := row.Scan(&u.ID, &u.Email, &name, &slug, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	var err error
	if u.CreatedAt, err = db.ParseTime(created); err != nil {
		return User{}, err
	}
	if u.UpdatedAt, err = db.ParseTime(updated); err != nil {
		return User{}, err
	}
	if name.Valid {
		u.Name = &name.String
	}
	u.Slug = slug.String
	return u, nil
}
