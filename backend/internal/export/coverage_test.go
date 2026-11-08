package export

import (
	"slices"
	"testing"

	"boardly/internal/dbtest"
)

// exportedTables are read into the export document.
var exportedTables = []string{
	"favourite_nodes", "tags", "favourite_node_tags",
	"dashboards", "dashboard_widgets",
	"note_categories", "notes", "note_attachments", "attachments",
	"media_sources", "feed_subscriptions", "feed_items", "feed_item_flags", "images", "offline_copies",
}

// notExportedTables belong to users but are left out on purpose: sign-in
// data, connected apps, the shared favicon cache and internal state.
var notExportedTables = []string{
	"sessions", "passkeys", "webauthn_challenges", "app_tokens", "favicons",
}

// TestEveryUserTableIsCovered fails when a table that holds user data (it
// references users, directly or through another such table) is neither
// exported nor explicitly left out, so a new table can't be forgotten.
func TestEveryUserTableIsCovered(t *testing.T) {
	database := dbtest.Open(t)
	rows, err := database.Query(
		`SELECT m.name, f."table" FROM sqlite_master m, pragma_foreign_key_list(m.name) f WHERE m.type = 'table'`)
	if err != nil {
		t.Fatal(err)
	}
	refs := map[string][]string{} // table → tables it references
	for rows.Next() {
		var table, target string
		if err := rows.Scan(&table, &target); err != nil {
			t.Fatal(err)
		}
		refs[table] = append(refs[table], target)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	userTables := map[string]bool{"users": true}
	for changed := true; changed; {
		changed = false
		for table, targets := range refs {
			if userTables[table] {
				continue
			}
			for _, target := range targets {
				if userTables[target] {
					userTables[table], changed = true, true
					break
				}
			}
		}
	}
	delete(userTables, "users")

	for table := range userTables {
		if !slices.Contains(exportedTables, table) && !slices.Contains(notExportedTables, table) {
			t.Errorf("table %q holds user data but is neither exported nor listed as left out", table)
		}
	}
	for _, table := range append(slices.Clone(exportedTables), notExportedTables...) {
		if !userTables[table] {
			t.Errorf("table %q is listed but doesn't hold user data (renamed or dropped?)", table)
		}
	}
}
