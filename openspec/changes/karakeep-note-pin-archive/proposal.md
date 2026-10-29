# Proposal

## Why

Once `note-pin-archive` gives notes a pinned and an archived flag, the Karakeep apps still show every text bookmark as neither favourited nor archived, and they reject archiving and favouriting. Mapping the flags lets people archive and favourite notes from the Karakeep apps and see the same state as in Boardly.

## What Changes

- Text bookmarks (notes) report `archived` from the note's archived flag and `favourited` from its pinned flag.
- `bookmarks.createBookmark` of type `text` and `bookmarks.updateBookmark` on a note accept `archived` and `favourited` and set the note's flags. Changing them counts as a change of the note.
- `bookmarks.getBookmarks` filters by `archived` and `favourited`: `true` returns only notes with that flag, `false` leaves out notes with it, and links count as neither archived nor favourited.
- `users.stats` counts archived and pinned notes as archived and favourite bookmarks.
- Links don't change: they are never archived or favourited, and setting either to `true` on a link still fails with `BAD_REQUEST`.

## Capabilities

### New Capabilities

### Modified Capabilities
- `karakeep-api`: notes as text bookmarks carry `archived` and `favourited`; create, update, list and statistics handle them for notes.

## Impact

- Backend: `backend/internal/karakeep` (`bookmarks.go`, `karakeep.go`, `user.go`) and its tests. It uses the note store's `Pinned` and `Archived` fields from `note-pin-archive`.
- Depends on `note-pin-archive`, which has to be applied first.
- No frontend, migration or import change. The Karakeep import keeps ignoring `archived` and `favourited`.
