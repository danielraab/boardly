# Proposal

## Why

A dashboard can show one note, the newest notes or the notes of a category, but not a hand-picked set of notes from different categories, like the five notes used every day. A widget that lists chosen notes in an order the owner sets fills that gap.

## What Changes

- New content widget kind "Selected notes" / "Ausgewählte Notizen" (`selected_notes`) in the "Notes" group of the widget picker.
- Settings: the notes (1–100, each at most once), added from a searchable list of the owner's notes and put in a manual order by dragging or with move up / move down buttons; an optional title; a size (S/M/L) and "Show on public dashboard".
- Rows look like those of "Note list": the note's icon and display title, and for every note with a category that category's title as a small second line with its path as tooltip. No colour, no content, no time.
- For the owner, outside edit mode, activating a row opens the note's editor in the same modal as "Recent notes" and "Note list".
- Deleted notes are skipped. The settings dialog marks them so they can be removed. When none of the notes exists anymore, the owner sees "None of the selected notes exist anymore".
- Visitors see only the shown rows, without note IDs, if "Show on public dashboard" is on. Rows can't be activated.

## Capabilities

### New Capabilities
- `selected-notes-widget`: a dashboard widget that lists notes the owner picked, in the order the owner set.

### Modified Capabilities
- `dashboard-editing`: the widget picker's "Notes" group also lists "Selected notes".

## Impact

- Backend: new widget kind in `backend/internal/widgets`, registered in `widgets.Register`; a notes store query for a user's notes by a list of IDs; reuses `notes.DisplayTitle`, `categoryOf`, the note list's row structs and `maxRecentLimit`. No migration.
- Frontend: new kind in `frontend/src/features/dashboards/widgets/selectedNotes`, added to the widget registry in the "notes" group; reuses `NoteRows`, `NoteDialog`, `categoryPath` and `@dnd-kit/sortable` (already a dependency); en + de translations.
- Public dashboard lookup: the widget is left out for visitors unless "Show on public dashboard" is on, and when none of its notes exists.
- Builds on the in-progress `add-recent-notes-widget` and `add-note-list-widget` changes (their code is already committed on this branch); all three modify "Add widgets" in `dashboard-editing`, so they should be archived in that order.
