// Package images stores images as files in the data directory: favicons the
// server fetched, shared by all users, and images users uploaded.
package images

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"famodash/internal/db"
)

// MaxSize is the largest image in bytes, for uploads and favicons alike.
const MaxSize = 1 << 20

// Sources of an image.
const (
	SourceFavicon = "favicon"
	SourceUpload  = "upload"
)

// Errors returned by the store.
var (
	ErrNotFound    = errors.New("image not found")
	ErrUnsupported = errors.New("unsupported image type")
	ErrTooLarge    = errors.New("image too large")
)

// Image describes a stored image.
type Image struct {
	ID          string
	Source      string
	ContentType string
	Size        int64
}

// Store keeps image files in Dir and their records in DB.
type Store struct {
	DB  *sql.DB
	Dir string
	Now func() time.Time
}

// NewStore returns a store for <dataDir>/images, creating the directory.
func NewStore(database *sql.DB, dataDir string) (*Store, error) {
	dir := filepath.Join(dataDir, "images")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("creating image directory: %w", err)
	}
	return &Store{DB: database, Dir: dir, Now: time.Now}, nil
}

// icoMagic starts every ICO file (reserved 0, type 1 = icon).
var icoMagic = []byte{0, 0, 1, 0}

// Sniff returns the content type of an allowed image, detected from its
// bytes: PNG, JPEG, GIF, WebP or ICO. SVG and everything else is refused.
func Sniff(data []byte) (string, error) {
	if len(data) > MaxSize {
		return "", ErrTooLarge
	}
	if len(data) == 0 {
		return "", ErrUnsupported
	}
	if bytes.HasPrefix(data, icoMagic) {
		return "image/x-icon", nil
	}
	switch t := http.DetectContentType(data); t {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return t, nil
	}
	return "", ErrUnsupported
}

// Save stores a new image. userID is set for uploads and empty for favicons.
func (s *Store) Save(ctx context.Context, source, userID string, data []byte) (Image, error) {
	contentType, err := Sniff(data)
	if err != nil {
		return Image{}, err
	}
	img := Image{ID: uuid.NewString(), Source: source, ContentType: contentType, Size: int64(len(data))}
	if err := s.writeFile(img.ID, data); err != nil {
		return Image{}, err
	}
	var owner any
	if userID != "" {
		owner = userID
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		img.ID, source, owner, contentType, img.Size, db.FormatTime(s.Now()),
	); err != nil {
		_ = os.Remove(s.path(img.ID))
		return Image{}, err
	}
	return img, nil
}

// Replace writes new bytes for an existing image under the same ID.
func (s *Store) Replace(ctx context.Context, id string, data []byte) error {
	contentType, err := Sniff(data)
	if err != nil {
		return err
	}
	if err := s.writeFile(id, data); err != nil {
		return err
	}
	res, err := s.DB.ExecContext(ctx,
		`UPDATE images SET content_type = ?, size = ? WHERE id = ?`, contentType, len(data), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_ = os.Remove(s.path(id))
		return ErrNotFound
	}
	return nil
}

// Open returns an image's record and its file, which the caller closes.
func (s *Store) Open(ctx context.Context, id string) (Image, *os.File, error) {
	img := Image{ID: id}
	err := s.DB.QueryRowContext(ctx,
		`SELECT source, content_type, size FROM images WHERE id = ?`, id,
	).Scan(&img.Source, &img.ContentType, &img.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return Image{}, nil, ErrNotFound
	}
	if err != nil {
		return Image{}, nil, err
	}
	f, err := os.Open(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return Image{}, nil, ErrNotFound
	}
	if err != nil {
		return Image{}, nil, err
	}
	return img, f, nil
}

// Delete removes an image's record and file.
func (s *Store) Delete(ctx context.Context, id string) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM images WHERE id = ?`, id); err != nil {
		return err
	}
	if err := os.Remove(s.path(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ReadLimited reads r up to MaxSize bytes and fails with ErrTooLarge beyond.
func ReadLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxSize {
		return nil, ErrTooLarge
	}
	return data, nil
}

// path is the file of an image. IDs are UUIDs, so they are safe file names;
// anything else is mapped to a name that can't exist.
func (s *Store) path(id string) string {
	if u, err := uuid.Parse(id); err != nil || u.String() != id {
		return filepath.Join(s.Dir, "invalid")
	}
	return filepath.Join(s.Dir, id)
}

// writeFile writes the bytes to a temporary file and renames it into place,
// so readers never see half a file.
func (s *Store) writeFile(id string, data []byte) error {
	tmp, err := os.CreateTemp(s.Dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path(id))
}
