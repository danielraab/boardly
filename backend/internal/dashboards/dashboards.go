// Package dashboards stores the users' dashboards and resolves dashboard
// addresses (/<user-slug>/<dashboard-slug>) with their visibility rules.
package dashboards

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/slugs"
)

// Dashboard is a dashboard as its owner sees it.
type Dashboard struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	IsDefault bool   `json:"isDefault"`
	IsPublic  bool   `json:"isPublic"`
	Columns
}

// Columns are a dashboard's grid settings: the desktop column count, the
// mobile column count and the viewport width below which the mobile count applies.
type Columns struct {
	DesktopColumns   int `json:"desktopColumns"`
	MobileColumns    int `json:"mobileColumns"`
	MobileBreakpoint int `json:"mobileBreakpoint"`
}

// DefaultColumns are the grid settings of a new dashboard.
var DefaultColumns = Columns{DesktopColumns: 6, MobileColumns: 2, MobileBreakpoint: 768}

// MaxColumns is the highest desktop and mobile column count.
const MaxColumns = 12

// Breakpoints are the allowed mobile breakpoints in pixels.
var Breakpoints = []int{640, 768, 1024}

// Entry is a dashboard as anyone allowed to see it sees it.
type Entry struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// View is the result of resolving a dashboard address.
type View struct {
	// ID is only set for the owner, who needs it to save the layout.
	ID string `json:"id,omitempty"`
	// IsDefault and IsPublic are only set for the owner, who can edit them.
	IsDefault *bool  `json:"isDefault,omitempty"`
	IsPublic  *bool  `json:"isPublic,omitempty"`
	Dashboard Entry  `json:"dashboard"`
	OwnerSlug string `json:"ownerSlug"`
	IsOwner   bool   `json:"isOwner"`
	// Dashboards are the switcher entries: all of the owner's dashboards for
	// the owner, only the public ones for everyone else.
	Dashboards    []Entry  `json:"dashboards"`
	Columns       Columns  `json:"columns"`
	LayoutVersion int      `json:"layoutVersion"`
	Sections      []Widget `json:"sections"`
}

// Errors returned by the store.
var (
	ErrNotFound       = errors.New("dashboard not found")
	ErrInvalidTitle   = errors.New("invalid title")
	ErrInvalidSlug    = errors.New("invalid slug")
	ErrSlugTaken      = errors.New("slug is taken")
	ErrInvalidColumns = errors.New("invalid column settings")
)

// MaxTitleLength is the longest title, in characters, after trimming.
const MaxTitleLength = 100

// Store reads and writes dashboards.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
	// Kinds are the content widget kinds besides sections, by name.
	Kinds map[string]Kind
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now, Kinds: map[string]Kind{}}
}

// Fields are the values of a dashboard to create; nil column settings get
// their defaults.
type Fields struct {
	Title            string
	Slug             string
	IsDefault        bool
	IsPublic         bool
	DesktopColumns   *int
	MobileColumns    *int
	MobileBreakpoint *int
}

// Patch lists the fields to change; nil fields stay as they are.
type Patch struct {
	Title            *string
	Slug             *string
	IsDefault        *bool
	IsPublic         *bool
	DesktopColumns   *int
	MobileColumns    *int
	MobileBreakpoint *int
}

const columns = `id, slug, title, is_default, is_public, desktop_columns, mobile_columns, mobile_breakpoint`

