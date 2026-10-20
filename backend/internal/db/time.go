package db

import (
	"context"
	"database/sql"
	"time"
)

// TimeFormat is the fixed-width RFC 3339 UTC format used for all stored
// timestamps. Fixed width keeps string comparison in SQL chronological.
const TimeFormat = "2006-01-02T15:04:05.000Z"

// FormatTime formats t for storage.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}

// ParseTime parses a stored timestamp.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}

// Querier is implemented by *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
