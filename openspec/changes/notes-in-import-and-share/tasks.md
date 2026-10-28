# Tasks

## 1. Converter

- [x] 1.1 Add `github.com/yuin/goldmark` and `notes.FromMarkdown` (headings, paragraphs with line breaks, quotes, code, dividers, bullet/ordered/task lists with depth limit, marks, links with allowed schemes, linkify, images/tables/HTML as text); verify with table tests for each scenario of "Content from Markdown" and that every result passes `ParseDoc`
- [x] 1.2 Add `notes.FromText` (blank-line paragraphs, line breaks, `http(s)` links, no Markdown); verify with tests for the "Content from plain text" scenarios
- [x] 1.3 Add a fuzz test that `FromMarkdown` output below the size limit is always accepted by `ParseDoc`; verify `go test -run Fuzz` passes with the seed corpus

## 2. Notes API and default category

- [x] 2.1 Accept `text` on `POST /api/notes` (both `content` and `text` or non-string `text` → 400 `invalid_request`, too large → `invalid_document`); verify with httptest cases
- [x] 2.2 Add migration `0018_note_category_default.sql` (`users.note_category_id … ON DELETE SET NULL`), `User.NoteCategoryID` and `Patch.NoteCategoryID` with `ErrInvalidCategory`; verify with users tests (own, other user's, unknown, null) and that deleting the category clears it
- [x] 2.3 Accept `noteCategoryId` in `PATCH /api/me` (→ 400 `invalid_category`); verify with httptest cases

## 3. Import backend

- [x] 3.1 Add `notes.ImportTree`/`ImportCategory`/`ImportNote` and make `importer.Parse` return `importer.Result{Links, Notes}`; update Linkwarden and Netscape parsers (no notes) and existing tests
- [x] 3.2 Read Karakeep text bookmarks into notes: title, Markdown content, `Note: `/`Summary: ` paragraphs, tag line, creation time, too-large count; build categories from manual lists and prune lists without notes; verify with importer tests for each scenario of "Imported notes" and "Categories from lists"
- [x] 3.3 Add `notes.Store.ImportTx` (categories reused by title ignoring case, notes in file order with created/updated time, duplicate notes by title + content without task IDs when `skipDuplicates`) and a category check; verify with store tests (reuse, top level, duplicates within file and existing, duplicates allowed)
- [x] 3.4 Split `favourites.Store.Import` into `ImportTx` plus `AfterImport`, and run links and notes in one transaction in `importFavourites` with `categoryId` (`invalid_category`) and the combined report; verify with httptest cases for the endpoint scenarios, the report and a rollback when writing notes fails

## 4. Frontend

- [x] 4.1 Add `noteCategoryId` to the `User` type and fixtures, and a `CategoryPicker` (tree with indented options, configurable empty choice); verify with a Testing Library test of the options and selection
- [x] 4.2 Add `NoteCategoryForm` to the Profile tab (save via `PATCH /api/me`, "No category" → `null`, error message); verify with Profile tests
- [x] 4.3 Import form: category picker for Karakeep only, `categoryId` sent only then, new checkbox label, report with notes, categories, duplicate notes and too-large notes, invalidate notes and categories queries; verify with ImportExport tests (Karakeep vs Linkwarden, report, request parameters)
- [x] 4.4 Share page: initial mode and note text in `sharedLink.ts`, link/note switch keeping both forms' values, note form (title, text, category from default), save via `POST /api/notes` with `text`, confirmation with "Open note"; verify with sharedLink and SharePage tests for each scenario of "Save link page" and "Save note from a share"
- [x] 4.5 Add all en + de strings and verify the i18n key parity test passes

## 5. Docs and check

- [x] 5.1 Update README (Importing bookmarks, Notes, Install as an app → share, Settings → Profile) and CHANGELOG
- [x] 5.2 Run `make check` and `make test` and verify both pass
- [ ] 5.3 Run the app, import a Karakeep export with text bookmarks in nested lists into a category twice (with and without skipping duplicates), and share text with and without a URL from a phone or with `/share?…` by hand
