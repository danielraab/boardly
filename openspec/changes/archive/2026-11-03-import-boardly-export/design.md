# Design

## Context

- The export (`backend/internal/export`) defines version 1 of the document as Go types. `Node` has a custom `MarshalJSON` but no `UnmarshalJSON` yet.
- `favourites.ImportTx` and `notes.ImportTx` already implement the merge rules: they take a parsed tree, reuse folders and categories by name, skip duplicates, and work in a caller's transaction. They create new IDs but don't tell the caller which new ID a source item got. Their input types also lack fields the Boardly format has (preview image on links; icon, colour, pinned and attachments on notes; icon and colour on categories).
- `images.Store.Save` and `attachments.Store.Create` validate a file (sniffed type, allowed types, maximum size), write the blob and insert the row, but use `s.DB`, not a transaction.
- Both blob directories have an orphan cleanup (`blobs.Dir.RemoveOrphans`) that deletes files without a record after an hour.
- Widget kinds validate only the shape of their config in `Normalize`. Missing targets are handled when a widget is shown, and lookups are always scoped to the owner, so a kept foreign or stale ID is harmless.
- All IDs in the export are UUIDs (`blobs.NewID`/`uuid.NewString`). Numeric IDs used by the app APIs aren't in the export.

## Goals / Non-Goals

**Goals:**
- One import that reuses the existing merge rules instead of duplicating them.
- References that follow imported, reused and skipped items, without per-widget-kind code.
- All database writes in one transaction; files never block or break it.

**Non-Goals:**
- Replacing or deleting data, restoring IDs, merging dashboards.
- Importing anything the export leaves out (passkeys, app tokens, …), or the profile name and slug.
- Larger uploads than 10 MB or streaming the upload to disk.
- Versions other than 1.

## Decisions

### A new package `dataimport` on top of the export types
`backend/internal/dataimport` holds the Boardly import. It parses the body into `export.Document` (adding `Node.UnmarshalJSON` in the export package, so both directions use the same type), builds the plan, writes files and runs the transaction. The existing `importer` package stays for the third-party formats. It produces trees for the existing importers, which is a different job.

*Alternative:* turning the Boardly document into an `importer` tree and going through `POST /api/favourites/import`. Rejected because that tree has no room for dashboards, media or IDs, and the endpoint would grow parameters that only one format uses.

### Reading the body
The first bytes decide the format: `PK\x03\x04` is a ZIP, anything else is parsed as JSON. In a ZIP, `boardly.json` is read through a limit of 50 MB decompressed. A 10 MB ZIP can hold more JSON than that only in an abusive file, which then fails with `invalid_import_file`. Each referenced file is read with a limit of one byte more than this server's maximum for its kind (`images.MaxSize`, the attachment maximum), so a zip bomb can't exhaust memory and an oversized file is skipped. Paths come only from the document's `file` values and are looked up as archive entries, never joined to a disk path.

### Source IDs and an ID map
The import keeps one `map[string]string` from file ID to account ID. The existing importers get an optional `SourceID` on `ImportFolder`, `ImportLink`, `ImportCategory` and `ImportNote`, and report `IDs map[string]string` in their result:

- a created item maps to its new ID,
- a reused folder or category maps to the existing one,
- a skipped duplicate link maps to the existing link with that URL, and a skipped duplicate note to the existing note with that key (both maps of the duplicate check then hold IDs instead of `true`).

Tags map by name to the user's tag or the created one. The other parts (images, attachments, media sources, dashboards, widgets) are created by the import itself and add to the same map.

The existing importers also get the missing fields (`PreviewImageID` on links; `Icon`, `Color`, `Pinned`, `AttachmentIDs` on notes; `Icon`, `Color` on categories), all optional, so the third-party formats are unchanged.

### Order of work
```
1. parse + validate parameters, folder and category        (no writes)
2. choose images and attachments the chosen parts use
3. validate and write their blobs, outside the transaction  (new IDs → map)
4. BEGIN
   a. insert image and attachment rows
   b. favourites tree (folders, links with mapped preview), unused tags
   c. categories and notes (content and icons rewritten through the map)
   d. media sources (dedup by URL, mapped icon)
   e. dashboards with rewritten widget configs, via the layout validation
   f. profile defaults when unset
   COMMIT
5. favourites.AfterImport (background fetches), response
```
Images come first because previews and icons refer to them. Dashboards come last because widgets can refer to everything else. On any error the transaction is rolled back, and the blobs from step 3 are left to the orphan cleanup.

To insert rows in the transaction, the images and attachments stores get `SaveTx`/`CreateTx`-style functions that split validation and blob writing (step 3) from the row insert (step 4a), reusing `Sniff` and `DetectType` so the rules stay identical to uploads. The dashboards store gets a way to create a dashboard and save its layout with a `db.Querier`, and the media store a way to insert a source with a `db.Querier`.

### Own images and attachments are reused
An image or attachment whose ID already belongs to the importing user maps to itself and isn't imported again. Note content holds attachment IDs, so this is what lets a re-import into the same account find duplicate notes. It also avoids copying every icon on each re-import. IDs of other users' files are never matched, because only the user's own rows are looked up.

### Generic reference rewriting
Widget configs and note content are rewritten by walking the decoded JSON and replacing every string value that is a key of the ID map. Strings that aren't keys stay unchanged, which keeps references to items that weren't imported. Because IDs are UUIDs, a real text value equal to an ID is practically impossible. After rewriting:

- note content is validated like any note. Attachment blocks whose ID isn't one of the imported attachments are dropped with the existing document helpers, before validation,
- icons (categories, notes, media) are kept only if their image was imported, and otherwise set to none,
- widget configs go through the kind's `Normalize` as part of saving the layout. A widget it rejects, or of an unknown kind, is left out and counted.

*Alternative:* a `RemapIDs` hook per widget kind. Rejected because every new kind would have to remember it, and the generic walk covers note content too.

### Dashboards
Fields from the file: title, columns, widgets. `is_public` and `is_default` are always false. The slug is normalized with `slugs.Normalize`. If it's invalid, a generated slug is used. If it's taken, `-2`, `-3`, … are tried against the user's slugs, including those added earlier in the same import.

### Endpoint and errors
`POST /api/import` in `router.go`, behind `requireUser` (session only) and the same body handling as the bookmark import (`application/octet-stream`, `http.MaxBytesReader` with 10 MB → `file_too_large`). The report is JSON with one object per part, `null` for parts not chosen, plus `files`.

### Frontend
The import section's format list gets "Boardly export (JSON or ZIP)". For it, the form shows the part checkboxes and hides or shows the pickers. It calls a new `importData` helper for `/api/import`, and on success invalidates the favourites, tags, dashboards, notes, categories and media query keys. The `file_too_large` message gets the extra hint when the Boardly format is chosen.

## Risks / Trade-offs

- [Changing the existing importers could change the third-party imports] → New fields are optional and default to today's behaviour. The existing importer tests must keep passing unchanged.
- [A widget kind that stores something ID-like that isn't an ID] → Only exact matches of keys in this import's map are replaced. Such a value would have to equal a UUID from the same file.
- [The 10 MB limit makes ZIPs with many files fail] → Accepted for now. The message points to the JSON export, and attachments can be added again by hand.
- [Long transaction on a big import] → SQLite with WAL keeps readers working. The import is bounded by the 10 MB body.
- [Blobs written for an import that then fails] → Removed by the existing orphan cleanup after an hour, the same as a failed upload.
- [A crafted file referencing another user's IDs] → Kept IDs are only looked up within the importing user's data, and icons are only kept for images imported in the same run, so nothing of another user becomes visible.
