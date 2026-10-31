package images

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"boardly/internal/db"
)

// UnusedUploadAge is how long an upload may stay unused, so one made for a
// draft that isn't saved yet survives.
const UnusedUploadAge = 24 * time.Hour

// UnusedPreviewAge is how long a preview or a raw image may stay unused. It
// protects a preview that was just saved and is about to be set on its link,
// and a raw file whose job is about to be queued.
const UnusedPreviewAge = time.Hour

// Sweep deletes uploads older than UnusedUploadAge that no dashboard widget,
// note, note category or media source refers to, previews older than UnusedPreviewAge that no link refers to,
// files that have no record, and raw images that no job names. A widget
// refers to an image by its ID under a config key named "imageId", at any
// depth. Favicons are kept. It returns the number of deleted images.
func (s *Store) Sweep(ctx context.Context) (int, error) {
	now := s.Now()
	uploads, err := s.deleteImages(ctx,
		`DELETE FROM images
		 WHERE source = 'upload' AND created_at < ?
		   AND NOT EXISTS (
		     SELECT 1 FROM dashboard_widgets AS w, json_tree(w.config) AS j
		     WHERE j.key = 'imageId' AND j.value = images.id)
		   AND NOT EXISTS (SELECT 1 FROM notes WHERE icon_image_id = images.id)
		   AND NOT EXISTS (SELECT 1 FROM note_categories WHERE icon_image_id = images.id)
		   AND NOT EXISTS (SELECT 1 FROM media_sources WHERE icon_image_id = images.id)
		 RETURNING id`, db.FormatTime(now.Add(-UnusedUploadAge)))
	if err != nil {
		return uploads, err
	}
	previews, err := s.deleteImages(ctx,
		`DELETE FROM images
		 WHERE source = 'preview' AND created_at < ?
		   AND NOT EXISTS (SELECT 1 FROM favourite_nodes WHERE preview_image_id = images.id)
		 RETURNING id`, db.FormatTime(now.Add(-UnusedPreviewAge)))
	if err != nil {
		return uploads + previews, err
	}
	if err := s.removeOrphans(ctx, now); err != nil {
		return uploads + previews, err
	}
	return uploads + previews, s.removeUnusedRaw(ctx, now)
}

// deleteImages runs a DELETE … RETURNING id and removes the files of the
// deleted records. It returns how many were deleted.
func (s *Store) deleteImages(ctx context.Context, query string, args ...any) (int, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	var deleted []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		deleted = append(deleted, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range deleted {
		if err := s.files().Remove(id); err != nil {
			return len(deleted), err
		}
	}
	return len(deleted), nil
}

// removeUnusedRaw deletes raw images older than UnusedPreviewAge that no
// preview_convert job names, such as leftovers of a crash.
func (s *Store) removeUnusedRaw(ctx context.Context, now time.Time) error {
	dir := filepath.Join(s.Dir, rawDirName)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || now.Sub(info.ModTime()) < UnusedPreviewAge {
			continue
		}
		var used bool
		if err := s.DB.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM jobs WHERE kind = 'preview_convert' AND json_extract(payload, '$.file') = ?)`,
			e.Name()).Scan(&used); err != nil {
			return err
		}
		if used {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// removeOrphans deletes files without a record, such as leftovers of a crash.
func (s *Store) removeOrphans(ctx context.Context, now time.Time) error {
	return s.files().RemoveOrphans(ctx, now, func(ctx context.Context, id string) (bool, error) {
		var exists bool
		err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM images WHERE id = ?)`, id).Scan(&exists)
		return exists, err
	})
}

// RunCleanup sweeps now and then every interval until ctx is done.
func (s *Store) RunCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if n, err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "cleaning up unused images failed", "error", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "deleted unused images", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
