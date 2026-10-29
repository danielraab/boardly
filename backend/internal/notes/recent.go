package notes

import (
	"context"
	"time"

	"boardly/internal/db"
)

// RecentNotes returns at most limit of the user's notes, pinned ones first,
// then newest first by creation time, or by time of last change when
// byUpdated is set, then by ID descending. A non-zero since leaves out notes whose time is before it.
func (s *Store) RecentNotes(ctx context.Context, userID string, byUpdated bool, since time.Time, limit int) ([]Note, error) {
	// The column is one of two fixed names, never from input.
	col := "created_at"
	if byUpdated {
		col = "updated_at"
	}
	after := ""
	if !since.IsZero() {
		after = db.FormatTime(since)
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE user_id = ? AND `+col+` >= ?
		ORDER BY pinned DESC, `+col+` DESC, id DESC LIMIT ?`, userID, after, limit)
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
