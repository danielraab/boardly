package notes

import (
	"context"
	"database/sql"
	"errors"
)

// ScopeMode is which of a user's notes a connected app sees.
type ScopeMode string

// Scope modes, matching the note scopes of app tokens.
const (
	ScopeAll      ScopeMode = "all"
	ScopeCategory ScopeMode = "category"
	ScopeNone     ScopeMode = "none"
)

// Scope is the part of a user's notes a connected app sees: all of them,
// those in CategoryID and its subcategories at every depth, or none.
type Scope struct {
	UserID     string
	Mode       ScopeMode
	CategoryID string
}

// scopeQuery returns the condition and arguments that limit notes to the
// scope, or ok false when it holds none.
func scopeQuery(sc Scope) (cond string, args []any, ok bool) {
	switch sc.Mode {
	case ScopeAll:
		return `user_id = ?`, []any{sc.UserID}, true
	case ScopeCategory:
		sub, subArgs := subtreeQuery(sc.UserID, sc.CategoryID)
		return `user_id = ? AND ` + sub, append([]any{sc.UserID}, subArgs...), true
	}
	return "", nil, false
}

// subtreeQuery returns the condition and arguments that limit notes to the
// category id of the user and its subcategories at every depth.
func subtreeQuery(userID, id string) (cond string, args []any) {
	return `category_id IN (
			WITH RECURSIVE sub (id) AS (
				SELECT id FROM note_categories WHERE id = ? AND user_id = ?
				UNION ALL SELECT c.id FROM note_categories c JOIN sub ON c.parent_id = sub.id
			) SELECT id FROM sub)`, []any{id, userID}
}

// ScopedNotes returns the notes in the scope, last changed first.
func (s *Store) ScopedNotes(ctx context.Context, sc Scope) ([]Note, error) {
	cond, args, ok := scopeQuery(sc)
	if !ok {
		return []Note{}, nil
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE `+cond+` ORDER BY updated_at DESC, id`, args...)
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

// ScopedNote returns the note id if it is in the scope, or ErrNotFound.
func (s *Store) ScopedNote(ctx context.Context, sc Scope, id string) (Note, error) {
	cond, args, ok := scopeQuery(sc)
	if !ok {
		return Note{}, ErrNotFound
	}
	n, err := scanNote(s.DB.QueryRowContext(ctx,
		`SELECT `+noteColumns+` FROM notes WHERE id = ? AND `+cond, append([]any{id}, args...)...))
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	return n, err
}
