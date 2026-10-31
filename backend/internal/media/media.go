// Package media stores each user's media library: for now internet radio
// streams that the web app plays directly from their URL. Audio never passes
// through the server; it only checks a stream's URL when it is saved.
package media

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/notes"
)

// Kinds of media sources.
const KindStream = "stream"

// MaxTitle bounds a source's title, in characters after trimming.
const MaxTitle = 200

// Errors of the store. The URL check's errors are in check.go.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidKind  = errors.New("invalid kind")
	ErrInvalidTitle = errors.New("invalid title")
	ErrInvalidIcon  = errors.New("invalid icon")
	ErrInvalidOrder = errors.New("invalid order")
)

// Source is a media source as the API returns it.
type Source struct {
	ID       string      `json:"id"`
	Kind     string      `json:"kind"`
	Title    string      `json:"title"`
	URL      string      `json:"url"`
	Icon     *notes.Icon `json:"icon"`
	Position int         `json:"position"`
}

// NewSource is the input of Create. LogoURL is fetched as the icon when Icon
// is nil.
type NewSource struct {
	Kind    string
	Title   string
	URL     string
	Icon    *notes.Icon
	LogoURL string
}

// Patch lists the fields to change; nil fields stay as they are, and a nil
// inner icon removes it.
type Patch struct {
	Title *string
	URL   *string
	Icon  **notes.Icon
}

// Store keeps media sources. Check resolves and validates a stream URL
// before it is saved, and FetchLogo stores a logo as an image of the user and
// returns its ID; both are required.
type Store struct {
	DB        *sql.DB
	Now       func() time.Time
	Check     func(ctx context.Context, rawURL string) (string, error)
	FetchLogo func(ctx context.Context, userID, rawURL string) (string, error)
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB, check func(context.Context, string) (string, error), fetchLogo func(context.Context, string, string) (string, error)) *Store {
	return &Store{DB: database, Now: time.Now, Check: check, FetchLogo: fetchLogo}
}

func normalizeTitle(title string) (string, error) {
	t := strings.TrimSpace(title)
	if t == "" || utf8.RuneCountInString(t) > MaxTitle {
		return "", ErrInvalidTitle
	}
	return t, nil
}

func checkIcon(ctx context.Context, q db.Querier, userID string, icon *notes.Icon) (symbol, imageID *string, err error) {
	symbol, imageID, err = notes.CheckIcon(ctx, q, userID, icon)
	if errors.Is(err, notes.ErrInvalidIcon) {
		err = ErrInvalidIcon
	}
	return symbol, imageID, err
}

const sourceColumns = `id, kind, title, url, icon_symbol, icon_image_id, position`

func scanSource(row interface{ Scan(...any) error }) (Source, error) {
	var s Source
	var symbol, imageID *string
	if err := row.Scan(&s.ID, &s.Kind, &s.Title, &s.URL, &symbol, &imageID, &s.Position); err != nil {
		return Source{}, err
	}
	s.Icon = notes.IconOf(symbol, imageID)
	return s, nil
}

