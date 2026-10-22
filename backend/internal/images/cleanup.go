package images

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"boardly/internal/db"
)

// UnusedUploadAge is how long an upload may stay unused, so one made for a
// draft that isn't saved yet survives.
const UnusedUploadAge = 24 * time.Hour

// Sweep deletes uploads older than UnusedUploadAge that no dashboard widget
// refers to, and then files that have no record. A widget refers to an image
// by its ID under a config key named "imageId", at any depth. Favicons are
// kept. It returns the number of deleted uploads.
func (s *Store) Sweep(ctx context.Context) (int, error) {
	now := s.Now()
	rows, err := s.DB.QueryContext(ctx,
		`DELETE FROM images
		 WHERE source = 'upload' AND created_at < ?
		   AND NOT EXISTS (
		     SELECT 1 FROM dashboard_widgets AS w, json_tree(w.config) AS j
		     WHERE j.key = 'imageId' AND j.value = images.id)
		 RETURNING id`, db.FormatTime(now.Add(-UnusedUploadAge)))
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
		if err := os.Remove(s.path(id)); err != nil && !os.IsNotExist(err) {
			return len(deleted), err
		}
	}
	return len(deleted), s.removeOrphans(ctx, now)
}

// removeOrphans deletes files without a record, such as leftovers of a crash.
// Only files older than an hour are touched, so a Save that has written its
// file but not yet its record isn't affected.
func (s *Store) removeOrphans(ctx context.Context, now time.Time) error {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || now.Sub(info.ModTime()) < time.Hour {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, ".tmp-") {
			var exists bool
			if err := s.DB.QueryRowContext(ctx,
				`SELECT EXISTS (SELECT 1 FROM images WHERE id = ?)`, name).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		if err := os.Remove(filepath.Join(s.Dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
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
