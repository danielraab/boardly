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

// KindSection is the kind of a section, the only kind allowed at the top
// level and the only kind with children.
const KindSection = "section"

// Widget is one node of a layout: a section with its children, in order, or
// a content widget of another kind with its config.
type Widget struct {
	// ID is the stored ID. On save it may be empty for a new widget; an ID
	// that isn't one of the dashboard's widgets is replaced.
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Width  int    `json:"width"`
	Height Height `json:"height"`

	// Title, Border and Children are only used by sections.
	Title    *string  `json:"title"`
	Border   bool     `json:"border"`
	Children []Widget `json:"children"`

	// Config holds the settings of a content widget, as its kind defines them.
	Config json.RawMessage `json:"config"`
	// Data is what the kind resolved for a lookup. It is never stored.
	Data any `json:"data"`
}

// MarshalJSON writes a section with its section fields and a content widget
// with its config and data.
func (w Widget) MarshalJSON() ([]byte, error) {
	if w.Kind == KindSection {
		children := w.Children
		if children == nil {
			children = []Widget{}
		}
		return json.Marshal(struct {
			ID       string   `json:"id"`
			Kind     string   `json:"kind"`
			Width    int      `json:"width"`
			Height   Height   `json:"height"`
			Title    *string  `json:"title"`
			Border   bool     `json:"border"`
			Children []Widget `json:"children"`
		}{w.ID, w.Kind, w.Width, w.Height, w.Title, w.Border, children})
	}
	config := w.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return json.Marshal(struct {
		ID     string          `json:"id"`
		Kind   string          `json:"kind"`
		Width  int             `json:"width"`
		Height Height          `json:"height"`
		Config json.RawMessage `json:"config"`
		Data   any             `json:"data,omitempty"`
	}{w.ID, w.Kind, w.Width, w.Height, config, w.Data})
}

// Kind is a content widget kind. Normalize validates and canonicalises a
// config of userID's widget; it returns an error wrapping ErrInvalidLayout for
// a config it doesn't accept. Resolve returns the data of a widget for a
// lookup, or ok=false to leave the widget out of the response. Resolve may
// be nil when a kind has no data and is shown to everyone. A FitContent
// kind is always as tall as its content: its widgets are stored with height
// "auto", whatever height they are sent with.
type Kind struct {
	Normalize  func(ctx context.Context, userID string, config json.RawMessage) (json.RawMessage, error)
	Resolve    func(ctx context.Context, viewer Viewer, config json.RawMessage) (data any, ok bool, err error)
	FitContent bool
}

// Viewer is who a layout is resolved for.
type Viewer struct {
	OwnerID string
	IsOwner bool
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
	Version  int      `json:"version"`
	Sections []Widget `json:"sections"`
}

