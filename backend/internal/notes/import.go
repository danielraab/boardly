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
	// IDs, if not nil, is filled with the SourceID of each category and note
	// mapped to its ID in the user's notes.
	IDs map[string]string
}

// ImportCategory is a category of an import file. The name of the tree's
// root is ignored; its contents go into the import category.
type ImportCategory struct {
	Name       string
	Categories []ImportCategory
	Notes      []ImportNote
	// SourceID, if set, is the category's ID in the file; IDs maps it to
	// the created or reused category.
	SourceID string
	// Icon and Color are set on a created category; a reused one keeps its own.
	Icon  *Icon
	Color *string
}

// ImportNote is a note of an import file. A zero CreatedAt means the import
// time; it is also the note's time of last change. Imported notes are never
// pinned.
type ImportNote struct {
	Title     *string
	Content   Doc
	CreatedAt time.Time
	Archived  bool
	// SourceID, if set, is the note's ID in the file; IDs maps it to the
	// created note, or to the existing note a duplicate was skipped for.
	SourceID string
	Icon     *Icon
	Color    *string
	Pinned   bool
	// UpdatedAt is the time of last change; zero means CreatedAt.
	UpdatedAt time.Time
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
	im := &importer{q: q, userID: userID, now: s.Now(), report: ImportReport{TooLargeNotes: tree.TooLarge}, ids: tree.IDs}
	if im.ids == nil {
		im.ids = map[string]string{}
	}
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
	// seen maps the duplicate keys of the user's notes, including those
	// imported so far, to their IDs, or is nil when duplicates are imported.
	seen   map[string]string
	report ImportReport
	// ids receives the SourceIDs of the imported, reused and skipped items.
	ids map[string]string
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
			symbol, imageID, err := checkIcon(ctx, im.q, im.userID, sub.Icon)
			if err != nil {
				return err
			}
			if err := checkColor(sub.Color); err != nil {
				return err
			}
			id = uuid.NewString()
			now := db.FormatTime(im.now)
			if _, err := im.q.ExecContext(ctx,
				`INSERT INTO note_categories (id, user_id, parent_id, title, icon_symbol, icon_image_id, color, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				id, im.userID, parentID, title, symbol, imageID, sub.Color, now, now,
			); err != nil {
				return err
			}
			children[strings.ToLower(title)] = id
			im.report.Categories++
		}
		if sub.SourceID != "" {
			im.ids[sub.SourceID] = id
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
	doc, content, err := parseContent(raw)
	if err != nil {
		return err
	}
	var key string
	if im.seen != nil {
		if key, err = noteKey(title, content); err != nil {
			return err
		}
		if existing, ok := im.seen[key]; ok {
			im.report.DuplicateNotes++
			if n.SourceID != "" {
				im.ids[n.SourceID] = existing
			}
			return nil
		}
	}
	symbol, imageID, err := checkIcon(ctx, im.q, im.userID, n.Icon)
	if err != nil {
		return err
	}
	if err := checkColor(n.Color); err != nil {
		return err
	}
	created := n.CreatedAt
	if created.IsZero() {
		created = im.now
	}
	updated := n.UpdatedAt
	if updated.IsZero() {
		updated = created
	}
	id := uuid.NewString()
	if _, err := im.q.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, category_id, title, icon_symbol, icon_image_id, color, content, pinned, archived, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		id, im.userID, categoryID, title, symbol, imageID, n.Color, content, n.Pinned, n.Archived, db.FormatTime(created), db.FormatTime(updated),
	); err != nil {
		return err
	}
	if im.seen != nil {
		im.seen[key] = id
	}
	if n.SourceID != "" {
		im.ids[n.SourceID] = id
	}
	if err := setAttachments(ctx, im.q, im.userID, id, doc); err != nil {
		return err
	}
	im.report.Notes++
	return nil
}

// noteKeys maps the duplicate keys of the user's notes to their IDs; of
// notes with the same key, the oldest wins.
func noteKeys(ctx context.Context, q db.Querier, userID string) (map[string]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, title, content FROM notes WHERE user_id = ? ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	keys := map[string]string{}
	for rows.Next() {
		var id string
		var title sql.NullString
		var content string
		if err := rows.Scan(&id, &title, &content); err != nil {
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
		if _, ok := keys[key]; !ok {
			keys[key] = id
		}
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
