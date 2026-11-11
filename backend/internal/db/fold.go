package db

import (
	"database/sql/driver"
	"strings"

	"modernc.org/sqlite"
)

// Fold is the case folding of boardly_fold, for values compared with it.
func Fold(s string) string { return strings.ToLower(s) }

// boardly_fold(text) lowers all letters, not only ASCII as SQLite's lower()
// does, and keeps NULL. It is registered before any connection is opened.
func init() {
	sqlite.MustRegisterDeterministicScalarFunction("boardly_fold", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			switch v := args[0].(type) {
			case nil:
				return nil, nil
			case string:
				return Fold(v), nil
			case []byte:
				return Fold(string(v)), nil
			default:
				return v, nil
			}
		})
}
