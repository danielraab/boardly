# Proposal

## Why

Notes can only hold formatted text, so photos, PDFs and other files that belong to a note have to
live somewhere else ([issue #72](https://github.com/danielraab/boardly/issues/72)). Users want to
attach files to a note, see them inside the note, and keep an overview of all their files, without
losing a file just because it was removed from a note by mistake.

## What Changes

- New **attachments**: files a user uploads, stored under `<data dir>/attachments/` in their own
  table, separate from `images`. An attachment has a name, a type, a size and an owner, and is
  served publicly at an unguessable URL `/api/attachments/{id}/{name}`.
- New **attachment block** in the note document: `{"type": "attachment", "id": "<id>"}`. Files
  always appear in a note's content: images, audio and video are shown inline, other files as a
  file card. The document version stays `1`. In Markdown, an attachment is
  `![name](<url>)` (images) or `[name](<url>)` (other files), and such a URL of one of the user's
  own attachments is read back as an attachment block.
- A note's **references** to attachments are taken from its content on every save. Archived notes
  still refer to their attachments.
- An **attachments button** (paperclip) in the note editor opens a dropdown with the note's
  attachments, an "Add attachment" action and "Remove from note". A new attachment is inserted at
  the cursor, or at the end of the note. Files can also be pasted or dropped into the content.
- Clicking an image opens an **image viewer**.
- **Lifecycle**: a cleanup job marks an attachment for deletion (sets `deleted_at`) when it is at
  least 24 hours old and no note refers to it. A marked attachment is deleted after
  `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` (default 30) days, unless a note refers to it again
  (which unmarks it) or the user sets **Keep**. Marked attachments are still served.
- New **attachment management page** `/attachments`, linked from the user menu: all attachments
  with the notes that use them (as links), filters (in use, unused, marked for deletion, kept),
  Keep / Don't keep, and Delete, which is only possible when no note refers to the attachment.
- New configuration: `BOARDLY_ATTACHMENT_MAX_SIZE_MB` (default 5),
  `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` (default 30) and `BOARDLY_ATTACHMENT_TYPES` (allowed
  types out of a fixed table of known types; default: images except SVG, and PDF). All are listed
  in `.env.example`, with every possible type.
- Not in this change: attachments in the Memos API (MoeMemos uploads), which come in the next
  change; renaming attachments.

## Capabilities

### New Capabilities
- `note-attachments`: attachment storage, known types and their configuration, upload, public
  serving, references from note content, the mark-and-delete lifecycle, the attachments API, and
  how attachments are added and shown in the editor, on cards and in widgets, including the image
  viewer.
- `attachment-management`: the `/attachments` page listing all attachments with their notes,
  filters, Keep and Delete.

### Modified Capabilities
- `note-document`: the document format gains the attachment block; validation only accepts the
  user's own attachments; Markdown in and out handles attachments.
- `notes`: notes are returned with their attachments; the note editor page gets the attachments
  button.
- `note-widget`: visitors of public dashboards also get the attachments the note shows.
- `app-shell`: the signed-in user menu gets an "Attachments" link.

## Impact

- Backend: new migration (`attachments`, `note_attachments`), new `internal/attachments` package
  (store, type table, upload/serve/list/patch/delete handlers, cleanup job), file storage code
  shared with `internal/images`, document validation and Markdown conversion in `internal/notes`,
  reference updates in every path that saves note content (notes API, Karakeep, Memos, import),
  widget rows, config, `main.go` wiring.
- Frontend: Lexical attachment node, attachments dropdown, upload with paste and drop, image viewer,
  rendering in `NoteRenderer`, new `/attachments` route, user menu, en + de strings.
- Configuration and docs: `.env.example`, README (notes, configuration, storage).
- New public endpoint `GET /api/attachments/{id}/{name}`; files of arbitrary allowed types are
  served with `nosniff`, `Content-Security-Policy: sandbox` and a download disposition for types
  that aren't safe to show inline.
