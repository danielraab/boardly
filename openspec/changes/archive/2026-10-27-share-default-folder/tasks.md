# Tasks

## 1. Backend

- [x] 1.1 Add migration `0016_share_folder.sql` with `users.share_folder_id TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL`; verify the server starts and `go test ./internal/db/...` passes
- [x] 1.2 Add `ShareFolderID *string` (`json:"shareFolderId"`) to `users.User` and the columns/scan in `internal/users`; verify `GET /api/me` returns `shareFolderId: null` for a new user in a test
- [x] 1.3 Add `Patch.ShareFolderID` and `ErrInvalidFolder` to `users.Store.Update`: own folder → stored, root → `NULL`, unknown/link/other user's node → `ErrInvalidFolder`; verify with users_test cases for each
- [x] 1.4 Accept `shareFolderId` in `PATCH /api/me` (non-string/non-null and `ErrInvalidFolder` → 400 `invalid_folder`; include it in the "nothing to update" check); verify with httptest cases in `auth_test.go`
- [x] 1.5 Test that deleting the default folder (directly and via its parent) clears `shareFolderId`

## 2. Frontend

- [x] 2.1 Add `shareFolderId: string | null` to the `User` type and test fixtures; verify `pnpm tsc` / `make check` passes
- [x] 2.2 Add `ShareFolderForm` to the Profile tab using `FavouritePicker` in folder mode, saving via `PATCH /api/me` (root → `null`) and updating the `me` query; verify with Profile.test.tsx (choose folder, back to root, error)
- [x] 2.3 Add en + de strings (`profile.shareFolder*`) and verify the i18n key parity test passes
- [x] 2.4 Make `SharePage` preselect `me.shareFolderId` when it is a folder in the tree, otherwise root; remove the localStorage helpers and `SHARE_FOLDER_STORAGE_KEY`; update SharePage.test.tsx (default preselected, other folder not remembered, missing default → root)

## 3. Docs and verification

- [x] 3.1 Update README (Save link page, Settings → Profile) and CHANGELOG with the new setting and the dropped "last used" behaviour
- [x] 3.2 Run `make check` and `make test` and verify both pass
