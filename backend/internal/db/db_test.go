package db

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func openTemp(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "nested", "famodash.db")
}

func TestOpenCreatesDirectoryAndSetsPragmas(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d, err %v", fk, err)
	}
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, err %v", mode, err)
	}
}

var fixture = fstest.MapFS{
	"0001_init.sql":   {Data: []byte("-- nothing\n")},
	"0002_items.sql":  {Data: []byte("CREATE TABLE items (id INTEGER PRIMARY KEY);")},
	"0003_insert.sql": {Data: []byte("INSERT INTO items (id) VALUES (1);")},
}

func count(t *testing.T, path, query string) int {
	t.Helper()
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func migrate(t *testing.T, path string) {
	t.Helper()
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, fixture); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestMigrateFreshAndRerun(t *testing.T) {
	path := openTemp(t)

	migrate(t, path)
	if n := count(t, path, `SELECT COUNT(*) FROM schema_migrations`); n != 3 {
		t.Fatalf("applied = %d, want 3", n)
	}

	// A second run must not apply anything again (the INSERT would fail on the primary key).
	migrate(t, path)
	if n := count(t, path, `SELECT COUNT(*) FROM items`); n != 1 {
		t.Fatalf("items = %d, want 1", n)
	}
}

func TestMigrateRejectsBadNames(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	bad := fstest.MapFS{"init.sql": {Data: []byte("")}}
	if err := Migrate(context.Background(), db, bad); err == nil {
		t.Fatal("expected an error for a migration without version prefix")
	}
}

func TestShippedMigrationsApply(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestDeletingUserCascadesToSessions(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const ts = "2026-01-01T00:00:00Z"
	if _, err := db.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id_hash, user_id, auth_method, created_at, last_used_at, expires_at, user_agent)
		VALUES (x'01', 'u1', 'magic_link', ?, ?, ?, '')`, ts, ts, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("sessions = %d, err %v; want 0 after deleting the user", n, err)
	}
}

// migrationsUpTo returns the shipped migrations with a version of at most maxVersion.
func migrationsUpTo(t *testing.T, maxVersion string) fstest.MapFS {
	t.Helper()
	entries, err := fs.ReadDir(Migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	out := fstest.MapFS{}
	for _, e := range entries {
		if e.Name()[:4] > maxVersion {
			continue
		}
		data, err := fs.ReadFile(Migrations, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = &fstest.MapFile{Data: data}
	}
	return out
}

func TestPasskeyMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0003")); err != nil {
		t.Fatalf("Migrate to 0003: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	if _, err := db.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id_hash, user_id, auth_method, created_at, last_used_at, expires_at, user_agent)
		VALUES (x'01', 'u1', 'magic_link', ?, ?, ?, '')`, ts, ts, ts); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var until *string
	if err := db.QueryRow(`SELECT passkey_enrollment_until FROM sessions`).Scan(&until); err != nil || until != nil {
		t.Fatalf("passkey_enrollment_until = %v, err %v; want NULL for existing sessions", until, err)
	}

	if _, err := db.Exec(`INSERT INTO passkeys (id, user_id, credential_id, public_key, aaguid, sign_count, transports, backup_eligible, backup_state, created_at)
		VALUES ('p1', 'u1', x'aa', x'bb', zeroblob(16), 0, '[]', 1, 1, ?)`, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO webauthn_challenges (id_hash, kind, user_id, session_id, data, expires_at)
		VALUES (x'02', 'register', 'u1', x'01', '{}', ?)`, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"passkeys", "webauthn_challenges"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s = %d, err %v; want 0 after deleting the user", table, n, err)
		}
	}
}

func TestFavouritesConstraints(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	exec := func(query string, args ...any) error {
		_, err := db.Exec(query, args...)
		return err
	}
	mustExec := func(query string, args ...any) {
		t.Helper()
		if err := exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	node := `INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, '` + ts + `', '` + ts + `')`

	mustExec(`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?), ('u2', 'b@example.org', ?, ?)`, ts, ts, ts, ts)
	mustExec(node, "r1", "u1", nil, "folder", nil, nil)
	mustExec(node, "r2", "u2", nil, "folder", nil, nil)
	mustExec(node, "f1", "u1", "r1", "folder", "Work", nil)
	mustExec(node, "l1", "u1", "f1", "link", "Docs", "https://example.org")

	for name, args := range map[string][]any{
		"link without parent":         {"x1", "u1", nil, "link", nil, "https://example.org"},
		"folder with url":             {"x2", "u1", "r1", "folder", "F", "https://example.org"},
		"link without url":            {"x3", "u1", "r1", "link", "L", nil},
		"second root":                 {"x4", "u1", nil, "folder", nil, nil},
		"root with title":             {"x5", "u2", nil, "folder", "Root", nil},
		"node in another user's tree": {"x6", "u2", "f1", "folder", "Foreign", nil},
		"unknown kind":                {"x7", "u1", "r1", "note", "N", nil},
	} {
		if err := exec(node, args...); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", name)
		}
	}

	mustExec(`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES ('t1', 'u1', 'Reading', 'reading', ?)`, ts)
	if err := exec(`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES ('t2', 'u1', 'reading', 'reading', ?)`, ts); err == nil {
		t.Error("duplicate tag key succeeded, want a constraint error")
	}
	mustExec(`INSERT INTO favourite_node_tags (node_id, tag_id) VALUES ('l1', 't1')`)

	// Deleting a folder removes its subtree and the subtree's tag links.
	mustExec(`DELETE FROM favourite_nodes WHERE id = 'f1'`)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM favourite_node_tags`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("favourite_node_tags = %d, err %v; want 0 after deleting the folder", n, err)
	}

	mustExec(`DELETE FROM users WHERE id = 'u1'`)
	for _, table := range []string{"favourite_nodes WHERE user_id = 'u1'", "tags"} {
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s = %d, err %v; want 0 after deleting the user", table, n, err)
		}
	}
}
