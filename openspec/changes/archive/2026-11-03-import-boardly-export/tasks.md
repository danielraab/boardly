# Tasks

## 1. Reading the export

- [x] 1.1 Add `Node.UnmarshalJSON` to `backend/internal/export` so a document round-trips. Verify with a test that marshals and unmarshals the seeded document and compares the trees.
- [x] 1.2 Create `backend/internal/dataimport` with a parser: ZIP by its magic bytes, else JSON; `boardly.json` read through a 50 MB limit; format `boardly` and version 1 only (`ErrInvalidFile` otherwise); a function to read one referenced archive file with a byte limit. Verify with tests: JSON and ZIP accepted, version 2, other formats, broken ZIP, ZIP without `boardly.json`, oversized file entry.

## 2. Existing importers report IDs

- [x] 2.1 Extend `favourites.ImportTx`: optional `SourceID` on folders and links, optional `PreviewImageID` on links, and an `IDs` map in the report (created, reused folders, skipped duplicates → existing link). Verify with new tests for the map, and that all existing favourites and httpapi import tests pass unchanged.
- [x] 2.2 Extend `notes.ImportTx`: optional `SourceID` on categories and notes, optional `Icon`, `Color`, `Pinned` and `UpdatedAt`, and an `IDs` map (created, reused categories, skipped duplicates → existing note). Verify the same way.

## 3. Transaction-aware stores

- [x] 3.1 Split image and attachment saving into validate-and-write-blob and insert-row-in-a-`db.Querier`, reusing `Sniff`/`DetectType` and the allowed types and sizes; existing `Save`/`Create` use the split. Verify with tests for both steps and that existing image and attachment tests pass.
- [x] 3.2 Add creating a dashboard with its layout, and inserting a media source, with a `db.Querier`, keeping the existing validation (`Normalize` per widget, media kind). Verify with store tests inside a rolled-back transaction.

## 4. The import

- [x] 4.1 Implement the plan: chosen parts, the images and attachments they use, reuse of the user's own images and attachments by ID, validated blob writes, and skipped-file counting. Verify with tests: only used images per part, own files reused, a disallowed attachment type and a missing file are skipped and counted, a JSON export imports no files.
- [x] 4.2 Implement the transaction: rows for files, favourites with unused tags, categories and notes with rewritten content and icons, media sources with URL dedup, dashboards with slug suffixes, not public, not default, rewritten widget configs with rejected widgets counted, profile defaults only when unset. Verify with tests for each scenario of the `data-import` spec, including references to reused folders and skipped duplicates, kept references for items not imported, and a rollback leaving no rows.
- [x] 4.3 Implement the generic ID rewrite over JSON and the removal of attachment blocks that weren't imported. Verify with unit tests: nested objects and arrays, non-matching strings untouched, numbers untouched.

## 5. Endpoint

- [x] 5.1 Add `POST /api/import` behind `requireUser` with `parts`, `folderId`, `categoryId`, `skipDuplicates`, the 10 MB limit and the error codes; run `AfterImport` after commit; return the report. Verify with httptest: 200 JSON and ZIP, `invalid_request`, `invalid_import_file`, `invalid_folder`, `invalid_category`, 413, 403 for form posts, 401 signed out and with an app token, and a round trip (export Ann, import into Bob, compare counts).

## 6. Frontend

- [x] 6.1 Add the `importData` API helper and report type for `/api/import`. Verify with a Vitest test for the query string and body.
- [x] 6.2 Rename the section to "Import", add the format "Boardly export (JSON or ZIP)", accept `.zip`, add the part checkboxes with the picker visibility and button rules, show the Boardly report, invalidate all affected queries, and show the 10 MB message with the hint for the Boardly format. Add en/de strings. Verify in `ImportExport.test.tsx` with the scenarios of the "Boardly format in the import form" and "Import form" requirements.

## 7. Documentation and checks

- [x] 7.1 Update README (import of a Boardly export under "Exporting your data" or its own section; remove "not supported yet") and CHANGELOG. Verify by reading the sections.
- [x] 7.2 Run `make check` and `make test` and confirm both pass; run `openspec validate import-boardly-export --strict`.
