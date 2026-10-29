# Proposal

## Why

Notes in different categories can have the same title (e.g. "Groceries" in "Home" and in "Work"). The note widget's settings offer notes in a plain select by title only, and the selected notes widget's list of chosen notes shows only the title too. In both places the owner can't tell such notes apart, or see which notes are archived.

## What Changes

- The note widget's settings replace the note select with a search field, like "Add a note…" in the selected notes widget. Each result shows the note's icon, display title and category path, and typing filters by title or category path. Closed, the field shows the chosen note in the same way.
- The selected notes widget's list of chosen notes shows each note's category path after its title.
- Archived notes are recognizable in all three places (both searches and the chosen list): an archive icon with the tooltip "Archived" / "Archiviert", and a dimmed row.
- Notes with the same title in the same category are still indistinguishable. That case is out of scope.

## Capabilities

### New Capabilities

### Modified Capabilities
- `note-widget`: "Note widget settings" offers the note through a search with icon, display title, category path and archived mark instead of a select by display title.
- `selected-notes-widget`: "Adding notes" results mark archived notes; "Ordering and removing notes" shows each chosen note's category path and marks archived notes.

## Impact

- Frontend only: `features/dashboards/widgets/note/index.tsx` (settings), `features/dashboards/widgets/selectedNotes/` (`NoteSearch`, `ChosenNotes`, `selected.ts`), and a shared row for a note choice; new en/de strings if any.
- No backend, API or layout format change; the note widget's config stays `{ noteId, size, showIfPublic }`.
