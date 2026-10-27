package notes

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// MaxCategoryTitle bounds a category's title, in characters after trimming.
const MaxCategoryTitle = 200

// Category is a note category as the API exposes it.
type Category struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Icon     *Icon   `json:"icon"`
	Color    *string `json:"color"`
	ParentID *string `json:"parentId"`
}

// NewCategory is the input of CreateCategory.
type NewCategory struct {
	Title    string
	Icon     *Icon
	Color    *string
	ParentID *string
}

// CategoryPatch lists the fields to change; nil fields stay as they are, and
// a nil inner pointer clears the field.
type CategoryPatch struct {
	Title    *string
	Icon     **Icon
	Color    **string
	ParentID **string
}

// normalizeCategoryTitle trims a title; empty or too long ones return
// ErrInvalidTitle.
func normalizeCategoryTitle(title string) (string, error) {
	t := strings.TrimSpace(title)
	if t == "" || utf8.RuneCountInString(t) > MaxCategoryTitle {
		return "", ErrInvalidTitle
	}
	return t, nil
}

const categoryColumns = `id, title, icon_symbol, icon_image_id, color, parent_id`

func scanCategory(row interface{ Scan(...any) error }) (Category, error) {
	var c Category
	var symbol, imageID *string
	if err := row.Scan(&c.ID, &c.Title, &symbol, &imageID, &c.Color, &c.ParentID); err != nil {
		return Category{}, err
	}
	c.Icon = iconOf(symbol, imageID)
	return c, nil
}

// Categories returns all of the user's categories, oldest first.
func (s *Store) Categories(ctx context.Context, userID string) ([]Category, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+categoryColumns+` FROM note_categories WHERE user_id = ? ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// category returns one of the user's categories, or ErrNotFound.
func category(ctx context.Context, q db.Querier, userID, id string) (Category, error) {
	c, err := scanCategory(q.QueryRowContext(ctx,
		`SELECT `+categoryColumns+` FROM note_categories WHERE id = ? AND user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return c, err
}

// CreateCategory adds a category. Errors are ErrInvalidTitle, ErrInvalidIcon,
// ErrInvalidColor and ErrInvalidParent.
func (s *Store) CreateCategory(ctx context.Context, userID string, n NewCategory) (Category, error) {
	title, err := normalizeCategoryTitle(n.Title)
	if err != nil {
		return Category{}, err
	}
	if err := checkColor(n.Color); err != nil {
		return Category{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Category{}, err
	}
	defer func() { _ = tx.Rollback() }()
	symbol, imageID, err := checkIcon(ctx, tx, userID, n.Icon)
	if err != nil {
		return Category{}, err
	}
	if n.ParentID != nil {
		if _, err := category(ctx, tx, userID, *n.ParentID); err != nil {
			return Category{}, parentError(err)
		}
	}
	id := uuid.NewString()
	now := db.FormatTime(s.Now())
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO note_categories (id, user_id, parent_id, title, icon_symbol, icon_image_id, color, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, n.ParentID, title, symbol, imageID, n.Color, now, now,
	); err != nil {
		return Category{}, err
	}
	if err := tx.Commit(); err != nil {
		return Category{}, err
	}
	return Category{ID: id, Title: title, Icon: iconOf(symbol, imageID), Color: n.Color, ParentID: n.ParentID}, nil
}

// UpdateCategory changes a category. A parent that is the category itself or
// one of its subcategories returns ErrInvalidParent; other errors are those
// of CreateCategory and ErrNotFound.
func (s *Store) UpdateCategory(ctx context.Context, userID, id string, p CategoryPatch) (Category, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Category{}, err
	}
	defer func() { _ = tx.Rollback() }()
	c, err := category(ctx, tx, userID, id)
	if err != nil {
		return Category{}, err
	}
	sets := []string{"updated_at = ?"}
	args := []any{db.FormatTime(s.Now())}
	if p.Title != nil {
		if c.Title, err = normalizeCategoryTitle(*p.Title); err != nil {
			return Category{}, err
		}
		sets, args = append(sets, "title = ?"), append(args, c.Title)
	}
	if p.Icon != nil {
		symbol, imageID, err := checkIcon(ctx, tx, userID, *p.Icon)
		if err != nil {
			return Category{}, err
		}
		c.Icon = iconOf(symbol, imageID)
		sets, args = append(sets, "icon_symbol = ?", "icon_image_id = ?"), append(args, symbol, imageID)
	}
	if p.Color != nil {
		if err := checkColor(*p.Color); err != nil {
			return Category{}, err
		}
		c.Color = *p.Color
		sets, args = append(sets, "color = ?"), append(args, c.Color)
	}
	if p.ParentID != nil {
		if parent := *p.ParentID; parent != nil {
			if err := checkParent(ctx, tx, userID, id, *parent); err != nil {
				return Category{}, err
			}
		}
		c.ParentID = *p.ParentID
		sets, args = append(sets, "parent_id = ?"), append(args, c.ParentID)
	}
	args = append(args, id, userID)
	if _, err := tx.ExecContext(ctx,
		`UPDATE note_categories SET `+strings.Join(sets, ", ")+` WHERE id = ? AND user_id = ?`, args...,
	); err != nil {
		return Category{}, err
	}
	return c, tx.Commit()
}

// checkParent returns ErrInvalidParent unless parent is one of the user's
// categories outside the subtree of id.
func checkParent(ctx context.Context, q db.Querier, userID, id, parent string) error {
	var cycle bool
	err := q.QueryRowContext(ctx,
		`WITH RECURSIVE up(id, parent_id) AS (
			SELECT id, parent_id FROM note_categories WHERE id = ? AND user_id = ?
			UNION ALL
			SELECT c.id, c.parent_id FROM note_categories AS c JOIN up ON c.id = up.parent_id
		)
		SELECT COALESCE(MAX(id = ?), 0) FROM up HAVING COUNT(*) > 0`,
		parent, userID, id,
	).Scan(&cycle)
	if errors.Is(err, sql.ErrNoRows) || cycle {
		return ErrInvalidParent
	}
	return err
}

func parentError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidParent
	}
	return err
}

// DeleteCategory deletes a category. Its subcategories move to its parent,
// and its notes become uncategorized without changing their time of last
// change.
func (s *Store) DeleteCategory(ctx context.Context, userID, id string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	c, err := category(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE note_categories SET parent_id = ? WHERE parent_id = ? AND user_id = ?`, c.ParentID, id, userID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM note_categories WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