// SaveLayout replaces the layout of one of the user's dashboards, if version
// is still its current layout version. The whole tree is validated first. The
// returned layout is resolved for the owner.
func (s *Store) SaveLayout(ctx context.Context, userID, dashboardID string, version int, sections []Widget) (Layout, error) {
	if err := s.normalizeLayout(ctx, userID, sections); err != nil {
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
	if err := insertWidgets(ctx, tx, dashboardID, nil, sections, existing); err != nil {
		return Layout{}, err
	}
	saved, err := loadWidgets(ctx, tx, dashboardID)
	if err != nil {
		return Layout{}, err
	}
	if err := tx.Commit(); err != nil {
		return Layout{}, err
	}
	saved, err = s.resolve(ctx, Viewer{OwnerID: userID, IsOwner: true}, saved)
	if err != nil {
		return Layout{}, err
	}
	return Layout{Version: next, Sections: saved}, nil
}

// normalizeLayout checks the limits and the kinds, trims section titles
// (empty becomes nil), drops data and lets each kind normalize its config.
// Only sections count as nesting levels.
func (s *Store) normalizeLayout(ctx context.Context, userID string, sections []Widget) error {
	count := 0
	var walk func(list []Widget, level int) error
	walk = func(list []Widget, level int) error {
		for i := range list {
			w := &list[i]
			count++
			if count > MaxWidgets {
				return fmt.Errorf("%w: at most %d widgets", ErrInvalidLayout, MaxWidgets)
			}
			if w.Width < 1 || w.Width > MaxColumns {
				return fmt.Errorf("%w: width must be 1 to %d", ErrInvalidLayout, MaxColumns)
			}
			if w.Height < 0 || w.Height > MaxHeightRows {
				return fmt.Errorf("%w: height must be \"auto\" or 1 to %d rows", ErrInvalidLayout, MaxHeightRows)
			}
			w.Data = nil
			if w.Kind != KindSection {
				if err := s.normalizeContent(ctx, userID, w, level); err != nil {
					return err
				}
				continue
			}
			if level > MaxDepth {
				return fmt.Errorf("%w: sections nest at most %d levels deep", ErrInvalidLayout, MaxDepth)
			}
			if w.Config != nil {
				return fmt.Errorf("%w: sections have no config", ErrInvalidLayout)
			}
			if w.Title != nil {
				title := strings.TrimSpace(*w.Title)
				if utf8.RuneCountInString(title) > MaxSectionTitleLength {
					return fmt.Errorf("%w: titles have at most %d characters", ErrInvalidLayout, MaxSectionTitleLength)
				}
				if title == "" {
					w.Title = nil
				} else {
					w.Title = &title
				}
			}
			if err := walk(w.Children, level+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(sections, 1)
}

// normalizeContent checks a content widget at level (1 is the top level).
func (s *Store) normalizeContent(ctx context.Context, userID string, w *Widget, level int) error {
	kind, ok := s.Kinds[w.Kind]
	if !ok {
		return fmt.Errorf("%w: unknown widget kind %q", ErrInvalidLayout, w.Kind)
	}
	if level == 1 {
		return fmt.Errorf("%w: only sections are allowed at the top level", ErrInvalidLayout)
	}
	if w.Title != nil || w.Border || w.Children != nil {
		return fmt.Errorf("%w: only sections have a title, a border and children", ErrInvalidLayout)
	}
	config := w.Config
	if config == nil || string(config) == "null" {
		config = json.RawMessage(`{}`)
	}
	normalized, err := kind.Normalize(ctx, userID, config)
	if err != nil {
		return err
	}
	w.Config = normalized
	if kind.FitContent {
		w.Height = 0
	}
	return nil
}

// resolveCacheKey is the context key of the cache of one resolve.
type resolveCacheKey struct{}

// Cached returns the value that load returned for key earlier in the same
// lookup, or calls load and remembers its result. Kinds use it so several
// widgets showing the same thing load it once. Outside a lookup it calls load.
func Cached[T any](ctx context.Context, key any, load func() (T, error)) (T, error) {
	cache, _ := ctx.Value(resolveCacheKey{}).(map[any]any)
	if v, ok := cache[key]; ok {
		return v.(T), nil
	}
	v, err := load()
	if err == nil && cache != nil {
		cache[key] = v
	}
	return v, err
}

// resolve fills in the data of content widgets for viewer and leaves out the
// widgets their kind hides, with one cache for the whole tree.
func (s *Store) resolve(ctx context.Context, viewer Viewer, list []Widget) ([]Widget, error) {
	return s.resolveWidgets(context.WithValue(ctx, resolveCacheKey{}, map[any]any{}), viewer, list)
}

// resolveWidgets is resolve without a new cache. It returns a new tree.
func (s *Store) resolveWidgets(ctx context.Context, viewer Viewer, list []Widget) ([]Widget, error) {
	out := make([]Widget, 0, len(list))
	for _, w := range list {
		if w.Kind == KindSection {
			children, err := s.resolveWidgets(ctx, viewer, w.Children)
			if err != nil {
				return nil, err
			}
			w.Children = children
			out = append(out, w)
			continue
		}
		kind, ok := s.Kinds[w.Kind]
		if !ok {
			// A kind that was removed from the binary isn't shown.
			continue
		}
		if kind.Resolve != nil {
			data, show, err := kind.Resolve(ctx, viewer, w.Config)
			if err != nil {
				return nil, fmt.Errorf("resolving widget %s: %w", w.ID, err)
			}
			if !show {
				continue
			}
			w.Data = data
		}
		out = append(out, w)
	}
	return out, nil
}

// contentConfig is what a content widget stores in dashboard_widgets.config:
// its height and the config of its kind.
type contentConfig struct {
	Height Height          `json:"height"`
	Config json.RawMessage `json:"config"`
}

// widgetConfig returns what a widget stores in dashboard_widgets.config.
func widgetConfig(w Widget) ([]byte, error) {
	if w.Kind != KindSection {
		return json.Marshal(contentConfig{Height: w.Height, Config: w.Config})
	}
	return json.Marshal(sectionConfig{Title: w.Title, Border: w.Border, Height: w.Height})
}

// insertWidgets inserts the tree depth-first, so parents exist before their
// children. An ID is kept only once and only if it was one of the dashboard's.
func insertWidgets(ctx context.Context, q db.Querier, dashboardID string, parentID *string, list []Widget, existing map[string]bool) error {
	for pos, w := range list {
		id := w.ID
		if !existing[id] {
			id = uuid.NewString()
		}
		delete(existing, id)
		config, err := widgetConfig(w)
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, dashboardID, parentID, w.Kind, pos, w.Width, string(config)); err != nil {
			return err
		}
		if err := insertWidgets(ctx, q, dashboardID, &id, w.Children, existing); err != nil {
			return err
		}
	}
	return nil
}

// loadWidgets reads a dashboard's widgets as a tree.
func loadWidgets(ctx context.Context, q db.Querier, dashboardID string) ([]Widget, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, parent_id, kind, width, config FROM dashboard_widgets
		 WHERE dashboard_id = ? ORDER BY position`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byParent := map[string][]Widget{}
	for rows.Next() {
		var w Widget
		var parentID sql.NullString
		var config string
		if err := rows.Scan(&w.ID, &parentID, &w.Kind, &w.Width, &config); err != nil {
			return nil, err
		}
		if w.Kind == KindSection {
			var c sectionConfig
			if err := json.Unmarshal([]byte(config), &c); err != nil {
				return nil, fmt.Errorf("section %s: %w", w.ID, err)
			}
			w.Title, w.Border, w.Height = c.Title, c.Border, c.Height
		} else {
			var c contentConfig
			if err := json.Unmarshal([]byte(config), &c); err != nil {
				return nil, fmt.Errorf("widget %s: %w", w.ID, err)
			}
			w.Height, w.Config = c.Height, c.Config
		}
		byParent[parentID.String] = append(byParent[parentID.String], w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var build func(parentID string) []Widget
	build = func(parentID string) []Widget {
		list := byParent[parentID]
		if list == nil {
			return []Widget{}
		}
		for i := range list {
			if list[i].Kind == KindSection {
				list[i].Children = build(list[i].ID)
			}
		}
		return list
	}
	return build(""), nil
}
