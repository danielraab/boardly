# Proposal

## Why

Notes can't be kept at the top or put away once they are done, so the notes page keeps growing and important notes get lost among the rest. Pinning and archiving are also what the Memos app (MoeMemos) and the Karakeep apps expect, so the later `memos-api` and `karakeep-note-pin-archive` changes need these flags in Boardly's notes first.

## What Changes

- Notes get two flags, **pinned** and **archived**, both off by default. The notes API returns them and accepts them on create and on change. Changing a flag is an edit: the version goes up and the time of last change is set.
- `/notes`: pinned notes come first, and the chosen category still applies, so a pinned note of another category isn't shown. Pinned cards show a pin icon.
- `/notes`: archived notes are left out of "All notes", "Uncategorized" and the categories, and their counts. A new "Archived" / "Archiviert" entry in the category side panel shows them.
- Note widgets ("Recent notes", "Note list", "Selected notes" and the note widget): archived notes are still shown but marked as archived; within what the widget shows, pinned notes come first and show a pin icon.
- The note menu of the editor (page and dashboard modal) gets "Pin" / "Unpin" and "Archive" / "Unarchive" above "Delete note".
- The Karakeep and Linkwarden APIs don't change. The Karakeep API keeps reporting notes as not archived and not favourited until `karakeep-note-pin-archive`.

## Capabilities

### New Capabilities

### Modified Capabilities
- `notes`: the note entity and API get `pinned` and `archived`; the notes page puts pinned notes first and leaves archived ones out; cards show the pin icon and archived mark; the editor menu gets the pin and archive actions.
- `note-categories`: the side panel gets the "Archived" entry, and its counts leave archived notes out.
- `recent-notes-widget`: pinned notes first, pin icon and archived mark on rows, also for visitors.
- `note-list-widget`: the same.
- `selected-notes-widget`: the same, with pinned notes before the list order.
- `note-widget`: the note shows the pin icon and archived mark.

## Impact

- Backend: a migration adds `pinned` and `archived` to `notes`; `backend/internal/notes` reads, validates and writes them; the widget row queries in `backend/internal/notes` and `backend/internal/widgets` order pinned first and return both flags, also in the public dashboard lookup.
- Frontend: note types and API; `NotesPage`, `SidePanel`, `NoteCard`, `NoteEditorPage` (menu), `NoteRows` and the note widget; new en/de strings.
- No change to the Karakeep, Linkwarden or import APIs.
