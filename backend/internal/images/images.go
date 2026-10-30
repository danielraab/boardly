// Package images stores images as files in the data directory: favicons the
// server fetched, shared by all users, images users uploaded, and preview
// images of links.
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

	"boardly/internal/blobs"
	"boardly/internal/db"
)

// MaxSize is the largest image in bytes, for uploads and favicons alike.
// Previews have their own, configurable limit before conversion.
const MaxSize = 1 << 20

// Sources of an image.
const (
	SourceFavicon = "favicon"
	SourceUpload  = "upload"
	SourcePreview = "preview"
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
	dir, err := blobs.Open(filepath.Join(dataDir, "images"))
	if err != nil {
		return nil, fmt.Errorf("creating image directory: %w", err)
	}
	return &Store{DB: database, Dir: string(dir), Now: time.Now}, nil
}

// icoMagic starts every ICO file (reserved 0, type 1 = icon).
var icoMagic = []byte{0, 0, 1, 0}

// Sniff returns the content type of an allowed image of at most MaxSize
// bytes (see SniffLimit).
func Sniff(data []byte) (string, error) {
	return SniffLimit(data, MaxSize)
}

// SniffLimit returns the content type of an allowed image, detected from its
// bytes: PNG, JPEG, GIF, WebP or ICO. SVG and everything else is refused, and
// more than limit bytes fail with ErrTooLarge.
func SniffLimit(data []byte, limit int64) (string, error) {
	if int64(len(data)) > limit {
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

// Save stores a new image. userID is set for uploads and previews and empty
// for favicons. Previews are converted before they are saved, so MaxSize
// doesn't apply to them.
func (s *Store) Save(ctx context.Context, source, userID string, data []byte) (Image, error) {
	limit := int64(MaxSize)
	if source == SourcePreview {
		limit = int64(len(data))
	}
	contentType, err := SniffLimit(data, limit)
	if err != nil {
		return Image{}, err
	}
	img := Image{ID: blobs.NewID(), Source: source, ContentType: contentType, Size: int64(len(data))}
	if err := s.files().Write(img.ID, data); err != nil {
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
		_ = s.files().Remove(img.ID)
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
	if err := s.files().Write(id, data); err != nil {
		return err
	}
	res, err := s.DB.ExecContext(ctx,
		`UPDATE images SET content_type = ?, size = ? WHERE id = ?`, contentType, len(data), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_ = s.files().Remove(id)
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
	f, err := os.Open(s.files().Path(id))
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
	return s.files().Remove(id)
}

// ReadLimited reads r up to MaxSize bytes and fails with ErrTooLarge beyond.
func ReadLimited(r io.Reader) ([]byte, error) {
	return ReadLimit(r, MaxSize)
}

// ReadLimit reads r up to limit bytes and fails with ErrTooLarge beyond.
func ReadLimit(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

// files is the image directory as a blob directory.
func (s *Store) files() blobs.Dir { return blobs.Dir(s.Dir) }
