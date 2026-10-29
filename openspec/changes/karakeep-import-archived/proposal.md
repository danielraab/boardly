# Proposal

## Why

Karakeep exports say whether each bookmark is archived, but the import ignores it, so archived Karakeep text bookmarks turn into active notes and clutter `/notes`. With `note-pin-archive`, notes can be archived, so the import can keep that state.

## What Changes

- A text bookmark of a Karakeep JSON export with `archived` `true` becomes an archived note, in the same category it would go to anyway.
- Imported notes are never pinned. Karakeep's export has no favourite flag (its export schema only carries `archived`), so `favourited` stays ignored.
- Links stay as they are: the favourites tree has no archive, so `archived` stays ignored for links.
- Duplicate detection doesn't change: it compares title and content only, and a skipped duplicate leaves the existing note's flags as they are.

## Capabilities

### New Capabilities

### Modified Capabilities
- `bookmark-import`: the Karakeep JSON format reads `archived` for text bookmarks, and imported notes carry the archived flag.

## Impact

- Backend: `backend/internal/importer/karakeep.go` reads `archived`; `notes.ImportNote` gets `Archived` and `backend/internal/notes/import.go` stores it. Tests and the `karakeep.json` test data get an archived text bookmark.
- Depends on `note-pin-archive`, which has to be applied first.
- No API, frontend or migration change; the import report stays the same.
