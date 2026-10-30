# Proposal

## Why

Notes can have attachments now, but MoeMemos, the main mobile client for Boardly's notes, can't add any: uploading a photo or file fails with "Boardly doesn't support attachments yet", and memos always come with an empty attachment list, so MoeMemos only shows attachments as long links in the text.

## What Changes

- `POST /api/memos/api/v1/attachments` stores the uploaded file (base64 in JSON, as MoeMemos sends it) as an attachment of the token's owner, with the same size and type limits as uploads in the web app.
- A memo's `attachments` are the attachment blocks at the end of its note ("trailing attachments"). They are left out of the memo's `content`. Attachments elsewhere in the note stay in `content` as Markdown links, as today.
- Creating a memo with `attachments`, or updating its `attachments`, sets the note's trailing attachments to that list. Content and attachments can be changed together or separately.
- Each attachment is returned with `externalLink` set to its Boardly URL, so MoeMemos shows and downloads it from there.
- `GET /api/memos/api/v1/attachments` lists the attachments of the notes in the token's scope.
- `DELETE /api/memos/api/v1/attachments/{id}` succeeds without deleting the file. Unused files are deleted by the existing cleanup, so a file that is still used in another note isn't lost.
- **BREAKING** for clients relying on the old behaviour: attachments at the end of a note no longer appear in the memo's `content`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `memos-api`: memos carry their trailing attachments in `attachments`; create and update accept attachments; the attachment endpoints upload, list and delete instead of refusing.

## Impact

- Backend: `internal/memos` (memo mapping, create, update, new attachment endpoints), `internal/httpapi/memos.go` (routes, larger body limit for uploads), `internal/notes` (helpers to split and join trailing attachment blocks), `internal/attachments` (listing the attachments of a set of notes).
- No database migration and no frontend change.
- The companion change `karakeep-assets` does the same for the Karakeep apps.
