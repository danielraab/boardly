# Design

## Context

Three places in widget settings let the owner pick notes:

- The note widget settings (`widgets/note/index.tsx`) use a native `<select>` whose options show `displayTitle(n)` only.
- The selected notes widget's `NoteSearch` is a Base UI `Combobox` over `noteChoices()` (`selectedNotes/selected.ts`). That function already returns each note with its display title and category path, filters by title or path (case and accent insensitive) and sorts by title. Its rows render icon, title and a muted right-aligned path inline.
- The selected notes widget's `ChosenNotes` renders a sortable row per chosen note with icon and title only.

The archived look already exists: `NoteMarks` (`features/notes/NoteMarks.tsx`) renders an archive icon with the tooltip "Archived" in `compact` mode, and `NoteRows` and `NoteCard` dim archived notes with `opacity-70`.

## Goals / Non-Goals

**Goals:**
- One way to show a note as a choice (icon, title, path, archived mark) everywhere a note is picked in widget settings.
- One search behaviour for both comboboxes, from `noteChoices()`.

**Non-Goals:**
- Telling apart notes with the same title in the same category.
- Showing the pin mark in pickers (pinning doesn't help identify a note).
- Changing the order of results, e.g. sorting archived notes last.
- Any backend or config change.

## Decisions

**A shared `NoteChoiceLabel` for the inside of a row.** A small component in `widgets/`, next to `NoteRows`, takes the note, its display title and its path and renders: icon, truncating title, compact `NoteMarks` with `pinned={false}`, and the muted path. It's used in both comboboxes' items and in `ChosenNotes`, and in the note widget's closed field. The caller adds dimming (`opacity-70`) to its own row element, since the row is a `Combobox.Item` in one place and a sortable `<li>` in another. Alternative: copying the markup into each place. Rejected, since three copies would drift, and the archived mark is exactly the kind of detail that gets forgotten in one of them.

**Title and path share the line.** The title takes the free room and truncates; the path keeps its width up to half of the row (`max-w-1/2`) and truncates beyond that, so a long path can't push the title out. This is the same in the combobox items and in `ChosenNotes`, where the entry also gets a `title` tooltip of "Groceries · Work › Office". In `ChosenNotes` the dimming of archived notes is on the label, not on the whole entry, so the buttons keep their contrast. Alternative: path on a second line. Rejected, since it doubles the height of the list of up to 100 notes.

**`noteChoices()` is reused as is.** The selected notes search passes the chosen IDs as today; the note widget passes `[]`, so all notes are offered. A `notePath()` helper next to it gives a single note's category path, for `ChosenNotes` and the closed field.

**Note widget: a single-select `Combobox` in its own component.** A new `NotePicker` (in `widgets/note/`) wraps `Combobox.Root` with `value={noteId}`, `items` from `noteChoices()`, `filter={null}` (we filter ourselves) and the same popup and empty state as `NoteSearch`. The two comboboxes stay separate components because their behaviour differs (add-and-clear vs. select-and-show, the 100-note limit), while their item rendering is shared through `NoteChoiceLabel`. Alternative: generalise `NoteSearch` with a mode prop. Rejected, since the modes share little beyond the item rendering.

**Closed state of the note widget field.** While the input's text is the chosen note's display title, that text is made transparent and a `NoteChoiceLabel` of the chosen note is laid over it, inside the input's border and padding, so the field reads like a result row; all notes are offered then, not only those matching the title. Once the owner types, the overlay goes away and the typed text shows. The input value is controlled, so on closing the popup without choosing (`onOpenChange(false)`) the input is set back to the chosen note's title; choosing sets it to the new note's title through `itemToStringLabel`. Alternative: drawing icon and path as separate adornments with reserved input padding. Rejected, since the path's width varies. A deleted note leaves the field empty with the existing "This note no longer exists" hint below it.

**Strings.** The note widget reuses `dashboards.widgets.note.notePlaceholder` ("Choose a note") as placeholder and `dashboards.widgets.selectedNotes.noMatches`'s text, by moving "No matching notes" to a shared key, or adding `dashboards.widgets.note.noMatches` with the same text. Adding the key in the note namespace is preferred: keys are namespaced by area. The archived tooltip reuses `notes.archived`.

## Risks / Trade-offs

- [Combobox on mobile is less native than a `<select>`] → The selected notes search already uses it on mobile, and search helps more than a long native list once there are many notes.
- [The overlay hides the input's own text, so selecting that text shows nothing] → Selecting the closed field's text is rare; typing replaces the overlay right away.
- [Tests that select a note through `<select>` in `Note.test.tsx` and dashboard tests break] → They are rewritten to type into the search and choose a result, as `SelectedNotes.test.tsx` does.
