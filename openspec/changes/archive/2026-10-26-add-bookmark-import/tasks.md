## 1. Import model and store

- [x] 1.1 Add `ImportTree`, `ImportFolder`, `ImportLink` and `ImportReport` to `internal/favourites`
- [x] 1.2 Implement `Store.Import(ctx, userID, folderID, tree)` in one transaction: check the import folder (root allowed), load existing URLs for the duplicate check, reuse child folders case-insensitively, append nodes after existing children in file order, set `created_at` from the source or now, attach tags with `setTags`, queue fetches with `queueFetches`, and notify after commit
- [x] 1.3 Store tests: import into the root and into a subfolder, folder reuse in another case, same-named siblings merged, duplicates against the tree and within the file, order after existing children, `created_at` kept, existing tag matched in another case, fetch jobs queued, rollback on failure, link or foreign node as import folder

## 2. Parsers (`internal/importer`)

- [x] 2.1 Shared helpers: description composition (`Note: `, `Summary: `, blank-line separated, cut to 2047 runes plus `…`), title and folder-name cutting, dropping invalid tags, URL check via `favourites.NormalizeURL`, and parse counts
- [x] 2.2 Karakeep JSON parser: lists tree by ID (parent may follow child, unknown parent goes to the top level), smart lists skipped and counted, first manual list wins, non-link bookmarks counted as `notLinks`, `createdAt` in Unix seconds, note and summary into the description, subfolders before links
- [x] 2.3 Linkwarden JSON parser: collections tree by `parentId`, links with `name`, `url`, `description`, ISO `createdAt` and `tags[].name`, links without an `http(s)` URL counted as `notLinks`, lenient about missing or extra fields
- [x] 2.4 Netscape HTML parser with the `golang.org/x/net/html` tokenizer: `<H3>` folders with a `<DL>` stack, `<A>` with `HREF`, `ADD_DATE` and `TAGS`, `<DD>` descriptions, junk such as `<script>` ignored, and detection of non-Netscape files
- [x] 2.5 Synthetic fixtures (no personal data) in `internal/importer/testdata`: a Karakeep JSON with nested lists in parent-after-child order, a smart list, a text bookmark, a bookmark in two lists, a note and an archived link; a Karakeep-style flat HTML with `TAGS` and a trailing `<script>`; a browser-style HTML with nested `<H3>` and `<DD>`; a Linkwarden JSON with sub-collections, tags and a PDF link
- [x] 2.6 Parser tests for every format, including invalid files (wrong format, not JSON, empty HTML) returning the "invalid import file" error

## 3. HTTP endpoint

- [x] 3.1 Add `POST /api/favourites/import` (`requireUser`) in `internal/httpapi/import.go`: raw octet-stream body with a 10 MB `MaxBytesReader`, `format` and `folderId` as query parameters (multipart would bypass the CSRF content-type check), dispatch to the parser, call `Store.Import`, and return the report JSON
- [x] 3.2 Map errors to `invalid_request`, `invalid_folder`, `invalid_import_file` and `file_too_large` (413) in the standard error envelope
- [x] 3.3 httptest tests: successful Karakeep, Linkwarden and Netscape imports, wrong format, too large, missing fields, foreign or link folder, form post rejected (403), signed out (401)

## 4. Settings tab and import form

- [x] 4.1 Add the `data` tab ("Import / Export") to `SettingsTab`, `routes/settings.tsx` and `SettingsPage.tsx`, after "Connected apps"
- [x] 4.2 Add `features/settings/import/api.ts` to post the file as an octet-stream body with `format` and `folderId` query parameters, and type the report and error codes
- [x] 4.3 Build `ImportExportTab.tsx`: format radio buttons with `AppLogo` and the Karakeep JSON hint, file input, `FavouritePicker mode="folder"` with no initial selection, an Import button disabled until a file and a folder are chosen, a busy state, and the report (non-zero skip counts only) and error messages
- [x] 4.4 Invalidate the favourites query after a successful import
- [x] 4.5 Add en and de translations (`settings.tabs.data`, `settings.import.*`)
- [x] 4.6 Vitest tests: tab reflected in `?tab=data`, button disabled without a folder, report shown after success, error message for `invalid_import_file`

## 5. Docs and verification

- [x] 5.1 Document the import in README.md (formats, import folder, folder reuse, duplicates, dates, what isn't imported, size limit), and recommend the Karakeep JSON export over HTML
- [x] 5.2 `make check` and `make test` pass
- [x] 5.3 Manual check: import the user's real Karakeep JSON and HTML exports in a local instance, and compare the counts with the expected report
- [ ] 5.4 Verify the Linkwarden parser against a real Linkwarden JSON export (ask the user for one) and adjust the design and spec if the format differs
