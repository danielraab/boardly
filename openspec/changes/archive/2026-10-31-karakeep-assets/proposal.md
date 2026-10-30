# Proposal

## Why

Sharing a photo or PDF to Karakeep's mobile app fails against Boardly with "Boardly doesn't store files", but Boardly has had attachments since `add-note-attachments`. Text shared from Karakeep already becomes a note, and a shared file should too.

## What Changes

- New `POST /api/karakeep/api/assets` (multipart, outside tRPC) stores the uploaded file as an attachment of the token's owner, with the same limits as web uploads, and answers in Karakeep's upload shape.
- `bookmarks.createBookmark` with `type` `asset` creates a note in the token's root category with the file as its only content and the file name as its title, instead of failing.
- A note whose content is exactly one image or PDF attachment is returned as an asset bookmark, so the app shows the image or PDF. Every other note stays a text bookmark.
- New `GET /api/karakeep/api/assets/{id}` serves the file to the app with the token, for the app's image and PDF viewers.
- Updating such a bookmark's title, `archived` and `favourited` works as for text bookmarks. Deleting it deletes the note and leaves the file to the attachment cleanup.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `karakeep-api`: asset bookmarks are created as notes and read back from them; new upload and file endpoints; "Notes are text bookmarks" and "Create bookmarks" change.

## Impact

- Backend: `internal/karakeep` (asset create, bookmark mapping), `internal/httpapi/karakeep.go` (two REST routes next to tRPC), reuse of `internal/attachments` for storing and serving.
- No migration, no frontend change. README's Karakeep section gets a line about sharing files.
- Companion change: `memos-attachments`.
