package favourites

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"boardly/internal/db"
)

// FolderLink is a link of a folder with the time it was added and the IDs of
// its tags.
type FolderLink struct {
	Node      *Node
	CreatedAt time.Time
	TagIDs    []string
}

// FolderLinks returns the links directly inside the user's folder folderID in
// their order, and with subfolders also those of every subfolder at the place
// of their subfolder, depth first. It returns ErrNotFound when the user has no
// folder with that ID.
func (s *Store) FolderLinks(ctx context.Context, userID, folderID string, subfolders bool) ([]FolderLink, error) {
	r, err := load(ctx, s.DB, userID, folderID)
	if err != nil {
		return nil, err
	}
	if r.kind != KindFolder {
		return nil, ErrNotFound
	}
	rows, err := s.DB.QueryContext(ctx, subtreeCTE+`
		SELECT n.id, n.parent_id, n.kind, n.title, n.url, n.created_at
		FROM favourite_nodes n JOIN sub ON n.id = sub.id
		WHERE n.id != ?
		ORDER BY n.parent_id, n.position`, folderID, userID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// entry is a child of a folder: a subfolder's ID or a link.
	type entry struct {
		folderID string
		link     *FolderLink
	}
	children := map[string][]entry{}
	links := map[string]*FolderLink{}
	for rows.Next() {
		var r row
		var created string
		if err := rows.Scan(&r.id, &r.parentID, &r.kind, &r.title, &r.url, &created); err != nil {
			return nil, err
		}
		e := entry{folderID: r.id}
		if r.kind == KindLink {
			e = entry{link: &FolderLink{Node: r.node()}}
			if e.link.CreatedAt, err = db.ParseTime(created); err != nil {
				return nil, err
			}
			links[r.id] = e.link
		}
		children[r.parentID.String] = append(children[r.parentID.String], e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagRows, err := s.DB.QueryContext(ctx, subtreeCTE+`
		SELECT nt.node_id, nt.tag_id FROM favourite_node_tags nt JOIN sub ON nt.node_id = sub.id
		ORDER BY nt.node_id, nt.tag_id`, folderID, userID)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var nodeID, tagID string
		if err := tagRows.Scan(&nodeID, &tagID); err != nil {
			return nil, err
		}
		if l := links[nodeID]; l != nil {
			l.TagIDs = append(l.TagIDs, tagID)
		}
	}
	if err := tagRows.Err(); err != nil {
		return nil, err
	}

	out := []FolderLink{}
	var walk func(parentID string)
	walk = func(parentID string) {
		for _, e := range children[parentID] {
			switch {
			case e.link != nil:
				out = append(out, *e.link)
			case subfolders:
				walk(e.folderID)
			}
		}
	}
	walk(folderID)
	return out, nil
}

// TagExists reports whether the user has a tag with the given ID.
func (s *Store) TagExists(ctx context.Context, userID, tagID string) (bool, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM tags WHERE id = ? AND user_id = ?`, tagID, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
