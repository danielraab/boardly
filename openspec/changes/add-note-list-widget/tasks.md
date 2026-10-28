# Tasks

## 1. Backend

- [x] 1.1 Take the recursive subcategory condition out of `scopeQuery` into a shared helper and add `notes.Store.ListedNotes` with a `Selection` (all / none / category ± subcategories) and the sorts `updated`, `created`, `title` (Go collator, untitled last, ties by ID, limit after sorting); verify with tests in `notes/listed_test.go` covering each selection, each sort and the limit, and verify the existing scope tests still pass
- [x] 1.2 Add the `note_list` kind in `widgets/notelist.go` with strict `Normalize` (category mode, category ID length, sort, limit 1–`maxRecentLimit`, title, size) and register it in `widgets.Register`; verify with `notelist_test.go` cases for every `invalid_layout` rule and that a missing category still saves
- [x] 1.3 Implement `Resolve`: flag off or missing category → left out for visitors; missing category → `{"missing": true}` for the owner; rows with display title, icon and a second line only for notes not directly in the chosen category (path for the owner only), plus `id` for the owner; verify with tests that the visitor JSON has no IDs, content, times, paths or chosen category name
- [x] 1.4 Add a dashboard lookup test in `httpapi/dashboards_test.go` for a public note list (flag on/off); verify `make test` passes for the backend

## 2. Frontend

- [x] 2.1 Add `widgets/noteList/list.ts` with `noteList(notes, categories, config)` mirroring the server's selection, order, limit, second line and missing state; verify with `list.test.ts` using the same cases as the Go tests
- [x] 2.2 Pull the row rendering out of `RecentNotesView` into a shared `NoteRows` component with an optional trailing time; verify the existing `RecentNotes.test.tsx` still passes unchanged
- [x] 2.3 Let `CategoryPicker` take a second optional extra choice ahead of the tree (for "All notes" and "Uncategorized") without changing its existing callers; verify the settings, share and app-token tests still pass
- [x] 2.4 Add the `noteList` widget kind (settings dialog with category, disabled "Include subcategories" for All/Uncategorized, sort, number of notes, title, size, public flag with hint; view with title heading, rows, empty states, missing state, `NoteDialog` for the owner outside edit mode, edit-mode preview from the notes queries) and register it in the "notes" group after recent notes; verify with `NoteList.test.tsx` covering the settings defaults and validation, rows with and without the second line, opening the modal, edit mode and the visitor view
- [x] 2.5 Add en + de translations under `dashboards.widgets.noteList`; verify `make check` passes (including i18n key checks) and update `DashboardWidgets.test.tsx` for the picker's "Notes" group

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify both pass
- [ ] 3.2 Run the app, add a note list for "All notes", "Uncategorized" and a category with and without subcategories in each sort, open a note from it, delete its category, and verify the rows, second lines, empty and missing states and the public dashboard view by hand
