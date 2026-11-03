package dashboards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/slugs"
)

// StoredWidget turns a widget as dashboard_widgets stores it (its kind,
// width and config column, as the export carries it) back into a Widget,
// with children for a section.
func StoredWidget(kind string, width int, config json.RawMessage, children []Widget) (Widget, error) {
	w := Widget{Kind: kind, Width: width}
	if kind == KindSection {
		var c sectionConfig
		if err := json.Unmarshal(config, &c); err != nil {
			return Widget{}, fmt.Errorf("%w: section: %v", ErrInvalidLayout, err)
		}
		w.Title, w.Border, w.Height = c.Title, c.Border, c.Height
		w.Children = children
		if w.Children == nil {
			w.Children = []Widget{}
		}
		return w, nil
	}
	var c contentConfig
	if err := json.Unmarshal(config, &c); err != nil {
		return Widget{}, fmt.Errorf("%w: widget: %v", ErrInvalidLayout, err)
	}
	w.Height, w.Config = c.Height, c.Config
	return w, nil
}

// ImportTx adds a dashboard of an import with its sections in q: never the
// default and never public, with the first free slug of slug, slug-2,
// slug-3, … (a generated one when slug isn't valid). Content widgets whose
// kind or config isn't accepted are left out; if the layout is still not
// valid, all widgets are. It returns the dashboard and the number of
// widgets left out.
func (s *Store) ImportTx(ctx context.Context, q db.Querier, userID, title, slug string, cols Columns, sections []Widget) (Dashboard, int, error) {
	title, err := normalizeTitle(title)
	if err != nil {
		return Dashboard{}, 0, err
	}
	if !cols.valid() {
		cols = DefaultColumns
	}
	base, err := slugs.Normalize(slug)
	if err != nil {
		base = slugs.Generate()
	}
	slug, err = freeSlug(ctx, q, userID, base)
	if err != nil {
		return Dashboard{}, 0, err
	}

	kept, dropped := s.pruneLayout(ctx, userID, sections)
	if err := s.normalizeLayout(ctx, userID, kept); err != nil {
		dropped += countWidgets(kept)
		kept = []Widget{}
	}

	id := uuid.NewString()
	now := db.FormatTime(s.Now())
	d, err := scan(q.QueryRowContext(ctx,
		`INSERT INTO dashboards (id, user_id, slug, title, is_default, is_public,
		   desktop_columns, mobile_columns, mobile_breakpoint, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?) RETURNING `+columns,
		id, userID, slug, title, cols.DesktopColumns, cols.MobileColumns, cols.MobileBreakpoint, now, now))
	if err != nil {
		return Dashboard{}, 0, err
	}
	if err := insertWidgets(ctx, q, id, nil, kept, map[string]bool{}); err != nil {
		return Dashboard{}, 0, err
	}
	return d, dropped, nil
}

// pruneLayout returns the sections without the content widgets whose kind
// or config isn't accepted, and how many widgets it left out.
func (s *Store) pruneLayout(ctx context.Context, userID string, list []Widget) ([]Widget, int) {
	kept := []Widget{}
	dropped := 0
	for _, w := range list {
		if w.Kind == KindSection {
			children, n := s.pruneLayout(ctx, userID, w.Children)
			w.Children = children
			dropped += n
			kept = append(kept, w)
			continue
		}
		kind, ok := s.Kinds[w.Kind]
		config := w.Config
		if config == nil || string(config) == "null" {
			config = json.RawMessage(`{}`)
		}
		if !ok {
			dropped++
			continue
		}
		if _, err := kind.Normalize(ctx, userID, config); err != nil {
			dropped++
			continue
		}
		kept = append(kept, w)
	}
	return kept, dropped
}

func countWidgets(list []Widget) int {
	n := 0
	for _, w := range list {
		n += 1 + countWidgets(w.Children)
	}
	return n
}

// freeSlug returns base if the user has no dashboard with it, otherwise the
// first free base-2, base-3, ….
func freeSlug(ctx context.Context, q db.Querier, userID, base string) (string, error) {
	for i := 1; ; i++ {
		candidate := base
		if i > 1 {
			candidate = base + "-" + strconv.Itoa(i)
			if _, err := slugs.Normalize(candidate); err != nil {
				// Too long with the suffix: start over from a generated slug.
				base, i = slugs.Generate(), 0
				continue
			}
		}
		var taken bool
		err := q.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM dashboards WHERE user_id = ? AND slug = ?)`, userID, candidate).Scan(&taken)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		if i > 10_000 {
			return "", errors.New("no free dashboard slug")
		}
	}
}
