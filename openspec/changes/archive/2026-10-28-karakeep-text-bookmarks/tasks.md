# Tasks

Requires `notes-in-import-and-share` to be implemented first (`FromMarkdown`, `CategoryPicker`).

## 1. Note content as Markdown

- [x] 1.1 Add `notes.ToMarkdown` (all blocks, nesting, marks, links, breaks, escaping, code fences longer than backtick runs); verify with table tests for the "Content as Markdown" scenarios
- [x] 1.2 Add a round-trip test and a fuzz test: `FromMarkdown(ToMarkdown(d))` equals `d` ignoring task IDs for documents without the listed exceptions, and the exceptions behave as specified
- [x] 1.3 Add `notes.ReplaceContentFromMarkdown` with task matching by trimmed text in document order; verify with tests for reordered, renamed and repeated tasks

## 2. Token note scope

- [x] 2.1 Add migration `0019_app_token_note_scope.sql` (`note_scope` with CHECK and default, `root_category_id … ON DELETE SET NULL`, Karakeep tokens → `all`); verify with a migration test on a database with a Karakeep and a Linkwarden token
- [x] 2.2 Add `NoteScope`/`RootCategoryID` to `apptokens.Token` and `NewToken`, validate on create (`invalid_root_category`, Linkwarden with category); verify with store and httptest cases for each create scenario
- [x] 2.3 Add `apptokens.Store.Update` and `PATCH /api/app-tokens/{id}` (folder and category, inactive → 409, unknown fields → `invalid_request`, 404); verify with httptest cases for each "Edit an app token" scenario, including that the next Karakeep request uses the new folder
- [x] 2.4 Revoke active tokens rooted at a deleted category in `notes.Store.DeleteCategory`; verify with a store test (root category deleted → revoked and listed as `category` + `null`; subcategory root → still active; expired token untouched)
- [x] 2.5 Add `notes.Scope`, `ScopedNotes` and `ScopedNote` (recursive category subtree, `all`, `none`); verify with store tests for the "Token subtree scope" note scenarios

## 3. Karakeep API

- [x] 3.1 Introduce `karakeep.Scope{Links, Notes}`, build it from the token in `httpapi/karakeep.go`, and pass it to every procedure; existing tests keep passing
- [x] 3.2 Note bookmark shape and `getBookmark` for notes; verify with tests for "Notes are text bookmarks"
- [x] 3.3 Merge notes into `getBookmarks` (sort, cursor, list/tag filters drop notes) and `searchBookmarks` (title and Markdown text, `#` terms only links); verify with tests for the listing and search scenarios
- [x] 3.4 `createBookmark` with `type: text` into the root category (`all` → uncategorized, `none` → `BAD_REQUEST`, too large → `BAD_REQUEST`), `asset` still rejected; verify with tests
- [x] 3.5 `updateBookmark` on notes (title, text with task matching, unchanged text skipped, `url`/`description` on notes and `text` on links rejected, version conflict → `CONFLICT`) and `deleteBookmark` on notes; verify with tests for each scenario
- [x] 3.6 Notes in `updateTags` (empty ok, otherwise `BAD_REQUEST`), `getListsOfBookmark` (empty), `addToList`/`removeFromList` (`BAD_REQUEST`) and `users.stats` (counts and activity); verify with tests

## 4. Frontend

- [x] 4.1 Add `noteScope`/`rootCategoryId` to the app token type and fixtures; category picker ("All notes") in `CreateAppTokenDialog` for Karakeep only; verify with ConnectedApps tests (Karakeep shows it, Linkwarden doesn't, request body)
- [x] 4.2 Show the root category path ("All notes", "Deleted category") on the Connected apps tab and add "Edit" for active tokens with `EditAppTokenDialog` (folder, category for Karakeep, only changed values sent, error kept in dialog); verify with tests for each "Connected apps tab" and "Edit dialog" scenario
- [x] 4.3 Connected-apps warning in `DeleteCategoryDialog` (count of active tokens rooted at the category) and token query invalidation after delete; verify with a Testing Library test
- [x] 4.4 Add all en + de strings and verify the i18n key parity test passes

## 5. Docs and check

- [x] 5.1 Update README (Karakeep-compatible server: text bookmarks, root category, editing connected apps; Notes) and CHANGELOG
- [x] 5.2 Run `make check` and `make test` and verify both pass
- [x] 5.3 With the Karakeep mobile app connected: save a text note, see it on `/notes` in the root category, tick a task on `/notes`, edit the text in the app and check that the task keeps its ID and the app's checked state, change the token's root category in settings and check that the app sees the new category's notes
