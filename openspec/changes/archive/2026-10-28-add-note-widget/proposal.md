# Proposal

## Why

Notes with todos live only on `/notes`, so a shopping list or a sprint checklist can't sit on a dashboard next to the links it belongs to. A widget that shows one note, with tickable todos, makes a dashboard a real start page. With a fifth widget kind of a different family, the flat widget picker also needs grouping.

## What Changes

- New content widget kind `note` ("Note" / "Notiz"): shows one of the owner's notes like a note card, with its colour, icon, title, content and category badge.
- Settings: a required note (plain select of the owner's notes by display title), a size S/M/L (default M) that scales the text, a "Show on public dashboard" flag (default off), and the common width and height (auto or fixed rows; a fixed height scrolls).
- For the owner outside edit mode: task checkboxes toggle and save right away, and an "Open note" button opens the note's editor in a modal over the dashboard. No inline text editing on the widget itself.
- Visitors of public dashboards see the widget only with the flag on, and only read-only; the response contains only what the widget shows.
- A deleted note shows "This note no longer exists" to the owner and is left out for visitors.
- The widget picker groups the kinds under headings: "Favourites" (favourite explorer, link, link list, recent favourites) and "Notes" (note).

## Capabilities

### New Capabilities
- `note-widget`: the note widget's settings, rendering, sizes, todo toggling, "Open note", deleted note and public-dashboard behaviour.

### Modified Capabilities
- `dashboard-editing`: "Add widgets" lists the kinds grouped under translated headings instead of as one flat list.

## Impact

- Backend: new kind in `backend/internal/widgets` (config validation and per-viewer resolve of the note), registered in `widgets.Register`, which now also needs the notes store (`main.go`, test setup).
- Frontend: new `widgets/note` kind (view, settings), a `group` on `WidgetKind` and grouped rendering in `WidgetPicker`, a size option for `NoteRenderer`, en/de strings.
- API: the dashboard lookup carries note data for note widgets. No new endpoints; toggling uses the existing `PATCH /api/notes/{id}`.
