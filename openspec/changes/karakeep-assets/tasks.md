# Tasks

## 1. REST endpoints

- [ ] 1.1 Add a `karakeepREST` wrapper in `httpapi/karakeep.go` that writes errors as `{"error": msg}` with the status, and verify with a handler test that a token-less request to `/api/karakeep/api/assets` gets 401
- [ ] 1.2 `POST /api/karakeep/api/assets`: check the note scope, cap the body at max + 64 KB, stream the `file` part through `ReadLimited`, store via `attachments.Store.Create`, and respond `{assetId, contentType, size, fileName}`; verify the "Share a photo", "Too large" (413), "Type not allowed" and "Token without notes" scenarios, plus a missing `file` part
- [ ] 1.3 Extract the attachment response code of `GET /api/attachments/{id}/{name}` into a shared helper and verify the existing attachment serving tests still pass
- [ ] 1.4 `GET /api/karakeep/api/assets/{id}`: scope check (owner, and referenced in scope or unreferenced), then the shared helper; verify "View a shared photo", "Attachment outside the scope" and another user's ID (404)

## 2. Asset bookmarks

- [ ] 2.1 Add the asset content type and `Asset` type for `Bookmark.Assets`, and map a note with exactly one image or PDF attachment block to an asset bookmark in `noteBookmark`; verify the "Shared photo", "PDF", "Photo with text" and "Video" scenarios in `karakeep/notes_test.go`
- [ ] 2.2 Add the `asset` branch to `createBookmark` (resolve `assetId`, title from `title` or the file name cut to the limit, `archived` and `favourited`, ignore `assetType`, `fileName` and `sourceUrl`); verify "Save a file", "Unknown file", note scope `none`, and a 250-character file name
- [ ] 2.3 Verify update and delete on asset bookmarks: "Rename a shared file" stays an asset bookmark, setting `text` turns it into a text bookmark, and "Delete a shared file" leaves the attachment in place
- [ ] 2.4 Reword the "file bookmarks" unsupported message, and remove the "Boardly doesn't store files" wording from the code and comments; verify `go vet ./...` and `go test ./...` pass

## 3. Docs and check

- [ ] 3.1 Add to the README's Karakeep section that photos and PDFs shared to the Karakeep app become notes with the file
- [ ] 3.2 Manual check with the Karakeep mobile app against a local server: share a photo and a PDF, see both open in the app and as notes on `/notes`, rename one, and delete one
