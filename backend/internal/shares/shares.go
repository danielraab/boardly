// Package shares stores who a resource is shared with and answers what is
// shared with a user. For now only dashboards can be shared. The stores of the
// shared resources stay scoped to their owner.
package shares

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// Levels of access. Only LevelRead can be given for now.
const (
	LevelRead = "read"
	LevelEdit = "edit"
)

// Errors returned by the store.
var (
	// ErrNotFound means the dashboard isn't the user's, or the share doesn't exist.
	ErrNotFound = errors.New("share not found")
	// ErrInvalidLevel is returned for any level other than read.
	ErrInvalidLevel = errors.New("invalid share level")
	// ErrGroupNotFound means the group doesn't exist or the owner isn't in it.
	ErrGroupNotFound = errors.New("group not found")
	// ErrAlreadyShared means the dashboard is already shared with the group.
	ErrAlreadyShared = errors.New("already shared")
)

// User is a person with access. It never contains an email address.
type User struct {
	ID   string  `json:"id"`
	Slug string  `json:"slug"`
	Name *string `json:"name"`
}

// Group is a group with access.
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Share is an entry of a dashboard's access list: User or Group is set.
type Share struct {
	ID    string `json:"id"`
	Level string `json:"level"`
	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

// SharedDashboard is a dashboard shared with a user.
type SharedDashboard struct {
	OwnerSlug string `json:"ownerSlug"`
	// OwnerName is the owner's name, or their slug when they have none.
	OwnerName string `json:"ownerName"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Level     string `json:"level"`
}

// Store reads and writes shares.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

func requireOwner(ctx context.Context, q db.Querier, ownerID, dashboardID string) error {
	var one int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM dashboards WHERE id = ? AND user_id = ?`, dashboardID, ownerID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// List returns the shares of one of the owner's dashboards: people before
// groups, each by name, ignoring case.
func (s *Store) List(ctx context.Context, ownerID, dashboardID string) ([]Share, error) {
	if err := requireOwner(ctx, s.DB, ownerID, dashboardID); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT s.id, s.level, u.id, u.slug, u.name, g.id, g.name
		 FROM shares s
		 LEFT JOIN users u ON u.id = s.user_id
		 LEFT JOIN user_groups g ON g.id = s.group_id
		 WHERE s.dashboard_id = ?
		 ORDER BY s.group_id IS NOT NULL, COALESCE(u.name, u.slug, g.name) COLLATE NOCASE, s.id`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Share{}
	for rows.Next() {
		var sh Share
		var userID, userSlug, groupID, groupName sql.NullString
		var userName *string
		if err := rows.Scan(&sh.ID, &sh.Level, &userID, &userSlug, &userName, &groupID, &groupName); err != nil {
			return nil, err
		}
		if userID.Valid {
			sh.User = &User{ID: userID.String, Slug: userSlug.String, Name: userName}
		}
		if groupID.Valid {
			sh.Group = &Group{ID: groupID.String, Name: groupName.String}
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

// ShareWithGroup shares one of the owner's dashboards with a group they
// belong to.
func (s *Store) ShareWithGroup(ctx context.Context, ownerID, dashboardID, groupID, level string) (Share, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Share{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := requireOwner(ctx, tx, ownerID, dashboardID); err != nil {
		return Share{}, err
	}
	if level != LevelRead {
		return Share{}, ErrInvalidLevel
	}
	var name string
	err = tx.QueryRowContext(ctx,
		`SELECT g.name FROM user_groups g JOIN user_group_members m ON m.group_id = g.id
		 WHERE g.id = ? AND m.user_id = ?`, groupID, ownerID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return Share{}, ErrGroupNotFound
	}
	if err != nil {
		return Share{}, err
	}
	id := uuid.NewString()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO shares (id, owner_id, dashboard_id, group_id, level, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, ownerID, dashboardID, groupID, level, db.FormatTime(s.Now()))
	if db.IsUniqueViolation(err) {
		return Share{}, ErrAlreadyShared
	}
	if err != nil {
		return Share{}, err
	}
	if err := tx.Commit(); err != nil {
		return Share{}, err
	}
	return Share{ID: id, Level: level, Group: &Group{ID: groupID, Name: name}}, nil
}

// Remove deletes a share of one of the owner's dashboards.
func (s *Store) Remove(ctx context.Context, ownerID, dashboardID, shareID string) error {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM shares WHERE id = ? AND dashboard_id = ? AND owner_id = ?`, shareID, dashboardID, ownerID)
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

// SharedWithMe returns the dashboards shared with the user, directly or
// through a group, each once, sorted by title, ignoring case, then by owner
// slug. It leaves out the user's own dashboards and those of accounts
// scheduled for deletion.
func (s *Store) SharedWithMe(ctx context.Context, userID string) ([]SharedDashboard, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT o.slug, COALESCE(o.name, o.slug), d.slug, d.title,
		   CASE WHEN MAX(sh.level = 'edit') = 1 THEN 'edit' ELSE 'read' END
		 FROM shares sh
		 JOIN dashboards d ON d.id = sh.dashboard_id
		 JOIN users o ON o.id = d.user_id
		 WHERE d.user_id <> ? AND o.delete_after IS NULL
		   AND (sh.user_id = ? OR sh.group_id IN (SELECT group_id FROM user_group_members WHERE user_id = ?))
		 GROUP BY d.id
		 ORDER BY d.title COLLATE NOCASE, o.slug`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SharedDashboard{}
	for rows.Next() {
		var d SharedDashboard
		if err := rows.Scan(&d.OwnerSlug, &d.OwnerName, &d.Slug, &d.Title, &d.Level); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
