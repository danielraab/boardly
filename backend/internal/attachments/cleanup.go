package attachments

import (
	"context"
	"log/slog"
	"time"

	"boardly/internal/db"
)

// UnusedAge is how old an attachment must be before the cleanup marks it
// when no note refers to it, so one uploaded for a note that isn't saved yet
// isn't marked.
const UnusedAge = 24 * time.Hour

// Sweep marks attachments older than UnusedAge that no note refers to and
// that aren't kept, deletes those marked longer than DeleteAfter ago that
// still aren't used or kept, and removes files without a record. It returns
// how many were marked and deleted.
func (s *Store) Sweep(ctx context.Context) (marked, deleted int, err error) {
	now := s.Now()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE attachments SET deleted_at = ?
		 WHERE deleted_at IS NULL AND keep = 0 AND created_at <= ?
		   AND NOT EXISTS (SELECT 1 FROM note_attachments WHERE attachment_id = attachments.id)`,
		db.FormatTime(now), db.FormatTime(now.Add(-UnusedAge)))
	if err != nil {
		return 0, 0, err
	}
	n, _ := res.RowsAffected()
	marked = int(n)

	rows, err := tx.QueryContext(ctx,
		`DELETE FROM attachments
		 WHERE keep = 0 AND deleted_at <= ?
		   AND NOT EXISTS (SELECT 1 FROM note_attachments WHERE attachment_id = attachments.id)
		 RETURNING id`, db.FormatTime(now.Add(-s.DeleteAfter)))
	if err != nil {
		return marked, 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return marked, 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return marked, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	for _, id := range ids {
		if err := s.Files.Remove(id); err != nil {
			return marked, len(ids), err
		}
	}
	return marked, len(ids), s.Files.RemoveOrphans(ctx, now, func(ctx context.Context, id string) (bool, error) {
		var exists bool
		err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM attachments WHERE id = ?)`, id).Scan(&exists)
		return exists, err
	})
}

// RunCleanup sweeps now and then every interval until ctx is done.
func (s *Store) RunCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if marked, deleted, err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "cleaning up unused attachments failed", "error", err)
		} else if marked > 0 || deleted > 0 {
			slog.InfoContext(ctx, "cleaned up unused attachments", "marked", marked, "deleted", deleted)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
