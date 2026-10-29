package db

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func openTemp(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "nested", "boardly.db")
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

func TestDashboardsMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0005")); err != nil {
		t.Fatalf("Migrate to 0005: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	if _, err := db.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?), ('u2', 'b@example.org', ?, ?)`, ts, ts, ts, ts); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	rows, err := db.Query(`SELECT slug FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for rows.Next() {
		var slug *string
		if err := rows.Scan(&slug); err != nil {
			t.Fatal(err)
		}
		if slug == nil || len(*slug) != 10 || strings.Trim(*slug, "0123456789abcdef") != "" {
			t.Fatalf("slug = %v, want 10 lowercase hex characters", slug)
		}
		seen[*slug] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 {
		t.Fatalf("slugs = %v, want two distinct slugs", seen)
	}
}

func TestDashboardsConstraints(t *testing.T) {
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
	dash := `INSERT INTO dashboards (id, user_id, slug, title, is_default, is_public, created_at, updated_at)
		VALUES (?, ?, ?, 'T', ?, 0, '` + ts + `', '` + ts + `')`

	mustExec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', ?, ?), ('u2', 'b@example.org', 'bob', ?, ?)`, ts, ts, ts, ts)
	if err := exec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u3', 'c@example.org', 'ann', ?, ?)`, ts, ts); err == nil {
		t.Error("duplicate user slug succeeded, want a constraint error")
	}
	mustExec(dash, "d1", "u1", "home", 1)
	mustExec(dash, "d2", "u1", "work", 0)
	mustExec(dash, "d3", "u2", "home", 1)

	for name, args := range map[string][]any{
		"second default":  {"x1", "u1", "links", 1},
		"duplicate slug":  {"x2", "u1", "home", 0},
		"uppercase slug":  {"x3", "u1", "Links", 0},
		"invalid default": {"x4", "u1", "other", 2},
	} {
		if err := exec(dash, args...); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", name)
		}
	}

	mustExec(`DELETE FROM users WHERE id = 'u1'`)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dashboards WHERE user_id = 'u1'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("dashboards = %d, err %v; want 0 after deleting the user", n, err)
	}
}

func TestDashboardLayoutMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0006")); err != nil {
		t.Fatalf("Migrate to 0006: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	if _, err := db.Exec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES ('d1', 'u1', 'home', 'Home', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var desktop, mobile, breakpoint, version int
	if err := db.QueryRow(`SELECT desktop_columns, mobile_columns, mobile_breakpoint, layout_version FROM dashboards`).
		Scan(&desktop, &mobile, &breakpoint, &version); err != nil {
		t.Fatal(err)
	}
	if desktop != 6 || mobile != 2 || breakpoint != 768 || version != 0 {
		t.Fatalf("columns = %d/%d/%d, version %d; want 6/2/768, version 0", desktop, mobile, breakpoint, version)
	}
}

