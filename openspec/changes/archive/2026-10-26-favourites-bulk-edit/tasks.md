# Tasks

## 1. Backend

- [x] 1.1 Extract the bodies of `Store.Move` and the gap closing of `Store.Delete` into helpers taking a `db.Querier`; verify the existing favourites store and httpapi tests pass unchanged
- [x] 1.2 Add `Store.Bulk` with `BulkOp` (delete, addTags, removeTags, setTags, move) in one transaction, with validation (ownership → `ErrNotFound`, kinds/root → `ErrInvalidField`, overlap or target in subtree → `ErrInvalidMove`); verify store tests for each action, tag cleanup, append order on move, and all-or-nothing on errors
- [x] 1.3 Add `POST /api/favourites/bulk` (1–1000 unique IDs, field validation, 204) to the router and handler; verify httpapi tests for each scenario of "Bulk API", including another user's ID → 404
- [x] 1.4 Run `go test ./...` in `backend/` and verify it passes

## 2. Frontend helpers

- [x] 2.1 Add `bulkFavourites(op)` to `api.ts` (with a `keepalive` option)
- [x] 2.2 Add `checkState`, `toggleSelection` (folder cascades over shown nodes), `selectRange`, `selectedLinks`, `moveRoots` and `selectionTagCounts` to `tree.ts`; verify unit tests (cascade, mixed state, empty folder, filtered folder, range, move roots for full and partial folders)

## 3. Selection mode

- [x] 3.1 Add the "Select" / "Done" toggle, Escape handling and checkboxes in place of drag handles in `TreeRow`, with mixed state, shift-click ranges and "Select all"; verify page tests for each scenario of "Selection mode"
- [x] 3.2 Add the sticky selection bar with link and folder counts (including hidden selected nodes), disabled states and the 1000-node limit message; verify tests for counts and "Only folders"

## 4. Bulk actions

- [x] 4.1 Add the "Add tags", "Remove tags" and "Set tags" dialogs (reusing `TagInput`, replace warning on set, counts on remove), sending the bulk request and keeping the selection; verify tests for the "Bulk tag actions" scenarios
- [x] 4.2 Add `disabledIds` to `FavouritePicker` and the "Move to…" dialog using `moveRoots`; verify picker tests for disabled folders and page tests for the "Bulk move" scenarios
- [x] 4.3 Add the `Toast` component (live region, action, pause on hover/focus); verify component tests
- [x] 4.4 Implement the delayed bulk delete (hidden pending IDs, Undo, send on timeout, next delete, route change and `pagehide` with keepalive, restore and error on failure); verify tests with fake timers for delete, undo, commit, folder stays, and failure

## 5. Translations and verification

- [x] 5.1 Add en/de strings (select, done, select all, bar counts with plurals, action labels, dialog texts, toast, limit message); verify the i18n key parity test passes
- [x] 5.2 Run frontend lint, typecheck and `vitest` and verify they pass
- [x] 5.3 Run the app and check selection, every bulk action, undo, selection with a filter active, and phone width in light and dark themes
