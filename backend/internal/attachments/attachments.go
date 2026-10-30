// Package attachments stores the files users attach to notes, under
// <data dir>/attachments. Which notes refer to an attachment is derived from
// their content by the notes store (table note_attachments); this package
// reads it, marks unused attachments for deletion and deletes them later.
package attachments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"boardly/internal/blobs"
	"boardly/internal/db"
)

// MaxNameLength bounds an attachment's name, in characters after trimming.
const MaxNameLength = 255

// Errors returned by the store.
var (
	ErrNotFound    = errors.New("attachment not found")
	ErrInvalidName = errors.New("invalid attachment name")
	ErrUnsupported = errors.New("unsupported attachment type")
	ErrTooLarge    = errors.New("attachment too large")
	ErrInUse       = errors.New("attachment in use")
)

// Attachment is an attachment as the API returns it.
type Attachment struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	ContentType string     `json:"contentType"`
	Size        int64      `json:"size"`
	URL         string     `json:"url"`
	CreatedAt   time.Time  `json:"createdAt"`
	DeletedAt   *time.Time `json:"deletedAt"`
	DeleteAt    *time.Time `json:"deleteAt"`
	Keep        bool       `json:"keep"`
	NoteIDs     []string   `json:"noteIds"`
}

// Store keeps attachment files in Files and their records in DB.
type Store struct {
	DB    *sql.DB
	Files blobs.Dir
	Now   func() time.Time

	// MaxBytes is the largest attachment, Allowed the allowed content types
	// and DeleteAfter how long a marked attachment is kept.
	MaxBytes    int64
	Allowed     []string
	DeleteAfter time.Duration
}

// NewStore returns a store for <dataDir>/attachments, creating the directory.
func NewStore(database *sql.DB, dataDir string, maxBytes int64, allowed []string, deleteAfterDays int) (*Store, error) {
	dir, err := blobs.Open(filepath.Join(dataDir, "attachments"))
	if err != nil {
		return nil, fmt.Errorf("creating attachment directory: %w", err)
	}
	return &Store{
		DB: database, Files: dir, Now: time.Now,
		MaxBytes: maxBytes, Allowed: allowed, DeleteAfter: time.Duration(deleteAfterDays) * 24 * time.Hour,
	}, nil
}

// URL is the path an attachment is served at.
func URL(id, name string) string {
	return "/api/attachments/" + id + "/" + url.PathEscape(name)
}

// CleanName trims a file name and checks it: 1 to MaxNameLength characters
// without slashes, backslashes or control characters.
func CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength || !utf8.ValidString(name) ||
		strings.ContainsFunc(name, func(r rune) bool { return r == '/' || r == '\\' || unicode.IsControl(r) }) {
		return "", ErrInvalidName
	}
	return name, nil
}

// ReadLimited reads r up to the store's maximum and fails with ErrTooLarge
// beyond, without reading further.
func (s *Store) ReadLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, s.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.MaxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

