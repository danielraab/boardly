package dashboards

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"famodash/internal/db"
)

// Limits of a layout.
const (
	MaxDepth              = 5
	MaxWidgets            = 500
	MaxHeightRows         = 50
	MaxSectionTitleLength = 100
)

// Errors returned by SaveLayout.
var (
	ErrInvalidLayout  = errors.New("invalid layout")
	ErrLayoutConflict = errors.New("layout was changed in the meantime")
)

// Section is a section widget with its children, in order.
type Section struct {
	// ID is the stored ID. On save it may be empty for a new section; an ID
	// that isn't one of the dashboard's widgets is replaced.
	ID       string    `json:"id"`
	Width    int       `json:"width"`
	Title    *string   `json:"title"`
	Border   bool      `json:"border"`
	Height   Height    `json:"height"`
	Children []Section `json:"children"`
}

// Height is a section's height in rows; 0 means "auto". In JSON it is the
// string "auto" or the number of rows.
type Height int

func (h Height) MarshalJSON() ([]byte, error) {
	if h == 0 {
		return []byte(`"auto"`), nil
	}
	return json.Marshal(int(h))
}

func (h *Height) UnmarshalJSON(data []byte) error {
	if string(data) == `"auto"` {
		*h = 0
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil || n < 1 {
		return fmt.Errorf("height must be \"auto\" or a positive number of rows")
	}
	*h = Height(n)
	return nil
}

// sectionConfig is what a section stores in dashboard_widgets.config.
type sectionConfig struct {
	Title  *string `json:"title"`
	Border bool    `json:"border"`
	Height Height  `json:"height"`
}

// Layout is a saved layout with its version.
type Layout struct {
	Version  int       `json:"version"`
	Sections []Section `json:"sections"`
}

// SaveLayout replaces the layout of one of the user's dashboards, if version
// is still its current layout version. The whole tree is validated first.
func (s *Store) SaveLayout(ctx context.Context, userID, dashboardID string, version int, sections []Section) (Layout, error) {
	if err := normalizeLayout(sections); err != nil {
		return Layout{}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Layout{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var next int
	err = tx.QueryRowContext(ctx,
		`UPDATE dashboards SET layout_version = layout_version + 1, updated_at = ?
		 WHERE id = ? AND user_id = ? AND layout_version = ? RETURNING layout_version`,
		db.FormatTime(s.Now()), dashboardID, userID, version).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM dashboards WHERE id = ? AND user_id = ?)`, dashboardID, userID).Scan(&exists); err != nil {
			return Layout{}, err
		}
		if exists {
			return Layout{}, ErrLayoutConflict
		}
		return Layout{}, ErrNotFound
	}
	if err != nil {
		return Layout{}, err
	}

	existing := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM dashboard_widgets WHERE dashboard_id = ?`, dashboardID)
	if err != nil {
		return Layout{}, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return Layout{}, err
		}
		existing[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Layout{}, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM dashboard_widgets WHERE dashboard_id = ?`, dashboardID); err != nil {
		return Layout{}, err
	}
	if err := insertSections(ctx, tx, dashboardID, nil, sections, existing); err != nil {
		return Layout{}, err
	}
	saved, err := loadSections(ctx, tx, dashboardID)
	if err != nil {
		return Layout{}, err
	}
	return Layout{Version: next, Sections: saved}, tx.Commit()
}

// normalizeLayout trims titles (empty becomes nil) and checks the limits.
func normalizeLayout(sections []Section) error {
	count := 0
	var walk func(list []Section, level int) error
	walk = func(list []Section, level int) error {
		if level > MaxDepth && len(list) > 0 {
			return fmt.Errorf("%w: sections nest at most %d levels deep", ErrInvalidLayout, MaxDepth)
		}
		for i := range list {
			sec := &list[i]
			count++
			if count > MaxWidgets {
				return fmt.Errorf("%w: at most %d widgets", ErrInvalidLayout, MaxWidgets)
			}
			if sec.Width < 1 || sec.Width > MaxColumns {
				return fmt.Errorf("%w: width must be 1 to %d", ErrInvalidLayout, MaxColumns)
			}
			if sec.Height < 0 || sec.Height > MaxHeightRows {
				return fmt.Errorf("%w: height must be \"auto\" or 1 to %d rows", ErrInvalidLayout, MaxHeightRows)
			}
			if sec.Title != nil {
				title := strings.TrimSpace(*sec.Title)
				if utf8.RuneCountInString(title) > MaxSectionTitleLength {
					return fmt.Errorf("%w: titles have at most %d characters", ErrInvalidLayout, MaxSectionTitleLength)
				}
				if title == "" {
					sec.Title = nil
				} else {
					sec.Title = &title
				}
			}
			if err := walk(sec.Children, level+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(sections, 1)
}

// insertSections inserts the tree depth-first, so parents exist before their
// children. An ID is kept only once and only if it was one of the dashboard's.
func insertSections(ctx context.Context, q db.Querier, dashboardID string, parentID *string, list []Section, existing map[string]bool) error {
	for pos, sec := range list {
		id := sec.ID
		if !existing[id] {
			id = uuid.NewString()
		}
		delete(existing, id)
		config, err := json.Marshal(sectionConfig{Title: sec.Title, Border: sec.Border, Height: sec.Height})
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config)
			 VALUES (?, ?, ?, 'section', ?, ?, ?)`,
			id, dashboardID, parentID, pos, sec.Width, string(config)); err != nil {
			return err
		}
		if err := insertSections(ctx, q, dashboardID, &id, sec.Children, existing); err != nil {
			return err
		}
	}
	return nil
}

// loadSections reads a dashboard's widgets as a tree.
func loadSections(ctx context.Context, q db.Querier, dashboardID string) ([]Section, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, parent_id, width, config FROM dashboard_widgets
		 WHERE dashboard_id = ? AND kind = 'section' ORDER BY position`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byParent := map[string][]Section{}
	for rows.Next() {
		var sec Section
		var parentID sql.NullString
		var config string
		if err := rows.Scan(&sec.ID, &parentID, &sec.Width, &config); err != nil {
			return nil, err
		}
		var c sectionConfig
		if err := json.Unmarshal([]byte(config), &c); err != nil {
			return nil, fmt.Errorf("section %s: %w", sec.ID, err)
		}
		sec.Title, sec.Border, sec.Height = c.Title, c.Border, c.Height
		byParent[parentID.String] = append(byParent[parentID.String], sec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var build func(parentID string) []Section
	build = func(parentID string) []Section {
		list := byParent[parentID]
		if list == nil {
			return []Section{}
		}
		for i := range list {
			list[i].Children = build(list[i].ID)
		}
		return list
	}
	return build(""), nil
}
