// Package notes stores each user's notes and note categories. A note's
// content is a JSON document of its own format (see doc.go).
package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// MaxNoteTitle bounds a note's title, in characters after trimming.
const MaxNoteTitle = 200

// Errors of the store. ErrInvalidDocument is in doc.go.
var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidTitle    = errors.New("invalid title")
	ErrInvalidIcon     = errors.New("invalid icon")
	ErrInvalidColor    = errors.New("invalid color")
	ErrInvalidCategory = errors.New("invalid category")
	ErrInvalidParent   = errors.New("invalid parent")
	ErrVersionConflict = errors.New("version conflict")
)

// Store keeps notes and note categories.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

// Note is a note as the API exposes it.
type Note struct {
	ID         string    `json:"id"`
	Title      *string   `json:"title"`
	Icon       *Icon     `json:"icon"`
	Color      *string   `json:"color"`
	CategoryID *string   `json:"categoryId"`
	Content    Doc       `json:"content"`
	Pinned     bool      `json:"pinned"`
	Archived   bool      `json:"archived"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// NewNote is the input of CreateNote. A nil Content is an empty document.
type NewNote struct {
	Title      *string
	Icon       *Icon
	Color      *string
	CategoryID *string
	Content    json.RawMessage
	Pinned     bool
	Archived   bool
}

// NotePatch lists the fields to change; nil fields stay as they are, and a
// nil inner pointer clears the field.
type NotePatch struct {
	Title      **string
	Icon       **Icon
	Color      **string
	CategoryID **string
	Content    json.RawMessage
	Pinned     *bool
	Archived   *bool
}

func (p NotePatch) empty() bool {
	return p.Title == nil && p.Icon == nil && p.Color == nil && p.CategoryID == nil && p.Content == nil &&
		p.Pinned == nil && p.Archived == nil
}

// normalizeNoteTitle trims a title; empty ones become nil and too long ones
// return ErrInvalidTitle.
func normalizeNoteTitle(title *string) (*string, error) {
	if title == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*title)
	if utf8.RuneCountInString(t) > MaxNoteTitle {
		return nil, ErrInvalidTitle
	}
	if t == "" {
		return nil, nil
	}
	return &t, nil
}

const noteColumns = `id, title, icon_symbol, icon_image_id, color, category_id, content, pinned, archived, version, created_at, updated_at`

func scanNote(row interface{ Scan(...any) error }) (Note, error) {
	var n Note
	var symbol, imageID *string
	var content, created, updated string
	if err := row.Scan(&n.ID, &n.Title, &symbol, &imageID, &n.Color, &n.CategoryID, &content, &n.Pinned, &n.Archived, &n.Version, &created, &updated); err != nil {
		return Note{}, err
	}
	n.Icon = iconOf(symbol, imageID)
	if err := json.Unmarshal([]byte(content), &n.Content); err != nil {
		return Note{}, err
	}
	var err error
	if n.CreatedAt, err = db.ParseTime(created); err != nil {
		return Note{}, err
	}
	if n.UpdatedAt, err = db.ParseTime(updated); err != nil {
		return Note{}, err
	}
	return n, nil
}

// Notes returns all of the user's notes, last changed first.
func (s *Store) Notes(ctx context.Context, userID string) ([]Note, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE user_id = ? ORDER BY updated_at DESC, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Note returns one of the user's notes, or ErrNotFound.
func (s *Store) Note(ctx context.Context, userID, id string) (Note, error) {
	return note(ctx, s.DB, userID, id)
}

// NotesByID returns those of the user's notes whose ID is among ids, in no
// particular order. Unknown IDs and other users' notes are left out.
func (s *Store) NotesByID(ctx context.Context, userID string, ids []string) ([]Note, error) {
	out := []Note{}
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE user_id = ? AND id IN (?`+strings.Repeat(`, ?`, len(ids)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func note(ctx context.Context, q db.Querier, userID, id string) (Note, error) {
	n, err := scanNote(q.QueryRowContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE id = ? AND user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	return n, err
}

// checkCategory returns ErrInvalidCategory unless id is nil or one of the
// user's categories.
func checkCategory(ctx context.Context, q db.Querier, userID string, id *string) error {
	if id == nil {
		return nil
	}
	if _, err := category(ctx, q, userID, *id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidCategory
		}
		return err
	}
	return nil
}

// parseContent validates a document; nil is an empty one.
func parseContent(raw json.RawMessage) (string, error) {
	d := EmptyDoc()
	if raw != nil {
		var err error
		if d, err = ParseDoc(raw); err != nil {
			return "", err
		}
	}
	b, err := json.Marshal(d)
	return string(b), err
}

// CreateNote adds a note with version 1. Errors are ErrInvalidTitle,
// ErrInvalidIcon, ErrInvalidColor, ErrInvalidCategory and
// ErrInvalidDocument.
func (s *Store) CreateNote(ctx context.Context, userID string, n NewNote) (Note, error) {
	title, err := normalizeNoteTitle(n.Title)
	if err != nil {
		return Note{}, err
	}
	if err := checkColor(n.Color); err != nil {
		return Note{}, err
	}
	content, err := parseContent(n.Content)
	if err != nil {
		return Note{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Note{}, err
	}
	defer func() { _ = tx.Rollback() }()
	symbol, imageID, err := checkIcon(ctx, tx, userID, n.Icon)
	if err != nil {
		return Note{}, err
	}
	if err := checkCategory(ctx, tx, userID, n.CategoryID); err != nil {
		return Note{}, err
	}
	id := uuid.NewString()
	now := db.FormatTime(s.Now())
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, category_id, title, icon_symbol, icon_image_id, color, content, pinned, archived, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		id, userID, n.CategoryID, title, symbol, imageID, n.Color, content, n.Pinned, n.Archived, now, now,
	); err != nil {
		return Note{}, err
	}
	created, err := note(ctx, tx, userID, id)
	if err != nil {
		return Note{}, err
	}
	return created, tx.Commit()
}

// UpdateNote changes a note that is at the given version, raises its version
// by one and sets its time of last change. A note at another version returns
// ErrVersionConflict and stays unchanged. A patch without fields changes
// nothing. Other errors are those of CreateNote and ErrNotFound.
func (s *Store) UpdateNote(ctx context.Context, userID, id string, version int, p NotePatch) (Note, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Note{}, err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := note(ctx, tx, userID, id)
	if err != nil {
		return Note{}, err
	}
	if current.Version != version {
		return Note{}, ErrVersionConflict
	}
	if p.empty() {
		return current, nil
	}
	sets := []string{"version = version + 1", "updated_at = ?"}
	args := []any{db.FormatTime(s.Now())}
	if p.Title != nil {
		title, err := normalizeNoteTitle(*p.Title)
		if err != nil {
			return Note{}, err
		}
		sets, args = append(sets, "title = ?"), append(args, title)
	}
	if p.Icon != nil {
		symbol, imageID, err := checkIcon(ctx, tx, userID, *p.Icon)
		if err != nil {
			return Note{}, err
		}
		sets, args = append(sets, "icon_symbol = ?", "icon_image_id = ?"), append(args, symbol, imageID)
	}
	if p.Color != nil {
		if err := checkColor(*p.Color); err != nil {
			return Note{}, err
		}
		sets, args = append(sets, "color = ?"), append(args, *p.Color)
	}
	if p.CategoryID != nil {
		if err := checkCategory(ctx, tx, userID, *p.CategoryID); err != nil {
			return Note{}, err
		}
		sets, args = append(sets, "category_id = ?"), append(args, *p.CategoryID)
	}
	if p.Content != nil {
		content, err := parseContent(p.Content)
		if err != nil {
			return Note{}, err
		}
		sets, args = append(sets, "content = ?"), append(args, content)
	}
	if p.Pinned != nil {
		sets, args = append(sets, "pinned = ?"), append(args, *p.Pinned)
	}
	if p.Archived != nil {
		sets, args = append(sets, "archived = ?"), append(args, *p.Archived)
	}
	args = append(args, id, userID, version)
	res, err := tx.ExecContext(ctx,
		`UPDATE notes SET `+strings.Join(sets, ", ")+` WHERE id = ? AND user_id = ? AND version = ?`, args...)
	if err != nil {
		return Note{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Note{}, err
	} else if n == 0 {
		return Note{}, ErrVersionConflict
	}
	updated, err := note(ctx, tx, userID, id)
	if err != nil {
		return Note{}, err
	}
	return updated, tx.Commit()
}

// DeleteNote deletes one of the user's notes, or returns ErrNotFound.
func (s *Store) DeleteNote(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM notes WHERE id = ? AND user_id = ?`, id, userID)
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
