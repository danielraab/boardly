# Tasks

## 1. Prerequisite

- [ ] 1.1 Confirm `note-pin-archive` is applied (notes have `Pinned` and `Archived` in the store); verify `go test ./internal/notes/...` passes

## 2. Bookmark shape and listing

- [ ] 2.1 Return `archived` and `favourited` from the note's flags in text bookmarks; verify with tests for "Pinned and archived note" and the existing "Note as a bookmark"
- [ ] 2.2 Replace the "empty the list" shortcut in `bookmarks.getBookmarks` with per-item filtering for `archived` and `favourited` (`true` and `false`); verify with tests for "Archived bookmarks", "Not archived", "Favourites" and paging over a filtered list

## 3. Create and update

- [ ] 3.1 Accept `archived` and `favourited` for `text` bookmarks in `bookmarks.createBookmark` and keep rejecting `true` for links; verify with tests for "Save a favourite note" and "Save an archived link"
- [ ] 3.2 Accept `archived` and `favourited` on notes in `bookmarks.updateBookmark` through `NotePatch`, keeping the link check; verify with tests for "Archive a note" (version goes up), "Unfavourite a note" and "Archive a bookmark"

## 4. Statistics and wrap-up

- [ ] 4.1 Count pinned notes as `numFavorites` and archived notes as `numArchived` in `users.stats`; verify "Favourites and archived in the statistics"
- [ ] 4.2 Update the README's Karakeep section, then run `make check` and `make test` and verify both pass