func TestDashboardWidgetsConstraints(t *testing.T) {
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
	widget := `INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config)
		VALUES (?, ?, ?, ?, 0, ?, ?)`

	mustExec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', ?, ?)`, ts, ts)
	mustExec(`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES ('d1', 'u1', 'home', 'H', ?, ?), ('d2', 'u1', 'work', 'W', ?, ?)`, ts, ts, ts, ts)
	mustExec(widget, "s1", "d1", nil, "section", 6, "{}")
	mustExec(widget, "s2", "d1", "s1", "section", 3, `{"border":true}`)
	mustExec(widget, "s3", "d1", "s2", "section", 2, "{}")
	mustExec(widget, "t1", "d2", nil, "section", 6, "{}")

	for name, args := range map[string][]any{
		"parent in another dashboard": {"x1", "d2", "s1", "section", 6, "{}"},
		"width 0":                     {"x3", "d1", nil, "section", 0, "{}"},
		"width 13":                    {"x4", "d1", nil, "section", 13, "{}"},
		"invalid config":              {"x5", "d1", nil, "section", 6, "{"},
	} {
		if err := exec(widget, args...); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", name)
		}
	}
	if err := exec(`UPDATE dashboards SET mobile_breakpoint = 800 WHERE id = 'd1'`); err == nil {
		t.Error("breakpoint 800 succeeded, want a constraint error")
	}

	// Deleting a section removes its subtree; deleting a dashboard removes the rest.
	mustExec(`DELETE FROM dashboard_widgets WHERE id = 's2'`)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dashboard_widgets WHERE dashboard_id = 'd1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("widgets = %d, err %v; want 1 after deleting the middle section", n, err)
	}
	mustExec(`DELETE FROM dashboards WHERE id = 'd1'`)
	if err := db.QueryRow(`SELECT COUNT(*) FROM dashboard_widgets WHERE dashboard_id = 'd1'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("widgets = %d, err %v; want 0 after deleting the dashboard", n, err)
	}
}

func TestWidgetKindsMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0007")); err != nil {
		t.Fatalf("Migrate to 0007: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	widget := `INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config)
		VALUES (?, 'd1', ?, ?, ?, ?, ?)`
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', ?, ?)`, []any{ts, ts}},
		{`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES ('d1', 'u1', 'home', 'Home', ?, ?)`, []any{ts, ts}},
		{widget, []any{"s1", nil, "section", 0, 6, `{"title":"Work","border":true}`}},
		{widget, []any{"s2", "s1", "section", 1, 3, `{"border":false}`}},
	} {
		if _, err := db.Exec(q.query, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var parent, config string
	var width, position int
	if err := db.QueryRow(`SELECT parent_id, position, width, config FROM dashboard_widgets WHERE id = 's2'`).
		Scan(&parent, &position, &width, &config); err != nil {
		t.Fatal(err)
	}
	if parent != "s1" || position != 1 || width != 3 || config != `{"border":false}` {
		t.Fatalf("s2 = %s/%d/%d/%s after the migration", parent, position, width, config)
	}

	// Any kind is accepted now, and the composite foreign key still holds.
	if _, err := db.Exec(widget, "n1", "s1", "note", 2, 2, "{}"); err != nil {
		t.Fatalf("kind note: %v", err)
	}
	if _, err := db.Exec(widget, "n2", "missing", "note", 3, 2, "{}"); err == nil {
		t.Fatal("unknown parent accepted")
	}
	if _, err := db.Exec(`DELETE FROM dashboard_widgets WHERE id = 's1'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dashboard_widgets`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("widgets = %d, err %v; want 0 after deleting the parent", n, err)
	}
}

func TestImagesConstraints(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	image := `INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, ?, ?, 'image/png', ?, ?)`
	if _, err := db.Exec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]any{{"up", "upload", "u1", 10, ts}, {"fav", "favicon", nil, 10, ts}} {
		if _, err := db.Exec(image, args...); err != nil {
			t.Fatal(err)
		}
	}
	for name, args := range map[string][]any{
		"upload without user": {"x1", "upload", nil, 10, ts},
		"favicon with user":   {"x2", "favicon", "u1", 10, ts},
		"unknown source":      {"x3", "thumbnail", nil, 10, ts},
		"empty":               {"x4", "favicon", nil, 0, ts},
	} {
		if _, err := db.Exec(image, args...); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", name)
		}
	}

	// Deleting the image keeps the origin's row, without an image.
	if _, err := db.Exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://example.org', 'fav', ?)`, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM images WHERE id = 'fav'`); err != nil {
		t.Fatal(err)
	}
	var imageID sql.NullString
	if err := db.QueryRow(`SELECT image_id FROM favicons`).Scan(&imageID); err != nil || imageID.Valid {
		t.Fatalf("image_id = %v, err %v; want NULL", imageID, err)
	}

	// Deleting a user deletes their uploads.
	if _, err := db.Exec(`DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM images`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("images = %d, err %v; want 0", n, err)
	}
}

func TestMagicLinkCodeMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0009")); err != nil {
		t.Fatalf("Migrate to 0009: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	if _, err := db.Exec(`INSERT INTO magic_link_tokens (token_hash, email, locale, created_at, expires_at, request_ip, browser_hash)
		VALUES (x'01', 'a@example.org', 'en', ?, ?, '127.0.0.1', x'02')`, ts, ts); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var code []byte
	var attempts int
	if err := db.QueryRow(`SELECT code_hash, code_attempts FROM magic_link_tokens`).Scan(&code, &attempts); err != nil {
		t.Fatal(err)
	}
	if code != nil || attempts != 0 {
		t.Fatalf("code_hash = %x, code_attempts = %d; want NULL and 0 for existing links", code, attempts)
	}
}

func TestAppTokensConstraints(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	for _, q := range []string{
		`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('u1', 'a@example.org', 'ann', '` + ts + `', '` + ts + `')`,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, created_at, updated_at) VALUES ('root', 'u1', NULL, 'folder', 0, '` + ts + `', '` + ts + `')`,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, created_at, updated_at) VALUES ('f1', 'u1', 'root', 'folder', 0, 'Reading', '` + ts + `', '` + ts + `')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	token := `INSERT INTO app_tokens (id, user_id, name, app, root_id, token_hash, prefix, created_at) VALUES (?, 'u1', 'Phone', ?, 'f1', ?, 'bdly_abcdefg', ?)`
	if _, err := db.Exec(token, "t1", "karakeep", []byte{1}, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(token, "t4", "linkwarden", []byte{4}, ts); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]any{
		"unknown app":    {"t2", "pocket", []byte{2}, ts},
		"duplicate hash": {"t3", "karakeep", []byte{1}, ts},
	} {
		if _, err := db.Exec(token, args...); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", name)
		}
	}

	// Deleting the root folder keeps the token without a root.
	if _, err := db.Exec(`DELETE FROM favourite_nodes WHERE id = 'f1'`); err != nil {
		t.Fatal(err)
	}
	var rootID sql.NullString
	if err := db.QueryRow(`SELECT root_id FROM app_tokens WHERE id = 't1'`).Scan(&rootID); err != nil || rootID.Valid {
		t.Fatalf("root_id = %v, err %v; want NULL", rootID, err)
	}

	// Deleting the user deletes their tokens.
	if _, err := db.Exec(`DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM app_tokens`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("app_tokens = %d, err %v; want 0", n, err)
	}
}

func TestIntIDsMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0011")); err != nil {
		t.Fatalf("Migrate to 0011: %v", err)
	}

	const t1, t2 = "2026-01-01T00:00:00.000Z", "2026-01-02T00:00:00.000Z"
	node := `INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, created_at, updated_at)
		VALUES (?, 'u1', ?, ?, 0, ?, ?, ?, ?)`
	for _, q := range [][]any{
		{`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?)`, t1, t1},
		{node, "root", nil, "folder", nil, nil, t1, t1},
		{node, "b", "root", "link", "B", "https://b.example", t2, t2},
		{node, "a", "root", "folder", "A", nil, t1, t1},
		{`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES ('t2', 'u1', 'Two', 'two', ?), ('t1', 'u1', 'One', 'one', ?)`, t2, t1},
	} {
		if _, err := db.Exec(q[0].(string), q[1:]...); err != nil {
			t.Fatal(err)
		}
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	nums := func(query string) map[string]int64 {
		t.Helper()
		rows, err := db.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]int64{}
		for rows.Next() {
			var id string
			var n int64
			if err := rows.Scan(&id, &n); err != nil {
				t.Fatal(err)
			}
			out[id] = n
		}
		return out
	}
	// Creation order, then ID for equal times.
	if got := nums(`SELECT id, num FROM favourite_nodes`); got["a"] != 1 || got["root"] != 2 || got["b"] != 3 {
		t.Fatalf("node nums = %v, want a=1 root=2 b=3", got)
	}
	if got := nums(`SELECT id, num FROM tags`); got["t1"] != 1 || got["t2"] != 2 {
		t.Fatalf("tag nums = %v, want t1=1 t2=2", got)
	}

	// New rows continue after the highest number, also after it was deleted.
	if _, err := db.Exec(`DELETE FROM favourite_nodes WHERE id = 'b'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(node, "c", "root", "folder", "C", nil, t2, t2); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES ('t3', 'u1', 'Three', 'three', ?)`, t2); err != nil {
		t.Fatal(err)
	}
	if got := nums(`SELECT id, num FROM favourite_nodes WHERE id = 'c'`); got["c"] != 4 {
		t.Fatalf("new node num = %v, want 4", got)
	}
	if got := nums(`SELECT id, num FROM tags WHERE id = 't3'`); got["t3"] != 3 {
		t.Fatalf("new tag num = %v, want 3", got)
	}
}

