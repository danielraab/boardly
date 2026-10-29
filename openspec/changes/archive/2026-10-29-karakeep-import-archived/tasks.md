# Tasks

## 1. Prerequisite

- [x] 1.1 Confirm `note-pin-archive` is applied (the `notes` table has `archived`); verify `go test ./internal/notes/...` passes

## 2. Import

- [x] 2.1 Add `Archived` to `notes.ImportNote` and write it in the note importer's insert, with version 1 and the bookmark's time unchanged; verify with a notes import test for an archived note
- [x] 2.2 Decode `archived` leniently in `karakeepBookmark` and pass it to `ImportNote` for text bookmarks only; add archived text bookmarks (including a non-boolean `archived`) to the inline export of `TestKarakeepNotes`, so the shared `testdata/karakeep.json` and its HTTP import counts stay unchanged, and verify with importer tests for "Archived text bookmark", "Archived bookmark" (link unchanged) and "Archived note"
- [x] 2.3 Verify the "Archived duplicate" scenario with a test: the duplicate is skipped and the existing note stays unarchived

## 3. Wrap-up

- [x] 3.1 Mention in the README's import section that archived Karakeep text bookmarks become archived notes, then run `make check` and `make test` and verify both pass
