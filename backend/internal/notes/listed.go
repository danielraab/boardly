package notes

import (
	"cmp"
	"context"
	"slices"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// SelectionMode is which of a user's notes a note list shows.
type SelectionMode string

// Selection modes of a note list.
const (
	SelectAll           SelectionMode = "all"
	SelectUncategorized SelectionMode = "none"
	SelectCategory      SelectionMode = "category"
)

// Selection is the notes a note list shows: all of them, those without a
// category, or those in CategoryID, with its subcategories at every depth
// when Subcategories is set.
type Selection struct {
	Mode          SelectionMode
	CategoryID    string
	Subcategories bool
}

// Sort orders of ListedNotes.
const (
	SortUpdated = "updated"
	SortCreated = "created"
	SortTitle   = "title"
)

// ListedNotes returns at most limit of the user's notes in sel, pinned ones
// first, each sorted by order: SortUpdated and SortCreated newest first, then by ID descending;
// SortTitle by display title ignoring case and accents, notes without one
// last, then by ID.
func (s *Store) ListedNotes(ctx context.Context, userID string, sel Selection, order string, limit int) ([]Note, error) {
	cond, args := `user_id = ?`, []any{userID}
	switch sel.Mode {
	case SelectUncategorized:
		cond += ` AND category_id IS NULL`
	case SelectCategory:
		if sel.Subcategories {
			sub, subArgs := subtreeQuery(userID, sel.CategoryID)
			cond += ` AND ` + sub
			args = append(args, subArgs...)
		} else {
			cond += ` AND category_id = ?`
			args = append(args, sel.CategoryID)
		}
	}
	// The display title only exists in Go, so a title sort loads every
	// selected note and limits after sorting. The column is one of two fixed
	// names, never from input.
	query := `SELECT ` + noteColumns + ` FROM notes WHERE ` + cond
	if order != SortTitle {
		col := "updated_at"
		if order == SortCreated {
			col = "created_at"
		}
		query += ` ORDER BY pinned DESC, ` + col + ` DESC, id DESC LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if order == SortTitle {
		out = sortByTitle(out)
		if len(out) > limit {
			out = out[:limit]
		}
	}
	return out, nil
}

// sortByTitle sorts pinned notes first, then by display title, ignoring case
// and accents, with untitled notes last and ties by ID.
func sortByTitle(list []Note) []Note {
	type titled struct {
		note  Note
		title *string
	}
	all := make([]titled, len(list))
	for i, n := range list {
		all[i] = titled{n, DisplayTitle(n)}
	}
	// A collator isn't safe for concurrent use, so each sort has its own.
	c := collate.New(language.Und, collate.IgnoreCase, collate.IgnoreDiacritics)
	slices.SortFunc(all, func(a, b titled) int {
		if a.note.Pinned != b.note.Pinned {
			if a.note.Pinned {
				return -1
			}
			return 1
		}
		switch {
		case a.title == nil && b.title == nil:
			return cmp.Compare(a.note.ID, b.note.ID)
		case a.title == nil:
			return 1
		case b.title == nil:
			return -1
		}
		return cmp.Or(c.CompareString(*a.title, *b.title), cmp.Compare(a.note.ID, b.note.ID))
	})
	for i, t := range all {
		list[i] = t.note
	}
	return list
}
