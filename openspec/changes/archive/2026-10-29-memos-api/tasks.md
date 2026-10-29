# Tasks

## 1. Prerequisite

- [x] 1.1 Confirm `note-pin-archive` is applied (notes have `Pinned` and `Archived`); verify `go test ./internal/notes/...` passes

## 2. Memos app tokens

- [x] 2.1 Add `backend/internal/db/migrations/0021_memos_app_tokens.sql`, which rebuilds `app_tokens` with `memos` allowed and keeps every column and row; verify the db migration test passes with existing Karakeep and Linkwarden tokens
- [x] 2.2 Add `apptokens.AppMemos`: create without a root folder (reject `rootId` with `invalid_root`), note scope from `rootCategoryId`, edit only the note scope; verify with apptokens and httptest cases for "Memos token", "Memos token with a root category", "Root folder for Memos" and "Change the category of a Memos token"
- [x] 2.3 Allow `PATCH` in the app endpoints' preflight; verify the CORS test for "Preflight for an update"

## 3. Memos API foundation

- [x] 3.1 Create `backend/internal/memos` with the Memos error type, millisecond UTC time formatting and the memo and user shapes, mounted at `/api/memos/` through `appAPI`; verify with tests for "Error format", "Legacy status endpoint" and "Memos endpoint with a Karakeep token"
- [x] 3.2 Implement `instance/profile` (public, `0.31.0`), `auth/me`, `users/{id}`, `users/{id}/settings/GENERAL` and `users/{id}:getStats`; verify with tests for "Connect", "Log in" and "Other user"

## 4. Memos as notes

- [x] 4.1 Implement memo rendering (title as a leading `# Title`, state, pinned, empty lists) and the leading-heading parser for title and content; verify with unit tests for "Titled note", "Untitled note", "Pinned and archived", "Rename by heading", "Heading removed", "Second-level heading" and "Round trip"
- [x] 4.2 Implement `GET memos` with `state`, `pageSize`, an opaque `pageToken` and the two recognised filter forms; verify with tests for "Sync", "Archive list", "Explore", "Notes outside the scope" and an invalid page token
- [x] 4.3 Add an optional `CreatedAt` to `notes.NewNote` (clamped to now) and implement `POST memos` in the token's root category; verify with tests for "Write a memo", "Written offline" and "Memo with a photo"
- [x] 4.4 Implement `PATCH memos/{id}` with `updateMask`, content, pinned, state and attachments rules, patching against the current version; verify with tests for "Tick off a task", "Pin", "Archive", "Attachment on update", unchanged content not bumping the version, and the web editor getting `version_conflict` afterwards
- [x] 4.5 Implement `DELETE memos/{id}` and the attachment endpoints; verify with tests for "Delete", "Note outside the scope", "Sync without attachments" and "Upload a photo"

## 5. Frontend

- [x] 5.1 Add "Memos" to the create dialog (no folder, root category, MoeMemos hint, server address `<BOARDLY_BASE_URL>/api/memos`); verify with tests for "Memos without a folder" and the updated "Root category only for Karakeep"
- [x] 5.2 Show Memos tokens in the list with the Memos logo, the category and no folder, and give the edit dialog only the category choice; add the logo with its source in `logos/README.md`; verify with tests for "Memos token in the list" and the edit dialog's "Memos token"
- [x] 5.3 Add the new en/de strings; verify the i18n test passes

## 6. Wrap-up

- [x] 6.1 Add a "Memos-compatible server" section to the README (MoeMemos setup, what syncs, the limits on attachments and visibility, moving notes out of the category); verify the README describes every limit in the proposal
- [x] 6.2 Run `make check` and `make test` and verify both pass
- [x] 6.3 Manual check with MoeMemos on an Android device or emulator against a dev server: log in, sync, create, tick off a task, pin, archive and delete; verify each change shows on `/notes`