// Create stores a new attachment of the user.
func (s *Store) Create(ctx context.Context, userID, name string, data []byte) (Attachment, error) {
	name, err := CleanName(name)
	if err != nil {
		return Attachment{}, err
	}
	if int64(len(data)) > s.MaxBytes {
		return Attachment{}, ErrTooLarge
	}
	t, ok := DetectType(name, data)
	if !ok || !slices.Contains(s.Allowed, t.ContentType) {
		return Attachment{}, ErrUnsupported
	}
	now := s.Now().UTC().Truncate(time.Millisecond)
	a := Attachment{
		ID: blobs.NewID(), Name: name, ContentType: t.ContentType, Size: int64(len(data)),
		CreatedAt: now, NoteIDs: []string{},
	}
	a.URL = URL(a.ID, a.Name)
	if err := s.Files.Write(a.ID, data); err != nil {
		return Attachment{}, err
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO attachments (id, user_id, name, content_type, size, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, userID, a.Name, a.ContentType, a.Size, db.FormatTime(now),
	); err != nil {
		_ = s.Files.Remove(a.ID)
		return Attachment{}, err
	}
	return a, nil
}

const selectColumns = `a.id, a.name, a.content_type, a.size, a.created_at, a.deleted_at, a.keep,
	COALESCE((SELECT json_group_array(note_id) FROM note_attachments WHERE attachment_id = a.id), '[]')`

// scan reads a row of selectColumns.
func (s *Store) scan(row interface{ Scan(...any) error }) (Attachment, error) {
	var (
		a                Attachment
		created, noteIDs string
		deleted          sql.NullString
	)
	if err := row.Scan(&a.ID, &a.Name, &a.ContentType, &a.Size, &created, &deleted, &a.Keep, &noteIDs); err != nil {
		return Attachment{}, err
	}
	var err error
	if a.CreatedAt, err = db.ParseTime(created); err != nil {
		return Attachment{}, err
	}
	if deleted.Valid {
		t, err := db.ParseTime(deleted.String)
		if err != nil {
			return Attachment{}, err
		}
		a.DeletedAt = &t
		if !a.Keep {
			at := t.Add(s.DeleteAfter)
			a.DeleteAt = &at
		}
	}
	if err := json.Unmarshal([]byte(noteIDs), &a.NoteIDs); err != nil {
		return Attachment{}, err
	}
	slices.Sort(a.NoteIDs)
	a.URL = URL(a.ID, a.Name)
	return a, nil
}

// List returns the user's attachments, newest first, then by ID.
func (s *Store) List(ctx context.Context, userID string) ([]Attachment, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+selectColumns+` FROM attachments AS a WHERE a.user_id = ? ORDER BY a.created_at DESC, a.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Attachment{}
	for rows.Next() {
		a, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// Get returns one of the user's attachments.
func (s *Store) Get(ctx context.Context, userID, id string) (Attachment, error) {
	a, err := s.scan(s.DB.QueryRowContext(ctx,
		`SELECT `+selectColumns+` FROM attachments AS a WHERE a.id = ? AND a.user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	return a, err
}

// SetKeep sets or removes the keep flag of one of the user's attachments.
// Removing it also removes the deletion mark, so the cleanup can mark the
// attachment again later; setting it keeps the mark as information.
func (s *Store) SetKeep(ctx context.Context, userID, id string, keep bool) (Attachment, error) {
	query := `UPDATE attachments SET keep = 1 WHERE id = ? AND user_id = ?`
	if !keep {
		query = `UPDATE attachments SET keep = 0, deleted_at = NULL WHERE id = ? AND user_id = ?`
	}
	res, err := s.DB.ExecContext(ctx, query, id, userID)
	if err != nil {
		return Attachment{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Attachment{}, ErrNotFound
	}
	return s.Get(ctx, userID, id)
}

// Delete deletes one of the user's attachments, record and file. It fails
// with ErrInUse while a note refers to it.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM attachments WHERE id = ? AND user_id = ?
		   AND NOT EXISTS (SELECT 1 FROM note_attachments WHERE attachment_id = attachments.id)`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.Get(ctx, userID, id); err != nil {
			return err
		}
		return ErrInUse
	}
	return s.Files.Remove(id)
}

// Served is what serving an attachment needs.
type Served struct {
	Name        string
	ContentType string
	Inline      bool
	CreatedAt   time.Time
}

// Open returns any user's attachment and its file, which the caller closes.
// Attachments are public: anyone with the ID may load them.
func (s *Store) Open(ctx context.Context, id string) (Served, *os.File, error) {
	var (
		a       Served
		created string
	)
	err := s.DB.QueryRowContext(ctx,
		`SELECT name, content_type, created_at FROM attachments WHERE id = ?`, id,
	).Scan(&a.Name, &a.ContentType, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Served{}, nil, ErrNotFound
	}
	if err != nil {
		return Served{}, nil, err
	}
	if a.CreatedAt, err = db.ParseTime(created); err != nil {
		return Served{}, nil, err
	}
	t, _ := TypeOf(a.ContentType)
	a.Inline = t.Inline
	f, err := os.Open(s.Files.Path(id))
	if errors.Is(err, os.ErrNotExist) {
		return Served{}, nil, ErrNotFound
	}
	if err != nil {
		return Served{}, nil, err
	}
	return a, f, nil
}
