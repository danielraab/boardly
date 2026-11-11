package shares

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"boardly/internal/dbtest"
)

const now = "2026-03-01T12:00:00.000Z"

var ctx = context.Background()

func setup(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	database := dbtest.Open(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := database.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"ann", "bob", "carol"} {
		exec(`INSERT INTO users (id, email, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, id+"@example.org", id, id, now, now)
	}
	for _, d := range [][3]string{{"work", "ann", "Work"}, {"apple", "ann", "apple"}, {"bobs", "bob", "Bobs"}} {
		exec(`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			d[0], d[1], d[0], d[2], now, now)
	}
	exec(`INSERT INTO user_groups (id, name, created_at, updated_at) VALUES ('family', 'Family', ?, ?), ('zeta', 'Zeta', ?, ?)`, now, now, now, now)
	exec(`INSERT INTO user_group_members VALUES ('family', 'ann', 'admin', ?), ('family', 'bob', 'member', ?), ('zeta', 'carol', 'admin', ?)`, now, now, now)
	s := NewStore(database)
	s.Now = func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }
	return s, database
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want %v", got, want)
	}
}

func direct(t *testing.T, database *sql.DB, id, owner, dashboard, user string) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO shares (id, owner_id, dashboard_id, user_id, level, created_at) VALUES (?, ?, ?, ?, 'read', ?)`,
		id, owner, dashboard, user, now); err != nil {
		t.Fatal(err)
	}
}

func TestShareWithGroupErrors(t *testing.T) {
	s, _ := setup(t)
	_, err := s.ShareWithGroup(ctx, "bob", "work", "family", LevelRead)
	wantErr(t, err, ErrNotFound)
	_, err = s.ShareWithGroup(ctx, "ann", "work", "family", LevelEdit)
	wantErr(t, err, ErrInvalidLevel)
	_, err = s.ShareWithGroup(ctx, "ann", "work", "zeta", LevelRead) // not a member
	wantErr(t, err, ErrGroupNotFound)
	_, err = s.ShareWithGroup(ctx, "ann", "work", "nope", LevelRead)
	wantErr(t, err, ErrGroupNotFound)

	sh, err := s.ShareWithGroup(ctx, "ann", "work", "family", LevelRead)
	if err != nil || sh.Group == nil || sh.Group.Name != "Family" || sh.Level != LevelRead {
		t.Fatalf("share %+v %v", sh, err)
	}
	_, err = s.ShareWithGroup(ctx, "ann", "work", "family", LevelRead)
	wantErr(t, err, ErrAlreadyShared)
}

func TestListSortsPeopleBeforeGroups(t *testing.T) {
	s, database := setup(t)
	if _, err := s.ShareWithGroup(ctx, "ann", "work", "family", LevelRead); err != nil {
		t.Fatal(err)
	}
	direct(t, database, "s1", "ann", "work", "carol")
	direct(t, database, "s2", "ann", "work", "bob")

	list, err := s.List(ctx, "ann", "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].User.Slug != "bob" || list[1].User.Slug != "carol" || list[2].Group.Name != "Family" {
		t.Fatalf("list %+v", list)
	}
	_, err = s.List(ctx, "bob", "work")
	wantErr(t, err, ErrNotFound)
}

func TestRemove(t *testing.T) {
	s, database := setup(t)
	direct(t, database, "s1", "ann", "work", "bob")
	wantErr(t, s.Remove(ctx, "bob", "work", "s1"), ErrNotFound)
	wantErr(t, s.Remove(ctx, "ann", "apple", "s1"), ErrNotFound)
	wantErr(t, s.Remove(ctx, "ann", "work", "nope"), ErrNotFound)
	if err := s.Remove(ctx, "ann", "work", "s1"); err != nil {
		t.Fatal(err)
	}
	if list, err := s.List(ctx, "ann", "work"); err != nil || len(list) != 0 {
		t.Fatalf("list %+v %v", list, err)
	}
}

func TestSharedWithMe(t *testing.T) {
	s, database := setup(t)
	direct(t, database, "s1", "ann", "work", "bob") // direct
	for _, d := range []string{"work", "apple"} {   // group, and combined for work
		if _, err := s.ShareWithGroup(ctx, "ann", d, "family", LevelRead); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.ShareWithGroup(ctx, "bob", "bobs", "family", LevelRead); err != nil {
		t.Fatal(err)
	}

	list, err := s.SharedWithMe(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Title != "apple" || list[1].Title != "Work" || list[1].OwnerSlug != "ann" || list[1].OwnerName != "ann" || list[1].Level != LevelRead {
		t.Fatalf("bob sees %+v", list)
	}

	// Ann's own dashboards shared with Ann's group aren't listed, Bob's is.
	list, err = s.SharedWithMe(ctx, "ann")
	if err != nil || len(list) != 1 || list[0].Title != "Bobs" {
		t.Fatalf("ann sees %+v %v", list, err)
	}

	if list, err := s.SharedWithMe(ctx, "carol"); err != nil || len(list) != 0 {
		t.Fatalf("carol sees %+v %v", list, err)
	}

	if _, err := database.Exec(`UPDATE users SET delete_after = ? WHERE id = 'ann'`, now); err != nil {
		t.Fatal(err)
	}
	if list, err := s.SharedWithMe(ctx, "bob"); err != nil || len(list) != 0 {
		t.Fatalf("owner scheduled for deletion: %+v %v", list, err)
	}
}
