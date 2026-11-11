package groups

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"boardly/internal/db"
	"boardly/internal/dbtest"
)

const now = "2026-03-01T12:00:00.000Z"

var ctx = context.Background()

func setup(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	database := dbtest.Open(t)
	for _, id := range []string{"ann", "bob", "carol"} {
		if _, err := database.Exec(
			`INSERT INTO users (id, email, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, id+"@example.org", id, id, now, now); err != nil {
			t.Fatal(err)
		}
	}
	s := NewStore(database)
	c := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return c }
	return s, database
}

func add(t *testing.T, database *sql.DB, group, user, role string, joined time.Time) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO user_group_members VALUES (?, ?, ?, ?)`, group, user, role, db.FormatTime(joined)); err != nil {
		t.Fatal(err)
	}
}

func create(t *testing.T, s *Store, user, name string) Group {
	t.Helper()
	g, err := s.Create(ctx, user, name)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want %v", got, want)
	}
}

func TestCreateAndList(t *testing.T) {
	s, _ := setup(t)
	g := create(t, s, "ann", "  Family ")
	if g.Name != "Family" || g.Role != RoleAdmin || g.MemberCount != 1 {
		t.Fatalf("created %+v", g)
	}
	create(t, s, "ann", "archive")
	create(t, s, "bob", "Family") // names don't have to be unique

	list, err := s.List(ctx, "ann")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "archive" || list[1].Name != "Family" {
		t.Fatalf("list %+v", list)
	}

	for _, name := range []string{"", "   ", strings.Repeat("x", 101)} {
		_, err := s.Create(ctx, "ann", name)
		wantErr(t, err, ErrInvalidName)
	}
}

func TestVisibleOnlyToMembers(t *testing.T) {
	s, _ := setup(t)
	g := create(t, s, "ann", "Family")
	_, err := s.Get(ctx, "bob", g.ID)
	wantErr(t, err, ErrNotFound)
	_, err = s.Rename(ctx, "bob", g.ID, "X")
	wantErr(t, err, ErrNotFound)
	wantErr(t, s.Delete(ctx, "bob", g.ID), ErrNotFound)
	wantErr(t, s.SetRole(ctx, "bob", g.ID, "ann", RoleMember), ErrNotFound)
	wantErr(t, s.RemoveMember(ctx, "bob", g.ID, "ann"), ErrNotFound)
}

