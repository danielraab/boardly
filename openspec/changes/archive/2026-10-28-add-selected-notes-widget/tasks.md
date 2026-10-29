# Tasks

## 1. Backend

- [x] 1.1 Add `notes.Store.NotesByID(ctx, userID, ids)` returning the user's notes among the IDs; verify with tests that other users' notes and unknown IDs are left out and an empty ID list returns none
- [x] 1.2 Add the `selected_notes` kind in `widgets/selectednotes.go` with strict `Normalize` (1–`maxRecentLimit` note IDs, each non-empty, at most `maxIDLength` and unique; title; size) and register it in `widgets.Register`; verify with `selectednotes_test.go` cases for every `invalid_layout` rule (empty, 101 IDs, empty or too long ID, duplicate, title, size) and that IDs of missing notes still save
- [x] 1.3 Implement `Resolve`: flag off → left out for visitors; no note found → `{"missing": true}` for the owner and left out for visitors; otherwise rows in config order with missing notes skipped, display title, icon and a second line for every note with a category (path for the owner only), plus `id` for the owner; verify with tests for order, skipped notes, the second line, and that the visitor JSON has no IDs, content, times or paths
- [x] 1.4 Add a dashboard lookup test in `httpapi/dashboards_test.go` for public selected notes (flag on/off, all notes deleted); verify `make test` passes for the backend

## 2. Frontend

- [x] 2.1 Add `widgets/selectedNotes/selected.ts` with `selectedNotes(notes, categories, config)` mirroring the server's order, skipping, second line and missing state; verify with `selected.test.ts` using the same cases as the Go tests
- [x] 2.2 Add the note search (Base UI `Combobox` over the owner's notes not in the list, sorted by display title with untitled last, matching title or category path ignoring case and accents, "No matching notes", disabled at 100); verify with tests for filtering by title and by category, accent-insensitive matching, hiding chosen notes and the disabled state
- [x] 2.3 Add the sortable list of chosen notes (dnd-kit with pointer, touch and keyboard sensors on a drag handle; "Move up", "Move down", "Remove"; warning row "This note no longer exists" for IDs not in the notes query); verify with tests for move up/down including the disabled ends, remove offering the note again, and the warning row
- [x] 2.4 Add the `selectedNotes` widget kind (settings dialog with the search, the list, title, size, public flag with hint, not confirmable when empty; view with title heading, `NoteRows` without times, missing state, `NoteDialog` for the owner outside edit mode, edit-mode preview from the notes queries), sharing the note list's view where that reads well, and register it in the "notes" group after note list; verify with `SelectedNotes.test.tsx` covering the settings defaults and validation, rows in list order with and without the second line, skipped deleted notes, the all-missing state, opening the modal, edit mode and the visitor view
- [x] 2.5 Add en + de translations under `dashboards.widgets.selectedNotes`; update `DashboardWidgets.test.tsx` for the picker's "Notes" group; verify `make check` passes (including i18n key checks)

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify both pass
- [x] 3.2 Run the app, add selected notes from several categories, search by title and category, reorder by dragging, by keyboard and with the buttons, open a note from the widget, delete one and then all notes, and verify the rows, order, warning rows, missing state and the public dashboard view by hand
