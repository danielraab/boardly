package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// ImportTree is the notes of an import file, as a parser of the importer
// package reads it, with the number of notes it left out as too large.
type ImportTree struct {
	Root     ImportCategory
	TooLarge int
}

// ImportCategory is a category of an import file. The name of the tree's
// root is ignored; its contents go into the import category.
type ImportCategory struct {
	Name       string
	Categories []ImportCategory
	Notes      []ImportNote
}

// ImportNote is a note of an import file. A zero CreatedAt means the import
// time; it is also the note's time of last change. Imported notes are never
// pinned.
type ImportNote struct {
	Title     *string
	Content   Doc
	CreatedAt time.Time
	Archived  bool
}

// ImportReport counts the notes and categories an import added and left out.
type ImportReport struct {
	Notes          int
	Categories     int
	DuplicateNotes int
	TooLargeNotes  int
}

// ImportTx adds the tree's categories and notes below the user's category
// categoryID (nil for the top level) in the transaction q, and returns what
// it added. A category whose title equals an existing child category's,
// ignoring case, is reused. With skipDuplicates, a note with the same title
// and content as an existing note, or one imported before it, is skipped.
// Errors are ErrInvalidCategory and those of the database.
func (s *Store) ImportTx(ctx context.Context, q db.Querier, userID string, categoryID *string, tree ImportTree, skipDuplicates bool) (ImportReport, error) {
	if err := checkCategory(ctx, q, userID, categoryID); err != nil {
		return ImportReport{}, err
	}
	im := &importer{q: q, userID: userID, now: s.Now(), report: ImportReport{TooLargeNotes: tree.TooLarge}}
	if skipDuplicates {
		var err error
		if im.seen, err = noteKeys(ctx, q, userID); err != nil {
			return ImportReport{}, err
		}
	}
	if err := im.add(ctx, categoryID, tree.Root); err != nil {
		return ImportReport{}, err
	}
	return im.report, nil
}

// importer is the state of one ImportTx call.
type importer struct {
	q      db.Querier
	userID string
	now    time.Time
	// seen holds the duplicate keys of the user's notes, including those
	// imported so far, or is nil when duplicates are imported.
	seen   map[string]bool
	report ImportReport
}

// add adds the notes and subcategories of c to the category parentID.
func (im *importer) add(ctx context.Context, parentID *string, c ImportCategory) error {
	for _, n := range c.Notes {
		if err := im.addNote(ctx, parentID, n); err != nil {
			return err
		}
	}
	if len(c.Categories) == 0 {
		return nil
	}
	children, err := im.childCategories(ctx, parentID)
	if err != nil {
		return err
	}
	for _, sub := range c.Categories {
		title, err := normalizeCategoryTitle(sub.Name)
		if err != nil {
			return err
		}
		id, ok := children[strings.ToLower(title)]
		if !ok {
			id = uuid.NewString()
			now := db.FormatTime(im.now)
			if _, err := im.q.ExecContext(ctx,
				`INSERT INTO note_categories (id, user_id, parent_id, title, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				id, im.userID, parentID, title, now, now,
			); err != nil {
				return err
			}
			children[strings.ToLower(title)] = id
			im.report.Categories++
		}
		if err := im.add(ctx, &id, sub); err != nil {
			return err
		}
	}
	return nil
}

// childCategories returns the IDs of the child categories of parentID (nil
// for the top level) by lowercase title; of equal titles the first is kept.
func (im *importer) childCategories(ctx context.Context, parentID *string) (map[string]string, error) {
	query := `SELECT id, title FROM note_categories WHERE user_id = ? AND parent_id IS NULL ORDER BY created_at, id`
	args := []any{im.userID}
	if parentID != nil {
		query = `SELECT id, title FROM note_categories WHERE user_id = ? AND parent_id = ? ORDER BY created_at, id`
		args = append(args, *parentID)
	}
	rows, err := im.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	children := map[string]string{}
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		if _, ok := children[strings.ToLower(title)]; !ok {
			children[strings.ToLower(title)] = id
		}
	}
	return children, rows.Err()
}

// addNote inserts the note into the category, unless it is a skipped
// duplicate.
func (im *importer) addNote(ctx context.Context, categoryID *string, n ImportNote) error {
	title, err := normalizeNoteTitle(n.Title)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(n.Content)
	if err != nil {
		return err
	}
	content, err := parseContent(raw)
	if err != nil {
		return err
	}
	if im.seen != nil {
		key, err := noteKey(title, content)
		if err != nil {
			return err
		}
		if im.seen[key] {
			im.report.DuplicateNotes++
			return nil
		}
		im.seen[key] = true
	}
	created := n.CreatedAt
	if created.IsZero() {
		created = im.now
	}
	at := db.FormatTime(created)
	if _, err := im.q.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, category_id, title, content, archived, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		uuid.NewString(), im.userID, categoryID, title, content, n.Archived, at, at,
	); err != nil {
		return err
	}
	im.report.Notes++
	return nil
}

// noteKeys returns the duplicate keys of the user's notes.
func noteKeys(ctx context.Context, q db.Querier, userID string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT title, content FROM notes WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	keys := map[string]bool{}
	for rows.Next() {
		var title sql.NullString
		var content string
		if err := rows.Scan(&title, &content); err != nil {
			return nil, err
		}
		var t *string
		if title.Valid {
			t = &title.String
		}
		key, err := noteKey(t, content)
		if err != nil {
			return nil, err
		}
		keys[key] = true
	}
	return keys, rows.Err()
}

// noteKey is what makes two notes duplicates: the same title and the same
// content apart from task IDs.
func noteKey(title *string, content string) (string, error) {
	var d Doc
	if err := json.Unmarshal([]byte(content), &d); err != nil {
		return "", err
	}
	var clear func(b *Block)
	clear = func(b *Block) {
		if b.Items == nil {
			return
		}
		for i := range *b.Items {
			it := &(*b.Items)[i]
			it.ID = ""
			if it.Children != nil {
				clear(it.Children)
			}
		}
	}
	for i := range d.Blocks {
		clear(&d.Blocks[i])
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	t := ""
	if title != nil {
		t = *title
	}
	return t + "\x00" + string(b), nil
}
