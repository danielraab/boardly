// Package groups stores groups of users and their memberships. A group has no
// single owner: its group admins manage it, and it always keeps at least one.
package groups

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// Roles of a membership.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// MaxNameLength is the longest group name, in characters, after trimming.
const MaxNameLength = 100

// Errors returned by the store.
var (
	// ErrNotFound means the group or member doesn't exist, or the acting user
	// isn't a member of the group.
	ErrNotFound = errors.New("group not found")
	// ErrForbidden means a group member tried an action for group admins.
	ErrForbidden   = errors.New("group admins only")
	ErrInvalidName = errors.New("invalid group name")
	ErrInvalidRole = errors.New("invalid role")
	// ErrLastAdmin means the change would leave the group without a group admin.
	ErrLastAdmin = errors.New("last group admin")
)

// Group is a group as one of its members sees it.
type Group struct {
	ID string `json:"id"`
	// Name is the group's name.
	Name string `json:"name"`
	// Role is the requesting user's own role.
	Role        string `json:"role"`
	MemberCount int    `json:"memberCount"`
}

// Member is a member of a group. It never contains an email address.
type Member struct {
	ID   string  `json:"id"`
	Slug string  `json:"slug"`
	Name *string `json:"name"`
	Role string  `json:"role"`
}

// Details is a group with its members.
type Details struct {
	Group
	Members []Member `json:"members"`
}

// Store reads and writes groups.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

// ValidRole reports whether role is a role of a membership.
func ValidRole(role string) bool { return role == RoleAdmin || role == RoleMember }

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength {
		return "", ErrInvalidName
	}
	return name, nil
}

// Create adds a group with userID as its first group admin.
func (s *Store) Create(ctx context.Context, userID, name string) (Group, error) {
	name, err := normalizeName(name)
	if err != nil {
		return Group{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, err
	}
	defer func() { _ = tx.Rollback() }()

	id, now := uuid.NewString(), db.FormatTime(s.Now())
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO user_groups (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, id, name, now, now); err != nil {
		return Group{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO user_group_members (group_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		id, userID, RoleAdmin, now); err != nil {
		return Group{}, err
	}
	if err := tx.Commit(); err != nil {
		return Group{}, err
	}
	return Group{ID: id, Name: name, Role: RoleAdmin, MemberCount: 1}, nil
}

const groupColumns = `g.id, g.name, m.role, (SELECT COUNT(*) FROM user_group_members c WHERE c.group_id = g.id)`

// List returns the groups the user belongs to, sorted by name, ignoring case.
func (s *Store) List(ctx context.Context, userID string) ([]Group, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+groupColumns+` FROM user_groups g JOIN user_group_members m ON m.group_id = g.id
		 WHERE m.user_id = ? ORDER BY g.name COLLATE NOCASE, g.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Group{}
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Role, &g.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) group(ctx context.Context, q db.Querier, userID, id string) (Group, error) {
	var g Group
	err := q.QueryRowContext(ctx,
		`SELECT `+groupColumns+` FROM user_groups g JOIN user_group_members m ON m.group_id = g.id
		 WHERE g.id = ? AND m.user_id = ?`, id, userID,
	).Scan(&g.ID, &g.Name, &g.Role, &g.MemberCount)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	return g, err
}

// Get returns a group the user belongs to, with its members sorted by name,
// ignoring case.
func (s *Store) Get(ctx context.Context, userID, id string) (Details, error) {
	g, err := s.group(ctx, s.DB, userID, id)
	if err != nil {
		return Details{}, err
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT u.id, u.slug, u.name, m.role FROM user_group_members m JOIN users u ON u.id = m.user_id
		 WHERE m.group_id = ? ORDER BY COALESCE(u.name, u.slug) COLLATE NOCASE, u.id`, id)
	if err != nil {
		return Details{}, err
	}
	defer rows.Close()
	d := Details{Group: g, Members: []Member{}}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Slug, &m.Name, &m.Role); err != nil {
			return Details{}, err
		}
		d.Members = append(d.Members, m)
	}
	return d, rows.Err()
}

