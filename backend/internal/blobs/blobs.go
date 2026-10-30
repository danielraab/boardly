// Package blobs keeps files in a directory under random, unguessable IDs:
// the file storage shared by images and attachments. Records of the files
// live in the database of their owners; this package only handles the files.
package blobs

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrphanAge is how old a file without a record must be before RemoveOrphans
// deletes it, so a save that has written its file but not yet its record
// isn't affected.
const OrphanAge = time.Hour

// tmpPrefix starts the names of files that are still being written.
const tmpPrefix = ".tmp-"

// Dir is a directory of files named by their ID.
type Dir string

// Open returns the directory, creating it if needed.
func Open(dir string) (Dir, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return Dir(dir), nil
}

// NewID returns a new random ID.
func NewID() string {
	return uuid.NewString()
}

// Path is the file of an ID. IDs are UUIDs, so they are safe file names;
// anything else is mapped to a name that can't exist.
func (d Dir) Path(id string) string {
	if u, err := uuid.Parse(id); err != nil || u.String() != id {
		return filepath.Join(string(d), "invalid")
	}
	return filepath.Join(string(d), id)
}

// Write writes the bytes to a temporary file and renames it into place, so
// readers never see half a file.
func (d Dir) Write(id string, data []byte) error {
	tmp, err := os.CreateTemp(string(d), tmpPrefix+"*")
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
	return os.Rename(tmp.Name(), d.Path(id))
}

// Remove deletes the file of an ID. A missing file is no error.
func (d Dir) Remove(id string) error {
	if err := os.Remove(d.Path(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RemoveOrphans deletes files older than OrphanAge for which exists reports
// no record, and leftover temporary files. Subdirectories are left alone.
func (d Dir) RemoveOrphans(ctx context.Context, now time.Time, exists func(ctx context.Context, id string) (bool, error)) error {
	entries, err := os.ReadDir(string(d))
	if err != nil {
		return err
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || now.Sub(info.ModTime()) < OrphanAge {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, tmpPrefix) {
			ok, err := exists(ctx, name)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
		}
		if err := os.Remove(filepath.Join(string(d), name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// SetHeaders sets the response headers every stored file is served with:
// its content type, no sniffing, and a sandbox so that a file opened
// directly can't run scripts on the app's origin.
func SetHeaders(h http.Header, contentType string) {
	h.Set("Content-Type", contentType)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
}
