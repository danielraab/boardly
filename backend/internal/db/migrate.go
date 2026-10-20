package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations are the schema migrations shipped with the binary.
var Migrations fs.FS = mustSub(migrationFiles, "migrations")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

type migration struct {
	version int
	name    string
}

// Migrate applies every migration in fsys (files named NNNN_name.sql) that has
// not been applied yet, each in its own transaction, in version order.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	migrations, err := listMigrations(fsys)
	if err != nil {
		return err
	}

	applied := map[int]bool{}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("reading applied migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if err := apply(ctx, db, fsys, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		slog.InfoContext(ctx, "applied migration", "name", m.name)
	}
	return nil
}

func listMigrations(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("listing migrations: %w", err)
	}
	var out []migration
	seen := map[int]string{}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(prefix)
		if !ok || err != nil {
			return nil, fmt.Errorf("migration %q: name must start with a number and an underscore", e.Name())
		}
		if other, dup := seen[v]; dup {
			return nil, fmt.Errorf("migrations %q and %q share version %d", other, e.Name(), v)
		}
		seen[v] = e.Name()
		out = append(out, migration{version: v, name: e.Name()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func apply(ctx context.Context, db *sql.DB, fsys fs.FS, m migration) error {
	script, err := fs.ReadFile(fsys, m.name)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if strings.TrimSpace(stripComments(string(script))) != "" {
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		m.version, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// stripComments removes "--" line comments, so comment-only migrations are not executed.
func stripComments(script string) string {
	var b strings.Builder
	for line := range strings.Lines(script) {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			b.WriteString(line)
		}
	}
	return b.String()
}