func collect(rows *sql.Rows) ([]Source, error) {
	defer rows.Close()
	list := []Source{}
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// List returns the user's sources in their order.
func (s *Store) List(ctx context.Context, userID string) ([]Source, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+sourceColumns+` FROM media_sources WHERE user_id = ? ORDER BY position, id`, userID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

// ByIDs returns the user's sources with the given IDs, in the order of ids;
// IDs that don't exist are left out.
func (s *Store) ByIDs(ctx context.Context, userID string, ids []string) ([]Source, error) {
	if len(ids) == 0 {
		return []Source{}, nil
	}
	args := []any{userID}
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+sourceColumns+` FROM media_sources WHERE user_id = ? AND id IN (?`+strings.Repeat(", ?", len(ids)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	found, err := collect(rows)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Source, len(found))
	for _, src := range found {
		byID[src.ID] = src
	}
	list := []Source{}
	for _, id := range ids {
		if src, ok := byID[id]; ok {
			list = append(list, src)
		}
	}
	return list, nil
}

// Get returns one of the user's sources.
func (s *Store) Get(ctx context.Context, userID, id string) (Source, error) {
	src, err := scanSource(s.DB.QueryRowContext(ctx,
		`SELECT `+sourceColumns+` FROM media_sources WHERE id = ? AND user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	return src, err
}

// Create checks the source's URL and adds it at the end of the user's list.
// A logo that can't be fetched leaves the source without an icon.
func (s *Store) Create(ctx context.Context, userID string, n NewSource) (Source, error) {
	if n.Kind != KindStream {
		return Source{}, ErrInvalidKind
	}
	title, err := normalizeTitle(n.Title)
	if err != nil {
		return Source{}, err
	}
	symbol, imageID, err := checkIcon(ctx, s.DB, userID, n.Icon)
	if err != nil {
		return Source{}, err
	}
	resolved, err := s.Check(ctx, n.URL)
	if err != nil {
		return Source{}, err
	}
	if n.Icon == nil && n.LogoURL != "" {
		if id, err := s.FetchLogo(ctx, userID, n.LogoURL); err != nil {
			slog.DebugContext(ctx, "station logo not fetched", "url", n.LogoURL, "error", err)
		} else {
			imageID = &id
		}
	}
	id := uuid.NewString()
	now := db.FormatTime(s.Now())
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO media_sources (id, user_id, kind, title, url, icon_symbol, icon_image_id, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM media_sources WHERE user_id = ?), ?, ?)`,
		id, userID, n.Kind, title, resolved, symbol, imageID, userID, now, now,
	); err != nil {
		return Source{}, err
	}
	return s.Get(ctx, userID, id)
}

// Update changes the fields in p. A changed URL is checked again.
func (s *Store) Update(ctx context.Context, userID, id string, p Patch) (Source, error) {
	cur, err := s.Get(ctx, userID, id)
	if err != nil {
		return Source{}, err
	}
	sets, args := []string{}, []any{}
	if p.Title != nil {
		title, err := normalizeTitle(*p.Title)
		if err != nil {
			return Source{}, err
		}
		sets, args = append(sets, "title = ?"), append(args, title)
	}
	if p.Icon != nil {
		symbol, imageID, err := checkIcon(ctx, s.DB, userID, *p.Icon)
		if err != nil {
			return Source{}, err
		}
		sets, args = append(sets, "icon_symbol = ?", "icon_image_id = ?"), append(args, symbol, imageID)
	}
	if p.URL != nil && *p.URL != cur.URL {
		resolved, err := s.Check(ctx, *p.URL)
		if err != nil {
			return Source{}, err
		}
		sets, args = append(sets, "url = ?"), append(args, resolved)
	}
	if len(sets) == 0 {
		return cur, nil
	}
	sets, args = append(sets, "updated_at = ?"), append(args, db.FormatTime(s.Now()), id, userID)
	if _, err := s.DB.ExecContext(ctx,
		`UPDATE media_sources SET `+strings.Join(sets, ", ")+` WHERE id = ? AND user_id = ?`, args...); err != nil {
		return Source{}, err
	}
	return s.Get(ctx, userID, id)
}

// Reorder sets the order of the user's sources. ids must be exactly the IDs
// of all of them.
func (s *Store) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_sources WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrInvalidOrder
	}
	seen := make(map[string]bool, len(ids))
	now := db.FormatTime(s.Now())
	for i, id := range ids {
		if seen[id] {
			return ErrInvalidOrder
		}
		seen[id] = true
		res, err := tx.ExecContext(ctx,
			`UPDATE media_sources SET position = ?, updated_at = ? WHERE id = ? AND user_id = ?`, i, now, id, userID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return ErrInvalidOrder
		}
	}
	return tx.Commit()
}

// Delete removes one of the user's sources.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM media_sources WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}
