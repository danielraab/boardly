# Design

## Context

`ParseKarakeep` decodes `karakeepBookmark` without an `archived` field and turns text bookmarks into `notes.ImportNote` (title, content, creation time) through `karakeepNote`. The note importer (`notes/import.go`, `addNote`) inserts them with version 1 and checks duplicates by title and content. After `note-pin-archive`, the `notes` table has `pinned` and `archived`.

## Goals / Non-Goals

**Goals:**
- Archived Karakeep text bookmarks arrive as archived notes.

**Non-Goals:**
- Pinning imported notes. Karakeep's export (`zExportBookmarkSchema`) has no `favourited`.
- Any archive state for links, or for the Linkwarden and Netscape formats, which have no notes.

## Decisions

**Carry the flag on `ImportNote`.** Add `Archived bool` to `notes.ImportNote` and write it in `addNote`'s insert. The other import formats leave it `false`. Considered: archiving the notes after the import in a second pass. Rejected because it would bump the version and the time of last change, while imported notes should keep version 1 and the bookmark's date.

**Lenient decoding.** `archived` is decoded as an optional boolean; missing, `null` or a value that isn't a boolean counts as `false`. This follows the importer's existing leniency (`optString`, `flexTime`), so a slightly different export never fails the whole import.

**Duplicates ignore the flag.** A note that matches an existing note by title and content is still a duplicate, whether or not either is archived. Re-importing a file never changes notes that already exist.

## Risks / Trade-offs

- [Users who archived bookmarks in Karakeep may not find these notes at first] → They are shown under "Archived" on `/notes`, and the README's import section will mention it.
