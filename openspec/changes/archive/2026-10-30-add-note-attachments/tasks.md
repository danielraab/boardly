# Tasks

## 1. Shared file storage

- [x] 1.1 Extract `internal/blobs` (random IDs, atomic write under a directory, remove, sweep of files without a record, common response headers) from `internal/images`, and switch `images` to it; verify `go test ./internal/images/...` passes unchanged
- [x] 1.2 Add unit tests for `internal/blobs` (write/read/remove, sweep keeps files with a record); verify `go test ./internal/blobs/...` passes

## 2. Configuration and type table

- [x] 2.1 Add `internal/attachments/types.go` with the known-type table (content type, extensions, recognised by content or extension, inline or download, group) and `DetectType(name, data)`; verify with tests for "Recognised by content", "Recognised by extension", "Fake image", AVIF, ICO, WAV and Ogg normalisation
- [x] 2.2 Read `BOARDLY_ATTACHMENT_MAX_SIZE_MB` (5), `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` (30) and `BOARDLY_ATTACHMENT_TYPES` (default: images without SVG, plus PDF; `group/*` wildcards) in `internal/config`, failing on invalid values; verify with config tests for defaults, "Group" and "Unknown type"
- [x] 2.3 Add the three variables to `.env.example` with defaults and every known type listed, SVG and HTML marked as risky; verify by checking the file against the type table (a test that every table entry appears in `.env.example`)

## 3. Data model and store

- [x] 3.1 Add migration `0022_attachments.sql` with `attachments` and `note_attachments` as in design.md; verify the db migration test passes
- [x] 3.2 Implement `attachments.Store`: create (detect type, check allowed and size, write file and row), get, list for a user with `noteIds`, set/remove keep (removing also clears `deleted_at`), delete (409 when referenced); verify with store tests for each, including "Delete an attachment in use" and "Don't keep"
- [x] 3.3 Implement `RunCleanup` (mark after 24 h unused, delete after the configured days unless kept or referenced, sweep orphan files) and start it hourly in `main.go`; verify with tests for "Recent upload", "Unused upload", "Deleted after the period" and "Kept"

## 4. Note content and references

- [x] 4.1 Add the `attachment` block (`id` only) to `notes.Doc` and `ParseDoc` shape checks, and skip it in plain text and display title; verify with doc tests for a valid block, an unknown field and a missing ID
- [x] 4.2 In the notes store's create and update, check that every attachment ID belongs to the owner, replace `note_attachments` with positions, and clear `deleted_at` of referenced attachments, all in the content transaction; verify with tests for "Other user's attachment", "Unknown attachment", "Removed from the content", "Archived note" and "Referred to again"
- [x] 4.3 Return `attachments` (`id`, `name`, `contentType`, `size`, `url`, in content order, unique) on every note in the notes API; verify with the "Note with attachments" httptest case
- [x] 4.4 Add the attachment resolver to `ToMarkdown`, `FromMarkdown` and `ReplaceContentFromMarkdown` (image/file Markdown out; own attachment URLs in, splitting paragraphs for images; foreign URLs unchanged); verify with tests for "Own attachment", "File link alone in its paragraph", "Other user's attachment", "Attachments" round trip, and that existing tests pass with a `nil` resolver
- [x] 4.5 Pass a per-user resolver in the Karakeep API, the Memos API and the Karakeep importer; verify with a Memos test that updating a memo whose Markdown contains its attachment keeps the attachment block

## 5. Attachments HTTP API

- [x] 5.1 Add `POST /api/attachments?name=` (raw body, `MaxBytesReader`, `invalid_attachment_name`, `unsupported_attachment_type`, `attachment_too_large`); verify with httptest cases "Upload a PDF", "Too large", "Not allowed", "Path in the name" and 401 without a session
- [x] 5.2 Add public `GET /api/attachments/{id}` and `/{id}/{name}` with `ServeContent`, headers and inline/download disposition; verify with tests for "Visitor loads an image", "HTML is downloaded", a range request, "Marked attachment" and "Deleted attachment"
- [x] 5.3 Add `GET /api/attachments` (with `maxSize` and `allowedTypes`), `PATCH /api/attachments/{id}` and `DELETE /api/attachments/{id}`; verify with tests for "Keep", "Don't keep", "Delete an attachment in use", "Delete an unused attachment" and "Other user's attachment"
- [x] 5.4 Add attachment metadata to note widget rows for owners and visitors; verify the public dashboard test "Attachments for visitors" and that nothing else about other attachments is included

## 6. Frontend: document and rendering

- [x] 6.1 Add the attachment block and attachment types to `document/types.ts`, the API client (`attachments.ts`: list, upload, keep, delete) and the note types; verify with `api.test.ts`
- [x] 6.2 Add `AttachmentNode` (Lexical `DecoratorNode`) with the attachment map context, and map it in `document/lexical.ts` both ways; verify with `lexical.test.ts` round-trip cases
- [x] 6.3 Render attachment blocks (image, audio/video player, file card, "Attachment missing") in the editor and `NoteRenderer`; verify with tests for "Image in a note" and "PDF in a note"
- [x] 6.4 Add `ImageViewer` (Base UI Dialog, name, download, previous/next, arrow keys, focus return) and open it from images in the editors and widgets but not on cards; verify with tests for "Open an image" and "Next image"

## 7. Frontend: adding attachments

- [x] 7.1 Add the `insertAttachments` command with placeholders, remembered selection, end-of-note fallback and per-file error messages (too large with the maximum, type not allowed, failed); verify with tests for "Insert at the cursor", "Insert at the end" and "Upload too large"
- [x] 7.2 Add the paste/drop plugin to the content editor, including inline card editing; verify with "Paste a screenshot" and a drop test
- [x] 7.3 Add `AttachmentsButton` with count, list, viewer/open, "Remove from note", "Add attachment" and the empty state to the editor page and modal; verify with tests for "List", "Remove from note" and "Attachments button"

## 8. Frontend: attachments page and menu

- [x] 8.1 Add the "Attachments" link to the user menu; verify the app-shell menu tests "Menu with name", "Menu without name" and "Attachments link"
- [x] 8.2 Add the `/attachments` route with heading, totals, entries (preview, notes as links with display titles and archived mark, status texts), rows on desktop and stacked cards on phones, and the document title; verify with tests for "Open the page", "Signed out", "Note link", "Marked attachment" and "Kept attachment"
- [x] 8.3 Add the filters with counts, `?filter=` and empty states; verify with tests for "Unused" and "Unknown filter"
- [x] 8.4 Add Keep / Don't keep and Delete (disabled with an explanation when in use, confirmation, error rollback); verify with tests for "Keep a marked attachment", "Delete an unused attachment" and "Attachment in use"
- [x] 8.5 Add all new strings in English and German; verify the i18n test passes

## 9. Docs and final checks

- [x] 9.1 Update README: notes (attachments, viewer), the attachments page, configuration table (three variables), storage (`<data dir>/attachments`, public URLs, lifecycle), and the Memos section still listing attachments as not supported; verify by reading the sections
- [x] 9.2 Run `make check` and `make test`; verify both pass
