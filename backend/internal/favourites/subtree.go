package favourites

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"famodash/internal/db"
)

// ErrTagExists is returned when a tag is renamed to the name of another tag
// of the user, regardless of case.
var ErrTagExists = errors.New("tag name already exists")

// ScopedNode is a node of a scope's subtree, as SubtreeNodes returns it.
type ScopedNode struct {
	ID string
	// Num is the node's integer ID for apps.
	Num int64
	// ParentID is empty for the scope's root.
	ParentID  string
	Kind      Kind
	Position  int
	Title     *string
	URL       *string
	CreatedAt time.Time
	UpdatedAt time.Time
	// Tags are the tags of a link, sorted by name.
	Tags []Tag
}

// subtreeCTE selects the IDs of the scope's root (?1, owned by user ?2) and
// every node below it.
const subtreeCTE = `WITH RECURSIVE sub (id) AS (
	SELECT id FROM favourite_nodes WHERE id = ? AND user_id = ?
	UNION ALL
	SELECT n.id FROM favourite_nodes n JOIN sub ON n.parent_id = sub.id
)`

// SubtreeNodes returns the scope's root and every node below it, folders and
// links, ordered by parent and position. It is empty when the root isn't one
// of the user's folders.
func (s *Store) SubtreeNodes(ctx context.Context, sc Scope) ([]ScopedNode, error) {
	rows, err := s.DB.QueryContext(ctx, subtreeCTE+`
		SELECT n.id, n.num, n.parent_id, n.kind, n.position, n.title, n.url, n.created_at, n.updated_at
		FROM favourite_nodes n JOIN sub ON n.id = sub.id
		ORDER BY n.parent_id, n.position`, sc.RootID, sc.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := []ScopedNode{}
	index := map[string]int{}
	for rows.Next() {
		var n ScopedNode
		var parentID, title, link sql.NullString
		var created, updated string
		if err := rows.Scan(&n.ID, &n.Num, &parentID, &n.Kind, &n.Position, &title, &link, &created, &updated); err != nil {
			return nil, err
		}
		if n.ID != sc.RootID {
			n.ParentID = parentID.String
		}
		n.Title, n.URL = nullable(title), nullable(link)
		if n.CreatedAt, err = db.ParseTime(created); err != nil {
			return nil, err
		}
		if n.UpdatedAt, err = db.ParseTime(updated); err != nil {
			return nil, err
		}
		index[n.ID] = len(nodes)
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(nodes) > 0 && nodes[index[sc.RootID]].Kind != KindFolder {
		return []ScopedNode{}, nil
	}

	tags, err := s.DB.QueryContext(ctx, subtreeCTE+`
		SELECT nt.node_id, t.id, t.num, t.name, t.created_at
		FROM favourite_node_tags nt JOIN sub ON nt.node_id = sub.id JOIN tags t ON t.id = nt.tag_id
		ORDER BY t.name_key`, sc.RootID, sc.UserID)
	if err != nil {
		return nil, err
	}
	defer tags.Close()
	for tags.Next() {
		var nodeID string
		var t Tag
		var created string
		if err := tags.Scan(&nodeID, &t.ID, &t.Num, &t.Name, &created); err != nil {
			return nil, err
		}
		if t.CreatedAt, err = db.ParseTime(created); err != nil {
			return nil, err
		}
		n := &nodes[index[nodeID]]
		n.Tags = append(n.Tags, t)
	}
	return nodes, tags.Err()
}

// TagLinkCount returns on how many links the user's tag is, or ErrNotFound.
func (s *Store) TagLinkCount(ctx context.Context, userID, tagID string) (int, error) {
	var exists bool
	var count int
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM tags WHERE id = ? AND user_id = ?),
		        (SELECT COUNT(*) FROM favourite_node_tags WHERE tag_id = ?)`,
		tagID, userID, tagID,
	).Scan(&exists, &count)
	if err == nil && !exists {
		return 0, ErrNotFound
	}
	return count, err
}

// RenameTag renames the user's tag on all its links. The tag-name rules
// apply; a name that another tag of the user has, regardless of case,
// returns ErrTagExists.
func (s *Store) RenameTag(ctx context.Context, userID, tagID, name string) (Tag, error) {
	names, err := NormalizeTags([]string{name})
	if err != nil {
		return Tag{}, err
	}
	name = names[0]
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Tag{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var taken bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM tags WHERE user_id = ? AND name_key = ? AND id <> ?)`,
		userID, tagKey(name), tagID,
	).Scan(&taken); err != nil {
		return Tag{}, err
	}
	if taken {
		return Tag{}, ErrTagExists
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE tags SET name = ?, name_key = ? WHERE id = ? AND user_id = ?`,
		name, tagKey(name), tagID, userID)
	if err != nil {
		return Tag{}, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err == nil {
			err = ErrNotFound
		}
		return Tag{}, err
	}
	return Tag{ID: tagID, Name: name}, tx.Commit()
}

// DetachTag removes the user's tag from the given links and then removes the
// tag if it is on no link anymore. Links outside nodeIDs keep it.
func (s *Store) DetachTag(ctx context.Context, userID, tagID string, nodeIDs []string) error {
	if _, err := s.TagLinkCount(ctx, userID, tagID); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if len(nodeIDs) > 0 {
		in := strings.Repeat(",?", len(nodeIDs))[1:]
		args := []any{db.FormatTime(s.Now()), userID, tagID}
		for _, id := range nodeIDs {
			args = append(args, id)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE favourite_nodes SET updated_at = ? WHERE user_id = ? AND id IN (
				SELECT node_id FROM favourite_node_tags WHERE tag_id = ? AND node_id IN (`+in+`))`,
			args...); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM favourite_node_tags WHERE tag_id = ? AND node_id IN (`+in+`)`,
			args[2:]...); err != nil {
			return err
		}
	}
	if err := removeUnusedTags(ctx, tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}
