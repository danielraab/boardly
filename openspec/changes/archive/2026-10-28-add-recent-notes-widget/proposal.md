# Proposal

## Why

A dashboard can show a single note, but you can't see at a glance which notes were added or worked on lately. A widget that lists the newest notes, like "Recent favourites" does for links, gives quick access to them from the start page.

## What Changes

- New content widget kind "Recent notes" / "Neueste Notizen" (`recent_notes`) in the "Notes" group of the widget picker. It lists the owner's newest notes across all categories, newest first.
- Settings: "Sort by" with "Added" (creation time, default) or "Last changed" (time of last change), a required number of notes (1–100, default 10), an optional "only the last N days" (1–365), an optional title, a size (S/M/L) and "Show on public dashboard".
- The chosen time drives the order, the day limit and the time shown on each row.
- Each row shows the note's icon, its display title, and the chosen time as a relative time ("3 days ago") with the full date as tooltip. If the note has a category, its title is shown as a small second line, with the category's path as tooltip. The note's colour is not shown.
- For the owner, outside edit mode, activating a row opens the note's editor in the same modal as the note widget's "Open note".
- Empty state "No notes" / "No notes in the last N days".
- Visitors see only the shown rows, if "Show on public dashboard" is on. Rows can't be activated.

## Capabilities

### New Capabilities
- `recent-notes-widget`: a dashboard widget that lists the owner's most recently added or changed notes, limited by count and optionally by age.

### Modified Capabilities
- `dashboard-editing`: the widget picker's "Notes" group also lists "Recent notes".

## Impact

- Backend: new widget kind in `backend/internal/widgets`, registered in `widgets.Register`; a notes store query for the newest notes by `created_at` or `updated_at`; a Go version of the frontend's display title (first line of text) so visitors never get note content. No migration.
- Frontend: new kind in `frontend/src/features/dashboards/widgets/recentNotes`, added to the widget registry in the "notes" group; reuses the note widget's editor modal and the recent favourites' relative-time helpers; en + de translations.
- Public dashboard lookup: the widget is left out for visitors unless "Show on public dashboard" is on.
