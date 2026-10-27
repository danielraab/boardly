package favourites

import (
	"context"
	"time"

	"boardly/internal/db"
)

// RecentLink is a link with the time it was added.
type RecentLink struct {
	Node      *Node
	CreatedAt time.Time
}

// RecentLinks returns at most limit links inside the user's folder folderID
// and its subfolders, newest first by creation time and then by ID. A
// non-zero since leaves out links created before it. Imported links keep the
// creation time of their source. It returns ErrNotFound when the user has no
// folder with that ID.
func (s *Store) RecentLinks(ctx context.Context, userID, folderID string, since time.Time, limit int) ([]RecentLink, error) {
	r, err := load(ctx, s.DB, userID, folderID)
	if err != nil {
		return nil, err
	}
	if r.kind != KindFolder {
		return nil, ErrNotFound
	}
	after := ""
	if !since.IsZero() {
		after = db.FormatTime(since)
	}
	rows, err := s.DB.QueryContext(ctx, subtreeCTE+`
		SELECT n.id, n.title, n.url, n.created_at
		FROM favourite_nodes n JOIN sub ON n.id = sub.id
		WHERE n.kind = 'link' AND n.created_at >= ?
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT ?`, folderID, userID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	links := []RecentLink{}
	for rows.Next() {
		var r row
		var created string
		if err := rows.Scan(&r.id, &r.title, &r.url, &created); err != nil {
			return nil, err
		}
		r.kind = KindLink
		l := RecentLink{Node: r.node()}
		if l.CreatedAt, err = db.ParseTime(created); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
