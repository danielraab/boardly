package favourites

import (
	"context"
	"math"

	"boardly/internal/db"
)

// BulkAction is what Bulk does with its nodes.
type BulkAction string

// Bulk actions. Delete and the tag actions only take links; move takes any
// node except the root.
const (
	BulkDelete     BulkAction = "delete"
	BulkAddTags    BulkAction = "addTags"
	BulkRemoveTags BulkAction = "removeTags"
	BulkSetTags    BulkAction = "setTags"
	BulkMove       BulkAction = "move"
)

// MaxBulkNodes is the most nodes one Bulk call takes.
const MaxBulkNodes = 1000

// BulkOp is one action on many nodes. Tags is used by the tag actions,
// ParentID by move.
type BulkOp struct {
	IDs      []string
	Action   BulkAction
	Tags     []string
	ParentID string
}

// Bulk applies op to all its nodes in one transaction: all of them change,
// or none. Unknown IDs and other users' nodes return ErrNotFound; the root, a
// folder for delete or a tag action, and missing or duplicate fields return
// ErrInvalidField. A move appends the nodes to the end of the target folder in
// the given order; a node inside another given node, or a target inside a
// given node, returns ErrInvalidMove.
func (s *Store) Bulk(ctx context.Context, userID string, op BulkOp) error {
	if len(op.IDs) == 0 || len(op.IDs) > MaxBulkNodes {
		return ErrInvalidField
	}
	ids := map[string]bool{}
	for _, id := range op.IDs {
		if ids[id] {
			return ErrInvalidField
		}
		ids[id] = true
	}
	var tags []string
	switch op.Action {
	case BulkAddTags, BulkRemoveTags, BulkSetTags:
		var err error
		if tags, err = NormalizeTags(op.Tags); err != nil {
			return err
		}
		if len(tags) == 0 && op.Action != BulkSetTags {
			return ErrInvalidField
		}
	case BulkMove:
		if op.ParentID == "" {
			return ErrInvalidField
		}
	case BulkDelete:
	default:
		return ErrInvalidField
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range op.IDs {
		r, err := load(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		if !r.parentID.Valid || (op.Action != BulkMove && r.kind != KindLink) {
			return ErrInvalidField
		}
	}

	now := db.FormatTime(s.Now())
	switch op.Action {
	case BulkDelete:
		for _, id := range op.IDs {
			// Loaded again: earlier deletes shift the positions.
			r, err := load(ctx, tx, userID, id)
			if err != nil {
				return err
			}
			if err := shiftAfter(ctx, tx, r.parentID.String, r.position, -1); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM favourite_nodes WHERE id = ?`, id); err != nil {
				return err
			}
		}
	case BulkMove:
		target, err := load(ctx, tx, userID, op.ParentID)
		if err != nil {
			return err
		}
		if target.kind != KindFolder {
			return ErrInvalidParent
		}
		// No given node may lie inside another one, and the target not inside any.
		for _, id := range append([]string{op.ParentID}, op.IDs...) {
			above, err := ancestors(ctx, tx, id)
			if err != nil {
				return err
			}
			for _, a := range above {
				if ids[a] {
					return ErrInvalidMove
				}
			}
		}
		if ids[op.ParentID] {
			return ErrInvalidMove
		}
		for _, id := range op.IDs {
			if err := s.move(ctx, tx, userID, id, op.ParentID, math.MaxInt32); err != nil {
				return err
			}
		}
	default:
		for _, id := range op.IDs {
			current, err := nodeTags(ctx, tx, id)
			if err != nil {
				return err
			}
			next, err := bulkTags(op.Action, current, tags)
			if err != nil {
				return err
			}
			if err := s.setTags(ctx, tx, userID, id, next); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE favourite_nodes SET updated_at = ? WHERE id = ?`, now, id); err != nil {
				return err
			}
		}
	}
	if op.Action != BulkMove {
		if err := removeUnusedTags(ctx, tx, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// bulkTags returns a link's new tags for a tag action.
func bulkTags(action BulkAction, current, tags []string) ([]string, error) {
	switch action {
	case BulkAddTags:
		return NormalizeTags(append(current, tags...))
	case BulkRemoveTags:
		remove := map[string]bool{}
		for _, t := range tags {
			remove[tagKey(t)] = true
		}
		out := []string{}
		for _, t := range current {
			if !remove[tagKey(t)] {
				out = append(out, t)
			}
		}
		return out, nil
	}
	return tags, nil
}

// ancestors returns the IDs of the folders above id, nearest first.
func ancestors(ctx context.Context, q db.Querier, id string) ([]string, error) {
	rows, err := q.QueryContext(ctx,
		`WITH RECURSIVE up (id, parent_id) AS (
			SELECT id, parent_id FROM favourite_nodes WHERE id = ?
			UNION ALL
			SELECT n.id, n.parent_id FROM favourite_nodes n JOIN up ON n.id = up.parent_id
		 )
		 SELECT id FROM up WHERE id <> ?`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