// List returns the user's dashboards sorted by title, ignoring case.
func (s *Store) List(ctx context.Context, userID string) ([]Dashboard, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+columns+` FROM dashboards WHERE user_id = ? ORDER BY title COLLATE NOCASE, slug`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Dashboard{}
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Create adds a dashboard. If it is the default, the previous default loses the flag.
func (s *Store) Create(ctx context.Context, userID string, f Fields) (Dashboard, error) {
	title, err := normalizeTitle(f.Title)
	if err != nil {
		return Dashboard{}, err
	}
	slug, err := slugs.Normalize(f.Slug)
	if err != nil {
		return Dashboard{}, ErrInvalidSlug
	}
	cols := DefaultColumns
	if f.DesktopColumns != nil {
		cols.DesktopColumns = *f.DesktopColumns
	}
	if f.MobileColumns != nil {
		cols.MobileColumns = *f.MobileColumns
	}
	if f.MobileBreakpoint != nil {
		cols.MobileBreakpoint = *f.MobileBreakpoint
	}
	if !cols.valid() {
		return Dashboard{}, ErrInvalidColumns
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Dashboard{}, err
	}
	defer func() { _ = tx.Rollback() }()

	id := uuid.NewString()
	if f.IsDefault {
		if err := clearDefault(ctx, tx, userID, id); err != nil {
			return Dashboard{}, err
		}
	}
	now := db.FormatTime(s.Now())
	d, err := scan(tx.QueryRowContext(ctx,
		`INSERT INTO dashboards (id, user_id, slug, title, is_default, is_public,
		   desktop_columns, mobile_columns, mobile_breakpoint, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING `+columns,
		id, userID, slug, title, f.IsDefault, f.IsPublic,
		cols.DesktopColumns, cols.MobileColumns, cols.MobileBreakpoint, now, now))
	if db.IsUniqueViolation(err) {
		return Dashboard{}, ErrSlugTaken
	}
	if err != nil {
		return Dashboard{}, err
	}
	return d, tx.Commit()
}

// Update changes the given fields of one of the user's dashboards. Making it
// the default removes the flag from the previous default.
func (s *Store) Update(ctx context.Context, userID, id string, p Patch) (Dashboard, error) {
	sets := []string{"updated_at = ?"}
	args := []any{db.FormatTime(s.Now())}
	if p.Title != nil {
		title, err := normalizeTitle(*p.Title)
		if err != nil {
			return Dashboard{}, err
		}
		sets = append(sets, "title = ?")
		args = append(args, title)
	}
	if p.Slug != nil {
		slug, err := slugs.Normalize(*p.Slug)
		if err != nil {
			return Dashboard{}, ErrInvalidSlug
		}
		sets = append(sets, "slug = ?")
		args = append(args, slug)
	}
	if p.IsDefault != nil {
		sets = append(sets, "is_default = ?")
		args = append(args, *p.IsDefault)
	}
	if p.IsPublic != nil {
		sets = append(sets, "is_public = ?")
		args = append(args, *p.IsPublic)
	}
	// Each value is range-checked here; mobile <= desktop is checked on the
	// result, because a patch may change only one of the two.
	if p.DesktopColumns != nil {
		if !inRange(*p.DesktopColumns) {
			return Dashboard{}, ErrInvalidColumns
		}
		sets = append(sets, "desktop_columns = ?")
		args = append(args, *p.DesktopColumns)
	}
	if p.MobileColumns != nil {
		if !inRange(*p.MobileColumns) {
			return Dashboard{}, ErrInvalidColumns
		}
		sets = append(sets, "mobile_columns = ?")
		args = append(args, *p.MobileColumns)
	}
	if p.MobileBreakpoint != nil {
		if !slices.Contains(Breakpoints, *p.MobileBreakpoint) {
			return Dashboard{}, ErrInvalidColumns
		}
		sets = append(sets, "mobile_breakpoint = ?")
		args = append(args, *p.MobileBreakpoint)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Dashboard{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if p.IsDefault != nil && *p.IsDefault {
		if err := clearDefault(ctx, tx, userID, id); err != nil {
			return Dashboard{}, err
		}
	}
	args = append(args, id, userID)
	d, err := scan(tx.QueryRowContext(ctx,
		`UPDATE dashboards SET `+strings.Join(sets, ", ")+` WHERE id = ? AND user_id = ? RETURNING `+columns, args...))
	if db.IsUniqueViolation(err) {
		return Dashboard{}, ErrSlugTaken
	}
	if err != nil {
		return Dashboard{}, err
	}
	if !d.Columns.valid() {
		return Dashboard{}, ErrInvalidColumns
	}
	return d, tx.Commit()
}

// Delete removes one of the user's dashboards. Deleting the default leaves
// the user without one.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM dashboards WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Lookup resolves /<userSlug>/<dashboardSlug> for viewerID ("" when signed
// out), ignoring case. An unknown user, an unknown dashboard and another
// user's private dashboard all return ErrNotFound, so the response never
// reveals that a private dashboard exists. So do the public dashboards of
// another user whose account is scheduled for deletion.
func (s *Store) Lookup(ctx context.Context, userSlug, dashboardSlug, viewerID string) (View, error) {
	var ownerID, dashboardID string
	var v View
	var public, isDefault, deleting bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT u.id, u.slug, u.delete_after IS NOT NULL, d.id, d.slug, d.title, d.is_public, d.is_default,
		   d.desktop_columns, d.mobile_columns, d.mobile_breakpoint, d.layout_version
		 FROM users u JOIN dashboards d ON d.user_id = u.id
		 WHERE u.slug = ? AND d.slug = ?`,
		strings.ToLower(userSlug), strings.ToLower(dashboardSlug),
	).Scan(&ownerID, &v.OwnerSlug, &deleting, &dashboardID, &v.Dashboard.Slug, &v.Dashboard.Title, &public, &isDefault,
		&v.Columns.DesktopColumns, &v.Columns.MobileColumns, &v.Columns.MobileBreakpoint, &v.LayoutVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, err
	}
	v.IsOwner = viewerID != "" && viewerID == ownerID
	if !v.IsOwner && (!public || deleting) {
		return View{}, ErrNotFound
	}
	if v.IsOwner {
		v.ID = dashboardID
		v.IsDefault, v.IsPublic = &isDefault, &public
	}
	sections, err := loadWidgets(ctx, s.DB, dashboardID)
	if err != nil {
		return View{}, err
	}
	if v.Sections, err = s.resolve(ctx, Viewer{OwnerID: ownerID, IsOwner: v.IsOwner}, sections); err != nil {
		return View{}, err
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT slug, title FROM dashboards WHERE user_id = ? AND (? OR is_public = 1)
		 ORDER BY title COLLATE NOCASE, slug`, ownerID, v.IsOwner)
	if err != nil {
		return View{}, err
	}
	defer rows.Close()
	v.Dashboards = []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Slug, &e.Title); err != nil {
			return View{}, err
		}
		v.Dashboards = append(v.Dashboards, e)
	}
	return v, rows.Err()
}

// clearDefault removes the default flag from the user's dashboards except keepID.
func clearDefault(ctx context.Context, q db.Querier, userID, keepID string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE dashboards SET is_default = 0 WHERE user_id = ? AND is_default = 1 AND id <> ?`, userID, keepID)
	return err
}

func normalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > MaxTitleLength {
		return "", ErrInvalidTitle
	}
	return title, nil
}

func inRange(n int) bool { return n >= 1 && n <= MaxColumns }

func (c Columns) valid() bool {
	return inRange(c.DesktopColumns) && inRange(c.MobileColumns) &&
		c.MobileColumns <= c.DesktopColumns && slices.Contains(Breakpoints, c.MobileBreakpoint)
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (Dashboard, error) {
	var d Dashboard
	if err := row.Scan(&d.ID, &d.Slug, &d.Title, &d.IsDefault, &d.IsPublic,
		&d.DesktopColumns, &d.MobileColumns, &d.MobileBreakpoint); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Dashboard{}, ErrNotFound
		}
		return Dashboard{}, err
	}
	return d, nil
}