func TestLinkPreviewsMigrationOnExistingData(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0013")); err != nil {
		t.Fatalf("Migrate to 0013: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	node := `INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, created_at, updated_at)
		VALUES (?, 'u1', ?, ?, 0, ?, ?, ?, ?)`
	for _, q := range [][]any{
		{`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?)`, ts, ts},
		{node, "root", nil, "folder", nil, nil, ts, ts},
		{node, "l1", "root", "link", "L", "https://example.org", ts, ts},
		{`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES
			('fav1', 'favicon', NULL, 'image/png', 10, ?), ('fav2', 'favicon', NULL, 'image/png', 10, ?),
			('up', 'upload', 'u1', 'image/png', 10, ?)`, ts, ts, ts},
		{`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://a.example', 'fav1', ?), ('https://b.example', 'fav2', ?)`, ts, ts},
		{`INSERT INTO favicons (origin, failed_at) VALUES ('https://c.example', ?)`, ts},
	} {
		if _, err := db.Exec(q[0].(string), q[1:]...); err != nil {
			t.Fatal(err)
		}
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Every favicon still points to its image.
	rows, err := db.Query(`SELECT origin, image_id FROM favicons ORDER BY origin`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]sql.NullString{}
	for rows.Next() {
		var origin string
		var id sql.NullString
		if err := rows.Scan(&origin, &id); err != nil {
			t.Fatal(err)
		}
		got[origin] = id
	}
	rows.Close()
	if got["https://a.example"].String != "fav1" || got["https://b.example"].String != "fav2" || got["https://c.example"].Valid {
		t.Fatalf("favicons = %v, want a→fav1, b→fav2, c→NULL", got)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM images`); n != 3 {
		t.Fatalf("images = %d, want 3", n)
	}

	image := `INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, ?, ?, 'image/jpeg', 10, ?)`
	if _, err := db.Exec(image, "prev", "preview", "u1", ts); err != nil {
		t.Fatalf("preview with user: %v", err)
	}
	if _, err := db.Exec(image, "prev2", "preview", nil, ts); err == nil {
		t.Error("preview without user: insert succeeded, want a constraint error")
	}

	// Links take a description and a preview, folders don't.
	if _, err := db.Exec(`UPDATE favourite_nodes SET description = 'D', preview_image_id = 'prev' WHERE id = 'l1'`); err != nil {
		t.Fatalf("link: %v", err)
	}
	for name, q := range map[string]string{
		"folder description": `UPDATE favourite_nodes SET description = 'D' WHERE id = 'root'`,
		"folder preview":     `UPDATE favourite_nodes SET preview_image_id = 'prev' WHERE id = 'root'`,
		"unknown image":      `UPDATE favourite_nodes SET preview_image_id = 'nope' WHERE id = 'l1'`,
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("%s: update succeeded, want a constraint error", name)
		}
	}

	// Deleting the preview image leaves the link without one.
	if _, err := db.Exec(`DELETE FROM images WHERE id = 'prev'`); err != nil {
		t.Fatal(err)
	}
	var prev sql.NullString
	if err := db.QueryRow(`SELECT preview_image_id FROM favourite_nodes WHERE id = 'l1'`).Scan(&prev); err != nil || prev.Valid {
		t.Fatalf("preview_image_id = %v, err %v; want NULL", prev, err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestJobsConstraints(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const ts = "2026-01-01T00:00:00.000Z"
	job := `INSERT INTO jobs (kind, dedup_key, payload, created_at, started_at) VALUES (?, ?, '{}', ?, ?)`
	if _, err := db.Exec(job, "favicon", "https://a.example", ts, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(job, "favicon", "https://a.example", ts, nil); err == nil {
		t.Error("second pending job with the same key: insert succeeded, want a constraint error")
	}
	// A running job doesn't block a new pending one, and other kinds or no key don't collide.
	for name, args := range map[string][]any{
		"running":      {"favicon", "https://a.example", ts, ts},
		"other kind":   {"preview_fetch", "https://a.example", ts, nil},
		"no key":       {"preview_convert", nil, ts, nil},
		"no key again": {"preview_convert", nil, ts, nil},
	} {
		if _, err := db.Exec(job, args...); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := db.Exec(job, "thumbnail", nil, ts, nil); err == nil {
		t.Error("unknown kind: insert succeeded, want a constraint error")
	}
}

func TestNotesConstraints(t *testing.T) {
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
	category := `INSERT INTO note_categories (id, user_id, parent_id, title, icon_symbol, icon_image_id, created_at, updated_at)
		VALUES (?, ?, ?, 'C', ?, ?, '` + ts + `', '` + ts + `')`
	note := `INSERT INTO notes (id, user_id, category_id, icon_symbol, icon_image_id, content, created_at, updated_at)
		VALUES (?, 'u1', ?, ?, ?, '{"v":1,"blocks":[]}', '` + ts + `', '` + ts + `')`

	mustExec(`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', ?, ?), ('u2', 'b@example.org', ?, ?)`, ts, ts, ts, ts)
	mustExec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES ('i1', 'upload', 'u1', 'image/png', 1, ?)`, ts)
	mustExec(category, "c1", "u1", nil, nil, nil)
	mustExec(category, "c2", "u1", "c1", nil, "i1")
	mustExec(note, "n1", "c2", nil, "i1")
	mustExec(note, "n2", "c1", "home", nil)

	for name, args := range map[string][]any{
		"parent of another user": {"x1", "u2", "c1", nil, nil},
		"symbol and image":       {"x2", "u1", nil, "home", "i1"},
	} {
		if err := exec(category, args...); err == nil {
			t.Errorf("category %s: insert succeeded, want a constraint error", name)
		}
	}
	if err := exec(note, "x3", nil, "home", "i1"); err == nil {
		t.Error("note with symbol and image: insert succeeded, want a constraint error")
	}
	if err := exec(`DELETE FROM note_categories WHERE id = 'c1'`); err == nil {
		t.Error("deleting a category with subcategories succeeded, want a constraint error")
	}

	// Deleting an image clears the icons that use it.
	mustExec(`DELETE FROM images WHERE id = 'i1'`)
	var n int
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM notes WHERE icon_image_id IS NOT NULL) + (SELECT COUNT(*) FROM note_categories WHERE icon_image_id IS NOT NULL)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("icons with an image = %d, err %v; want 0 after deleting the image", n, err)
	}

	// Deleting a category makes its notes uncategorized.
	mustExec(`DELETE FROM note_categories WHERE id = 'c2'`)
	var categoryID *string
	if err := db.QueryRow(`SELECT category_id FROM notes WHERE id = 'n1'`).Scan(&categoryID); err != nil || categoryID != nil {
		t.Fatalf("category of n1 = %v, err %v; want NULL", categoryID, err)
	}

	mustExec(`DELETE FROM users WHERE id = 'u1'`)
	for _, table := range []string{"notes", "note_categories"} {
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s = %d, err %v; want 0 after deleting the user", table, n, err)
		}
	}
}

func TestAppTokenNoteScopeMigration(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0018")); err != nil {
		t.Fatalf("Migrate to 0018: %v", err)
	}
	const ts = "2026-01-01T00:00:00.000Z"
	for _, q := range []string{
		`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', '` + ts + `', '` + ts + `')`,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, created_at, updated_at) VALUES ('root', 'u1', NULL, 'folder', 0, '` + ts + `', '` + ts + `')`,
		`INSERT INTO app_tokens (id, user_id, name, app, root_id, token_hash, prefix, created_at) VALUES ('k', 'u1', 'Phone', 'karakeep', 'root', x'01', 'bdly_abcdefg', '` + ts + `')`,
		`INSERT INTO app_tokens (id, user_id, name, app, root_id, token_hash, prefix, created_at) VALUES ('l', 'u1', 'Tablet', 'linkwarden', 'root', x'02', 'bdly_abcdefh', '` + ts + `')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for id, want := range map[string]string{"k": "all", "l": "none"} {
		var scope string
		var category *string
		if err := db.QueryRow(`SELECT note_scope, root_category_id FROM app_tokens WHERE id = ?`, id).Scan(&scope, &category); err != nil || scope != want || category != nil {
			t.Errorf("token %s: scope %q, category %v, err %v; want %q", id, scope, category, err, want)
		}
	}

	// Deleting the root category leaves the scope, so it can't widen to all notes.
	if _, err := db.Exec(`INSERT INTO note_categories (id, user_id, title, created_at, updated_at) VALUES ('c1', 'u1', 'Inbox', ?, ?)`, ts, ts); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE app_tokens SET note_scope = 'category', root_category_id = 'c1' WHERE id = 'k'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM note_categories WHERE id = 'c1'`); err != nil {
		t.Fatal(err)
	}
	var scope string
	var category *string
	if err := db.QueryRow(`SELECT note_scope, root_category_id FROM app_tokens WHERE id = 'k'`).Scan(&scope, &category); err != nil || scope != "category" || category != nil {
		t.Errorf("after deleting the category: scope %q, category %v, err %v", scope, category, err)
	}
	if _, err := db.Exec(`UPDATE app_tokens SET note_scope = 'some' WHERE id = 'k'`); err == nil {
		t.Error("an unknown note scope was accepted")
	}
}

func TestNotePinArchiveMigration(t *testing.T) {
	db, err := Open(openTemp(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db, migrationsUpTo(t, "0019")); err != nil {
		t.Fatalf("Migrate to 0019: %v", err)
	}
	const ts = "2026-01-01T00:00:00.000Z"
	for _, q := range []string{
		`INSERT INTO users (id, email, created_at, updated_at) VALUES ('u1', 'a@example.org', '` + ts + `', '` + ts + `')`,
		`INSERT INTO notes (id, user_id, content, created_at, updated_at) VALUES ('n1', 'u1', '{}', '` + ts + `', '` + ts + `')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	if err := Migrate(ctx, db, Migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var pinned, archived int
	if err := db.QueryRow(`SELECT pinned, archived FROM notes WHERE id = 'n1'`).Scan(&pinned, &archived); err != nil || pinned != 0 || archived != 0 {
		t.Fatalf("existing note: pinned %d, archived %d, err %v; want 0, 0", pinned, archived, err)
	}
	if _, err := db.Exec(`UPDATE notes SET pinned = 2 WHERE id = 'n1'`); err == nil {
		t.Error("pinned = 2 was accepted")
	}
}
