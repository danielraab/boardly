# Proposal

## Why

The rows of the note widgets are plain text lines and look unlike notes everywhere else. Showing each row as a small card in the note's colour makes the widgets match the note cards and lets colour-coded notes be told apart at a glance.

## What Changes

- Every row of "Recent notes", "Note list" and "Selected notes" gets a border, rounded corners and the note's colour as background (`surface` without a colour), like a note card. Rows are separated by a small gap.
- The resolved rows carry the note's colour, for the owner and for visitors of a public dashboard.

## Capabilities

### New Capabilities

### Modified Capabilities
- `recent-notes-widget`: rows look like small note cards in the note's colour; visitors also get the colour.
- `note-list-widget`: the same.
- `selected-notes-widget`: the same.

## Impact

- Backend: `color` in the recent notes and listed note row structs in `backend/internal/widgets`. No migration.
- Frontend: the shared `NoteRows` component renders rows as cards with `noteBackground`; the row types get `color`.
- Public dashboard lookup: visitors now see each shown note's colour.
