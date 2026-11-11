package db

import (
	"context"
	"database/sql"
	"testing"
)

const sharingNow = "2026-03-01T12:00:00.000Z"

func openSharing(t *testing.T) *sql.DB {
	t.Helper()
	database, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := Migrate(context.Background(), database, Migrations); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES
		   ('ann', 'ann@example.org', 'ann', '` + sharingNow + `', '` + sharingNow + `'),
		   ('bob', 'bob@example.org', 'bob', '` + sharingNow + `', '` + sharingNow + `')`,
		`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES
		   ('d1', 'ann', 'one', 'One', '` + sharingNow + `', '` + sharingNow + `'),
		   ('d2', 'ann', 'two', 'Two', '` + sharingNow + `', '` + sharingNow + `')`,
		`INSERT INTO user_groups (id, name, created_at, updated_at) VALUES ('g1', 'Family', '` + sharingNow + `', '` + sharingNow + `')`,
	} {
		if _, err := database.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return database
}

func insertShare(database *sql.DB, id, dashboard, note any, user, group any) error {
	_, err := database.Exec(
		`INSERT INTO shares (id, owner_id, dashboard_id, note_id, user_id, group_id, level, created_at)
		 VALUES (?, 'ann', ?, ?, ?, ?, 'read', ?)`, id, dashboard, note, user, group, sharingNow)
	return err
}

func sharingRows(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestShareNeedsExactlyOneResourceAndPrincipal(t *testing.T) {
	database := openSharing(t)
	if _, err := database.Exec(
		`INSERT INTO notes (id, user_id, title, content, created_at, updated_at) VALUES ('n1', 'ann', 'N', '', ?, ?)`,
		sharingNow, sharingNow); err != nil {
		t.Fatal(err)
	}
	for name, err := range map[string]error{
		"two resources":  insertShare(database, "s1", "d1", "n1", "bob", nil),
		"no resource":    insertShare(database, "s2", nil, nil, "bob", nil),
		"two principals": insertShare(database, "s3", "d1", nil, "bob", "g1"),
		"no principal":   insertShare(database, "s4", "d1", nil, nil, nil),
	} {
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := insertShare(database, "s5", "d1", nil, "bob", nil); err != nil {
		t.Fatalf("valid share: %v", err)
	}
}

func TestSharePerPrincipalIsUnique(t *testing.T) {
	database := openSharing(t)
	for _, s := range []struct {
		id          string
		user, group any
	}{{"s1", "bob", nil}, {"s2", nil, "g1"}} {
		if err := insertShare(database, s.id, "d1", nil, s.user, s.group); err != nil {
			t.Fatal(err)
		}
		if err := insertShare(database, s.id+"x", "d1", nil, s.user, s.group); !IsUniqueViolation(err) {
			t.Errorf("duplicate %s: %v, want a unique violation", s.id, err)
		}
	}
	if err := insertShare(database, "s3", "d2", nil, "bob", nil); err != nil {
		t.Errorf("same person on another dashboard: %v", err)
	}
}

func insertInvitation(database *sql.DB, id, email string, dashboard, group, level, role any) error {
	_, err := database.Exec(
		`INSERT INTO invitations (id, inviter_id, email, locale, dashboard_id, group_id, level, role, created_at, expires_at)
		 VALUES (?, 'ann', ?, 'en', ?, ?, ?, ?, ?, ?)`, id, email, dashboard, group, level, role, sharingNow, sharingNow)
	return err
}

func TestInvitationShape(t *testing.T) {
	database := openSharing(t)
	for name, err := range map[string]error{
		"two targets":    insertInvitation(database, "i1", "a@example.org", "d1", "g1", "read", "member"),
		"no target":      insertInvitation(database, "i2", "a@example.org", nil, nil, "read", nil),
		"group level":    insertInvitation(database, "i3", "a@example.org", nil, "g1", "read", nil),
		"dashboard role": insertInvitation(database, "i4", "a@example.org", "d1", nil, nil, "member"),
	} {
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := insertInvitation(database, "i5", "a@example.org", "d1", nil, "read", nil); err != nil {
		t.Fatal(err)
	}
	if err := insertInvitation(database, "i6", "a@example.org", "d1", nil, "read", nil); !IsUniqueViolation(err) {
		t.Errorf("second open invitation: %v, want a unique violation", err)
	}
	if err := insertInvitation(database, "i7", "a@example.org", nil, "g1", nil, "member"); err != nil {
		t.Errorf("same address for a group: %v", err)
	}
}

func TestDeletingRemovesSharesAndInvitations(t *testing.T) {
	for name, del := range map[string]string{
		"dashboard": `DELETE FROM dashboards WHERE id = 'd1'`,
		"owner":     `DELETE FROM users WHERE id = 'ann'`,
	} {
		t.Run(name, func(t *testing.T) {
			database := openSharing(t)
			if err := insertShare(database, "s1", "d1", nil, "bob", nil); err != nil {
				t.Fatal(err)
			}
			if err := insertInvitation(database, "i1", "a@example.org", "d1", nil, "read", nil); err != nil {
				t.Fatal(err)
			}
			if _, err := database.Exec(del); err != nil {
				t.Fatal(err)
			}
			if n := sharingRows(t, database, "shares"); n != 0 {
				t.Errorf("shares = %d", n)
			}
			if n := sharingRows(t, database, "invitations"); n != 0 {
				t.Errorf("invitations = %d", n)
			}
		})
	}

	t.Run("user with a share", func(t *testing.T) {
		database := openSharing(t)
		if err := insertShare(database, "s1", "d1", nil, "bob", nil); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`DELETE FROM users WHERE id = 'bob'`); err != nil {
			t.Fatal(err)
		}
		if n := sharingRows(t, database, "shares"); n != 0 {
			t.Errorf("shares = %d", n)
		}
	})

	t.Run("group", func(t *testing.T) {
		database := openSharing(t)
		if err := insertShare(database, "s1", "d1", nil, nil, "g1"); err != nil {
			t.Fatal(err)
		}
		if err := insertInvitation(database, "i1", "a@example.org", nil, "g1", nil, "member"); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`INSERT INTO user_group_members VALUES ('g1', 'bob', 'admin', ?)`, sharingNow); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`DELETE FROM user_groups WHERE id = 'g1'`); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"shares", "invitations", "user_group_members"} {
			if n := sharingRows(t, database, table); n != 0 {
				t.Errorf("%s = %d", table, n)
			}
		}
	})
}
