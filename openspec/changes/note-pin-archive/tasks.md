# Tasks

## 1. Data model and store

- [x] 1.1 Add `backend/internal/db/migrations/0020_note_pin_archive.sql` with `pinned` and `archived` (`INTEGER NOT NULL DEFAULT 0`) on `notes`; verify the db migration test passes and existing notes read as unpinned and unarchived
- [x] 1.2 Add `Pinned` and `Archived` to `notes.Note`, `NewNote` and `NotePatch`, read and write them in the store, and make a flag change bump the version and `updated_at`; verify with store tests for create with flags, pin, archive and the version bump

## 2. Notes API

- [ ] 2.1 Decode `pinned` and `archived` in `POST /api/notes` and `PATCH /api/notes/{id}`, answer `invalid_note` for non-boolean values, and return both fields on every note; verify with httptest cases for "Archive through the API" and "Flag that isn't a boolean"

## 3. Widget queries and rows

- [ ] 3.1 Order `RecentNotes` and `ListedNotes` by pinned first before the limit, including the stable partition in the alphabetical sort; verify with tests for "Pinned first", "Pinned within the count" and "Pinned note outside the day limit"
- [ ] 3.2 Order selected notes pinned first, keeping list order in each group; verify with the selected notes "Pinned first" test
- [ ] 3.3 Add `pinned` and `archived` to the resolved rows of the recent notes, note list, selected notes and note widgets, for owners and visitors; verify the public dashboard tests contain the flags and still nothing else

## 4. Frontend: notes page and side panel

- [ ] 4.1 Add `pinned` and `archived` to the note types and API client; verify `api.test.ts`
- [ ] 4.2 Filter archived notes out of "All notes", "Uncategorized" and categories, add the "Archived" entry with its count and `?category=archived`, and sort pinned first on `/notes`; verify with tests for "Archived notes not counted", "Choose the archive", "Pinned first" and "Pinned note of another category"
- [ ] 4.3 Add the empty states "All notes are archived" and "No archived notes"; verify with tests for "Everything archived" and "Empty archive"
- [ ] 4.4 Make "New note" from "Archived" start without a category; verify "New note from the archive"

## 5. Frontend: marks and editor menu

- [ ] 5.1 Add a shared `NoteMarks` component (pin icon, archived mark) and use it on `NoteCard` with dimming for archived cards; verify "Pinned card" and "Archived card"
- [ ] 5.2 Add "Pin note"/"Unpin note" and "Archive note"/"Unarchive note" above "Delete note" in the editor menu, saving through the autosave flush, and show the marks in the editor header; verify "Menu order", "Pin from the editor", "Unarchive" and "Changed elsewhere"
- [ ] 5.3 Show the marks and dimming in `NoteRows` and the note widget, and refresh widgets after pinning or archiving in the dashboard modal; verify "Pinned and archived rows", "Archived note" and "Pin in the modal"
- [ ] 5.4 Add all new strings in English and German; verify the i18n test passes

## 6. Wrap-up

- [ ] 6.1 Document pinning and archiving in the README notes section, then run `make check` and `make test` and verify both pass