func TestMembersSortedWithoutEmail(t *testing.T) {
	s, database := setup(t)
	g := create(t, s, "bob", "Family")
	add(t, database, g.ID, "ann", RoleMember, time.Now())
	d, err := s.Get(ctx, "ann", g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Role != RoleMember || d.MemberCount != 2 || len(d.Members) != 2 || d.Members[0].ID != "ann" || d.Members[1].Role != RoleAdmin {
		t.Fatalf("details %+v", d)
	}
}

func TestAdminActions(t *testing.T) {
	s, database := setup(t)
	g := create(t, s, "ann", "Family")
	add(t, database, g.ID, "bob", RoleMember, time.Now())
	add(t, database, g.ID, "carol", RoleMember, time.Now())

	_, err := s.Rename(ctx, "bob", g.ID, "X")
	wantErr(t, err, ErrForbidden)
	wantErr(t, s.Delete(ctx, "bob", g.ID), ErrForbidden)
	wantErr(t, s.SetRole(ctx, "bob", g.ID, "carol", RoleAdmin), ErrForbidden)
	wantErr(t, s.RemoveMember(ctx, "bob", g.ID, "carol"), ErrForbidden)
	wantErr(t, s.SetRole(ctx, "ann", g.ID, "bob", "owner"), ErrInvalidRole)
	wantErr(t, s.SetRole(ctx, "ann", g.ID, "nobody", RoleAdmin), ErrNotFound)
	wantErr(t, s.RemoveMember(ctx, "ann", g.ID, "nobody"), ErrNotFound)
	_, err = s.Rename(ctx, "ann", g.ID, " ")
	wantErr(t, err, ErrInvalidName)

	if err := s.SetRole(ctx, "ann", g.ID, "bob", RoleAdmin); err != nil {
		t.Fatal(err)
	}
	renamed, err := s.Rename(ctx, "bob", g.ID, "Clan")
	if err != nil || renamed.Name != "Clan" || renamed.MemberCount != 3 {
		t.Fatalf("rename %+v %v", renamed, err)
	}
	if err := s.RemoveMember(ctx, "bob", g.ID, "carol"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "carol", g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("carol still sees the group: %v", err)
	}
}

func TestLastAdmin(t *testing.T) {
	s, database := setup(t)
	g := create(t, s, "ann", "Family")
	add(t, database, g.ID, "bob", RoleMember, time.Now())

	wantErr(t, s.RemoveMember(ctx, "ann", g.ID, "ann"), ErrLastAdmin)
	wantErr(t, s.SetRole(ctx, "ann", g.ID, "ann", RoleMember), ErrLastAdmin)

	if err := s.SetRole(ctx, "ann", g.ID, "bob", RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveMember(ctx, "ann", g.ID, "ann"); err != nil {
		t.Fatalf("second admin leaving: %v", err)
	}
	d, err := s.Get(ctx, "bob", g.ID)
	if err != nil || len(d.Members) != 1 || d.Role != RoleAdmin {
		t.Fatalf("after leaving %+v %v", d, err)
	}

	// The last admin can still delete the group.
	if err := s.Delete(ctx, "bob", g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "bob", g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted group: %v", err)
	}
}

func TestLeavingRemovesOwnSharesOnly(t *testing.T) {
	s, database := setup(t)
	g := create(t, s, "ann", "Family")
	add(t, database, g.ID, "bob", RoleMember, time.Now())
	for _, owner := range []string{"ann", "bob"} {
		if _, err := database.Exec(
			`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES (?, ?, 'd', 'D', ?, ?)`,
			"dash-"+owner, owner, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(
			`INSERT INTO shares (id, owner_id, dashboard_id, group_id, level, created_at) VALUES (?, ?, ?, ?, 'read', ?)`,
			"share-"+owner, owner, "dash-"+owner, g.ID, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoveMember(ctx, "bob", g.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	var owners []string
	rows, err := database.Query(`SELECT owner_id FROM shares`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var o string
		if err := rows.Scan(&o); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, o)
	}
	if len(owners) != 1 || owners[0] != "ann" {
		t.Fatalf("shares of %v, want only ann's", owners)
	}
}

func TestPrepareAccountDeletion(t *testing.T) {
	s, database := setup(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	solo := create(t, s, "ann", "Solo")
	handover := create(t, s, "ann", "Handover")
	add(t, database, handover.ID, "carol", RoleMember, base.Add(2*time.Hour))
	add(t, database, handover.ID, "bob", RoleMember, base.Add(time.Hour))
	tie := create(t, s, "ann", "Tie")
	add(t, database, tie.ID, "carol", RoleMember, base)
	add(t, database, tie.ID, "bob", RoleMember, base)
	coAdmin := create(t, s, "ann", "CoAdmin")
	add(t, database, coAdmin.ID, "bob", RoleAdmin, base)
	add(t, database, coAdmin.ID, "carol", RoleMember, base.Add(-time.Hour))
	member := create(t, s, "bob", "Member")
	add(t, database, member.ID, "ann", RoleMember, base)

	if err := PrepareAccountDeletion(ctx, database, "ann"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DELETE FROM users WHERE id = 'ann'`); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Get(ctx, "ann", solo.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("solo group: %v", err)
	}
	var n int
	if err := database.QueryRow(`SELECT COUNT(*) FROM user_groups WHERE id = ?`, solo.ID).Scan(&n); err != nil || n != 0 {
		t.Errorf("solo group rows = %d, %v", n, err)
	}
	role := func(group, user string) string {
		t.Helper()
		var r string
		if err := database.QueryRow(`SELECT role FROM user_group_members WHERE group_id = ? AND user_id = ?`, group, user).Scan(&r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	for _, c := range []struct{ name, group, user, want string }{
		{"earliest joined takes over", handover.ID, "bob", RoleAdmin},
		{"later member stays", handover.ID, "carol", RoleMember},
		{"tie goes to the lower user ID", tie.ID, "bob", RoleAdmin},
		{"tie loser stays", tie.ID, "carol", RoleMember},
		{"another admin exists", coAdmin.ID, "carol", RoleMember},
		{"ann wasn't admin", member.ID, "bob", RoleAdmin},
	} {
		if got := role(c.group, c.user); got != c.want {
			t.Errorf("%s: role = %s, want %s", c.name, got, c.want)
		}
	}
}
