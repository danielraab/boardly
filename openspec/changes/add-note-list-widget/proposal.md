# Proposal

## Why

A dashboard can show one note or the newest notes, but not the notes of one topic. A widget that lists the notes of a chosen category, like "Link list" does for a favourites folder, gives the start page quick access to e.g. all "Work" notes.

## What Changes

- New content widget kind "Note list" / "Notizliste" (`note_list`) in the "Notes" group of the widget picker.
- Settings: a required category, chosen from "All notes", "Uncategorized" or one of the owner's categories; "Include subcategories" (default on, only for a category); a sort ("Last changed" default, "Added", "Alphabetical"); a required number of notes (1–100, default 10); an optional title; a size (S/M/L) and "Show on public dashboard".
- Each row shows the note's icon if it has one and its display title (title, or the first line of its content). No colour, no content, no date.
- Notes that are in a subcategory of the chosen category (with "All notes": in any category) show that category's title as a small second line, with its path as tooltip.
- For the owner, outside edit mode, activating a row opens the note's editor in the same modal as "Recent notes".
- Empty state "No notes in this category"; a deleted category shows "This category no longer exists" to the owner.
- Visitors see only the shown rows, if "Show on public dashboard" is on. Rows can't be activated.

## Capabilities

### New Capabilities
- `note-list-widget`: a dashboard widget that lists the owner's notes of one category (or all, or the uncategorized ones), optionally with its subcategories, sorted and limited by count.

### Modified Capabilities
- `dashboard-editing`: the widget picker's "Notes" group also lists "Note list".

## Impact

- Backend: new widget kind in `backend/internal/widgets`, registered in `widgets.Register`; a notes store query for the notes of a category selection, reusing the recursive subcategory condition of `scope.go`; reuses `notes.DisplayTitle`, `categoryOf` and the recent notes' limit bounds. No migration.
- Frontend: new kind in `frontend/src/features/dashboards/widgets/noteList`, added to the widget registry in the "notes" group; reuses `CategoryPicker`, `filterNotes`, `NoteDialog` and the recent notes' row styling; en + de translations.
- Public dashboard lookup: the widget is left out for visitors unless "Show on public dashboard" is on, and when its category no longer exists.
- Builds on the in-progress `add-recent-notes-widget` change (its code is already committed on this branch).
