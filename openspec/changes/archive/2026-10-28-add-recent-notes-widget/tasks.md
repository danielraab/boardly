# Tasks

## 1. Notes store and display title

- [x] 1.1 Add `RecentNotes` to the notes store (created or updated column, optional `since`, `ORDER BY <col> DESC, id DESC LIMIT`) and verify with Go tests: order by created and by updated, limit, `since` filter on each column, ID tie-break, and that another user's notes are never returned
- [x] 1.2 Add `DisplayTitle` to `notes` (title, else first non-empty line in reading order cut after 80 characters with "…", else nil) and verify with Go tests mirroring `text.test.ts` (paragraphs with line breaks, headings, code, nested lists, dividers, links, long lines, empty content)

## 2. Widget kind (backend)

- [x] 2.1 Add the `recent_notes` kind in `backend/internal/widgets/recentnotes.go` with config validation (sortBy `created`/`updated`, limit 1–100, days nil or 1–365, title, size), sharing the limits with recent favourites, and verify Normalize tests reject each invalid value with `invalid_layout`
- [x] 2.2 Implement Resolve (left out for visitors without `showIfPublic`, `since = now − days×24h`, rows with display title, icon, chosen time and category title; owner rows add `id` and category path) and register it in `widgets.Register`; verify with tests for flag off/on, sort by updated, day limit, and that a visitor row has only title/icon/time/category title and no content or ID
- [x] 2.3 Add an httptest-level check that a dashboard lookup returns the widget's notes, and verify the backend tests pass

## 3. Frontend widget

- [x] 3.1 Move `relativeTime` and `fullDate` to a shared module for both recent widgets and add `recentNotes(notes, sortBy, limit, days, now)` for the edit-mode preview; verify unit tests for both sorts, window, tie-break and limit
- [x] 3.2 Add `widgets/recentNotes` with config and data types and the settings form (sort by, number of notes, optional days, title, size, public flag with hint; can't confirm with out-of-range numbers), registered in `registry.ts` in the "notes" group after "Note"; verify with a Testing Library test of the defaults, validation and the picker group
- [x] 3.3 Add the view: rows with icon, display title, category title as a smaller second line with the path (owner) or title (visitor) as tooltip, relative time with full-date tooltip, sizes, title heading, empty states; edit mode rendered from the notes queries; verify with Testing Library tests for each
- [x] 3.4 Open the note editor modal when the owner activates a row outside edit mode (reusing or generalising `NoteDialog`), refresh the widget after saves and deletion, and make rows inert in edit mode and for visitors; verify with Testing Library tests
- [x] 3.5 Add all en + de strings (kind name and description, setting labels and choices, public hint, empty states) and verify the i18n test passes

## 4. Check

- [x] 4.1 Run `make check` and `make test` and verify both pass
- [x] 4.2 Run the app, add a recent notes widget with each sort and a day limit, open a note from it, and verify the list, the empty state and the public dashboard view by hand