// RequireAdmin returns ErrNotFound unless userID is a member of the group and
// ErrForbidden unless they are one of its group admins. q may be a transaction.
func RequireAdmin(ctx context.Context, q db.Querier, groupID, userID string) error {
	role, err := roleOf(ctx, q, groupID, userID)
	if err != nil {
		return err
	}
	if role != RoleAdmin {
		return ErrForbidden
	}
	return nil
}

// roleOf returns the member's role, or ErrNotFound.
func roleOf(ctx context.Context, q db.Querier, groupID, userID string) (string, error) {
	var role string
	err := q.QueryRowContext(ctx,
		`SELECT role FROM user_group_members WHERE group_id = ? AND user_id = ?`, groupID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

// Rename changes the name of a group. Group admins only.
func (s *Store) Rename(ctx context.Context, userID, id, name string) (Group, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := RequireAdmin(ctx, tx, id, userID); err != nil {
		return Group{}, err
	}
	if name, err = normalizeName(name); err != nil {
		return Group{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE user_groups SET name = ?, updated_at = ? WHERE id = ?`, name, db.FormatTime(s.Now()), id); err != nil {
		return Group{}, err
	}
	g, err := s.group(ctx, tx, userID, id)
	if err != nil {
		return Group{}, err
	}
	return g, tx.Commit()
}

// Delete removes a group with its memberships, shares and invitations. Group
// admins only; the last group admin can do this as well.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := RequireAdmin(ctx, tx, id, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_groups WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func adminCount(ctx context.Context, q db.Querier, groupID string) (int, error) {
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_group_members WHERE group_id = ? AND role = ?`, groupID, RoleAdmin).Scan(&n)
	return n, err
}

// SetRole changes a member's role. Group admins only. The last group admin
// can't be given the role group member.
func (s *Store) SetRole(ctx context.Context, userID, id, memberID, role string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := RequireAdmin(ctx, tx, id, userID); err != nil {
		return err
	}
	if !ValidRole(role) {
		return ErrInvalidRole
	}
	current, err := roleOf(ctx, tx, id, memberID)
	if err != nil {
		return err
	}
	if current == RoleAdmin && role != RoleAdmin {
		n, err := adminCount(ctx, tx, id)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE user_group_members SET role = ? WHERE group_id = ? AND user_id = ?`, role, id, memberID); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveMember removes a member from a group, and with them the shares they
// made with the group. Group admins can remove anyone, and everyone can remove
// themselves to leave. The last group admin can't leave or be removed.
func (s *Store) RemoveMember(ctx context.Context, userID, id, memberID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	own, err := roleOf(ctx, tx, id, userID)
	if err != nil {
		return err
	}
	if userID != memberID && own != RoleAdmin {
		return ErrForbidden
	}
	role, err := roleOf(ctx, tx, id, memberID)
	if err != nil {
		return err
	}
	if role == RoleAdmin {
		n, err := adminCount(ctx, tx, id)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM user_group_members WHERE group_id = ? AND user_id = ?`, id, memberID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM shares WHERE group_id = ? AND owner_id = ?`, id, memberID); err != nil {
		return err
	}
	return tx.Commit()
}

// PrepareAccountDeletion hands over the groups of an account that is about to
// be deleted, in the caller's transaction: groups where it is the only member
// are deleted, and where it is the only group admin, the member who joined
// first (ties broken by user ID) becomes a group admin.
func PrepareAccountDeletion(ctx context.Context, tx db.Querier, userID string) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM user_groups WHERE id IN (
		   SELECT m.group_id FROM user_group_members m
		   WHERE m.user_id = ?
		     AND (SELECT COUNT(*) FROM user_group_members c WHERE c.group_id = m.group_id) = 1)`, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE user_group_members SET role = 'admin'
		 WHERE (group_id, user_id) IN (
		   SELECT o.group_id, (
		     SELECT n.user_id FROM user_group_members n
		     WHERE n.group_id = o.group_id AND n.user_id <> ?
		     ORDER BY n.joined_at, n.user_id LIMIT 1)
		   FROM user_group_members o
		   WHERE o.user_id = ? AND o.role = 'admin'
		     AND (SELECT COUNT(*) FROM user_group_members a WHERE a.group_id = o.group_id AND a.role = 'admin') = 1)`,
		userID, userID)
	return err
}
