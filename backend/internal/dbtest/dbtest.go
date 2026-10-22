// Package dbtest provides migrated temporary databases for tests.
package dbtest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"boardly/internal/db"
)

// Open returns a fresh database with all shipped migrations applied.
// It is closed when the test ends.
func Open(t testing.TB) *sql.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "boardly.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(context.Background(), database, db.Migrations); err != nil {
		t.Fatal(err)
	}
	return database
}
