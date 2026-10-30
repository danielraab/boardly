# Tasks

## 1. Notes helpers

- [x] 1.1 Add `SplitTrailingAttachments` and `JoinTrailingAttachments` on `notes.Doc` (run after the last non-attachment block, duplicates once, join appends each ID once) and verify with table tests: photo at the end, attachment in the middle, only attachments, duplicates, empty doc
- [x] 1.2 Add `CreatedAt` (`json:"-"`) to `NoteAttachment`, load it in `noteAttachmentsColumn`, and verify the notes tests pass and `GET /api/notes/{id}` JSON is unchanged

## 2. Memo mapping

- [x] 2.1 Add the memo attachment type (`name`, `createTime`, `filename`, `type`, `size` as string, `externalLink`, optional `memo`) and make `MemoOf`/`Content` use the split; verify with `memos_test.go` cases for "Photo at the end", "Attachment in the middle", "Only attachments" and the `externalLink` URL escaping
- [x] 2.2 Parse `attachments` in create and update as `[{name}]`, resolve `attachments/<id>` against the owner's attachments, and respond 400 code 3 "Unknown attachment." otherwise; verify with tests for another user's ID and a malformed name
- [x] 2.3 Create: join the listed attachments after the content blocks; verify "Memo with a photo" end to end in `httpapi/memos_test.go` (upload, create, note has the block, memo lists it)
- [x] 2.4 Update: merge body and run as in design.md (content only, attachments only, both, empty list) with the content-equality check against the split content; verify the "Attachment on update", "Remove a photo", "Edit text only", "Unchanged memo" (version unchanged) and "Inline attachment kept" scenarios

## 3. Attachment endpoints

- [x] 3.1 `POST /api/v1/attachments`: body limit of base64(max) + 64 KB, decode standard or URL-safe base64, store via `attachments.Store.Create`, map name, type and size errors to 400 code 3; verify the "Upload a photo", "Too large" and "Type not allowed" scenarios, and an oversized body that stops reading
- [x] 3.2 `GET /api/v1/attachments`: attachments referenced by notes in scope, once each, newest first then by ID; verify "Scoped list"
- [x] 3.3 `DELETE /api/v1/attachments/{id}`: `{}` for the owner's attachments that are in scope or unreferenced, otherwise 404 code 5, and never delete; verify "Remove a photo in MoeMemos", "Used in another note" and "Attachment outside the scope"
- [x] 3.4 Remove `ErrAttachments` and the "not supported" handlers and comments (package doc in `memos.go`, the comment in `httpapi/memos.go`), and verify `go vet ./...` and `go test ./...` pass

## 4. Docs and check

- [x] 4.1 Update the README's MoeMemos section so it says attachments are supported, and that files at the end of a note show as the memo's attachments
- [x] 4.2 Manual check with MoeMemos against a local server: attach a photo to a new memo, see it in `/notes/<id>`, add a file at the end of a note in the web app and see it in MoeMemos after a sync, then remove it in MoeMemos
