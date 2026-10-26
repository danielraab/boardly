package favourites

import (
	"context"
	"database/sql"
	"errors"

	"boardly/internal/db"
)

// TagUsage is a tag of a user with the number of links that have it.
type TagUsage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LinkCount int    `json:"linkCount"`
}

// ListTags returns all of the user's tags with their link counts, sorted by
// name ignoring case.
func (s *Store) ListTags(ctx context.Context, userID string) ([]TagUsage, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT t.id, t.name, COUNT(nt.node_id) FROM tags t
		 LEFT JOIN favourite_node_tags nt ON nt.tag_id = t.id
		 WHERE t.user_id = ? GROUP BY t.id ORDER BY t.name_key`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []TagUsage{}
	for rows.Next() {
		var t TagUsage
		if err := rows.Scan(&t.ID, &t.Name, &t.LinkCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// TagByName returns the user's tag with the given name, ignoring case, or
// ErrNotFound.
func (s *Store) TagByName(ctx context.Context, userID, name string) (Tag, error) {
	var t Tag
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, name FROM tags WHERE user_id = ? AND name_key = ?`,
		userID, tagKey(name),
	).Scan(&t.ID, &t.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	return t, err
}

// MergeTag puts the user's tag targetID on every link that has sourceID and
// then deletes sourceID. The target keeps its name. Unknown tags, other users'
// tags, or the same tag twice return ErrNotFound.
func (s *Store) MergeTag(ctx context.Context, userID, sourceID, targetID string) (Tag, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Tag{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var target Tag
	var sourceExists bool
	err = tx.QueryRowContext(ctx,
		`SELECT id, name, EXISTS (SELECT 1 FROM tags WHERE id = ? AND user_id = ?)
		 FROM tags WHERE id = ? AND user_id = ?`,
		sourceID, userID, targetID, userID,
	).Scan(&target.ID, &target.Name, &sourceExists)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!sourceExists || sourceID == targetID)) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, err
	}
	if err := touchTagLinks(ctx, tx, sourceID, db.FormatTime(s.Now())); err != nil {
		return Tag{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO favourite_node_tags (node_id, tag_id)
		 SELECT node_id, ? FROM favourite_node_tags WHERE tag_id = ?`, targetID, sourceID,
	); err != nil {
		return Tag{}, err
	}
	if err := deleteTag(ctx, tx, sourceID); err != nil {
		return Tag{}, err
	}
	return target, tx.Commit()
}

// DeleteTag removes the user's tag from all its links and deletes it. The
// links are kept. An unknown tag or another user's returns ErrNotFound.
func (s *Store) DeleteTag(ctx context.Context, userID, tagID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM tags WHERE id = ? AND user_id = ?)`, tagID, userID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if err := touchTagLinks(ctx, tx, tagID, db.FormatTime(s.Now())); err != nil {
		return err
	}
	if err := deleteTag(ctx, tx, tagID); err != nil {
		return err
	}
	return tx.Commit()
}

// touchTagLinks sets updated_at of every link that has the tag.
func touchTagLinks(ctx context.Context, q db.Querier, tagID, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE favourite_nodes SET updated_at = ?
		 WHERE id IN (SELECT node_id FROM favourite_node_tags WHERE tag_id = ?)`, now, tagID)
	return err
}

// deleteTag deletes a tag and its links to nodes.
func deleteTag(ctx context.Context, q db.Querier, tagID string) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM favourite_node_tags WHERE tag_id = ?`, tagID); err != nil {
		return err
	}
	_, err := q.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID)
	return err
}
