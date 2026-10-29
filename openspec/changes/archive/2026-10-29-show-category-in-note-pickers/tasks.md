# Tasks

## 1. Shared note choice rendering

- [x] 1.1 Add `NoteChoiceLabel` (icon, truncating title, compact `NoteMarks` for archived, muted category path) and use it in `NoteSearch` items, dimming archived results; verify with a `SelectedNotes.test.tsx` case for "Archived note in the search" (Adding notes) and that the existing search tests still pass
- [x] 1.2 Show the category path and archived mark in `ChosenNotes` via `NoteChoiceLabel` (categories passed in), with title and path truncating, a "title · path" tooltip and dimming for archived entries; verify with tests for "Same title in different categories" and "Archived note in the list"

## 2. Note widget picker

- [x] 2.1 Add a single-select `NotePicker` combobox in `widgets/note/` over `noteChoices(notes, categories, [], query)`, with "Choose a note" placeholder, "No matching notes" empty state and the closed field showing the chosen note's icon, title, path and archived mark; verify with `Note.test.tsx` cases for "Same title in different categories", "Search by category", "Archived note in the search", "Chosen note shown" and "No match"
- [x] 2.2 Replace the `<select>` in the note widget settings with `NotePicker`, keeping the confirm rule and the deleted-note hint; verify "Add a note widget", "No note chosen", "Untitled note in the select" and the deleted-note test pass after rewriting them to use the search
- [x] 2.3 Update other tests that pick a note through the old select (e.g. `Dashboards.test.tsx`) to use the search; verify they pass

## 3. Strings and wrap-up

- [x] 3.1 Add `dashboards.widgets.note.noMatches` in English and German; verify the i18n test passes
- [x] 3.2 Run `make check` and `make test` and verify both pass
