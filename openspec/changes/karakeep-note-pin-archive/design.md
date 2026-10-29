# Design

## Context

`backend/internal/karakeep` maps notes to text bookmarks (`noteItems`, the bookmark struct in `karakeep.go`). `getBookmarks` builds one item list of links and scoped notes and currently empties it when `archived` or `favourited` is `true`. Create and update collect unsupported fields (`"archiving"`, …) into a `BAD_REQUEST`. After `note-pin-archive`, `notes.Note` has `Pinned` and `Archived`, and `NewNote` and `NotePatch` accept them.

## Goals / Non-Goals

**Goals:**
- Notes round-trip archive and favourite through the Karakeep apps.

**Non-Goals:**
- Archiving or favouriting links. The favourites tree has no such state, and the user scoped this change to notes.
- Karakeep's search qualifiers `is:archived` and `is:fav` (qualifiers stay unsupported, see "Search bookmarks").
- The Karakeep import.

## Decisions

**`favourited` maps to pinned.** Karakeep has no pin; its favourite is the closest "keep this at hand" flag. Memos (in the later `memos-api` change) maps its own `pinned` to the same flag, so all apps agree.

**Filter on the combined item list.** Each item gets `archived` and `favourited` (always `false` for links). `getBookmarks` drops items that don't match a given filter before sorting and paging, replacing today's "empty the list" shortcut. `archived: false` therefore keeps links. Cursor paging stays correct because filtering happens before the cursor is applied.

**Field checks depend on the bookmark type.** The unsupported-field check for `archiving` and `favouriting` only runs for links. For notes, the values go into `NewNote` or `NotePatch`. A note update that only changes flags still goes through `UpdateNote` with the current version, so it counts as a change like everything else.

## Risks / Trade-offs

- [The Karakeep apps may hide archived bookmarks from their home list depending on `archiveDisplayBehaviour`] → `users.settings` keeps `show`, so archived notes stay visible there, matching today's behaviour. Changing it is left to a later change if needed.
- [A user favourites a link in the app and gets an error] → Same as today; the message says Boardly doesn't support archiving or favouriting links.
