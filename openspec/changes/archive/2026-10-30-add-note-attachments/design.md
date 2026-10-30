# Design

## Context

- Files today live only in `images` (`internal/images`): favicons (shared, no owner, replaced in
  place), link previews (server-made, GC after 1 h unused) and uploads used as icons (GC after 24 h
  unused). All are image-only, at most 1 MB, served publicly at `/api/images/{id}`.
- A note's content is a JSON document validated by `notes.ParseDoc`, which is pure and knows nothing
  about the database. Content is saved through the notes store from four places: the notes API, the
  Karakeep API (`karakeep/notes.go`), the Memos API (`memos/service.go`) and the Karakeep importer.
  The Markdown converters `FromMarkdown`, `ReplaceContentFromMarkdown` and `ToMarkdown` are pure too.
- The editor is Lexical (`frontend/src/features/notes/editor`), with conversion between the stored
  document and Lexical state in `document/lexical.ts` and read-only rendering in `NoteRenderer.tsx`.

## Goals / Non-Goals

**Goals:**
- Ownership of attachment IDs in content is enforced on the server for every save path, in the
  same transaction that writes the references.
- Markdown round trips through connected apps keep attachments.
- Files of any allowed type can be served publicly without becoming an XSS vector on Boardly's
  origin.

**Non-Goals:**
- Moving images into the attachments table (see Decision 1).
- Thumbnails or image conversion for attachments; images are shown as uploaded.
- Memos API attachments (next change), renaming, deduplication, per-user quotas.

## Decisions

### 1. Separate `attachments` table; share only the file storage code

`images` holds three sources with three lifecycles, two of them without a user choice. Merging
would add a `source` column that every attachment query and the management page must filter on,
and would put icon uploads in front of users as "files" with a 30-day deletion mark they never asked
for. Instead:

```
images (table) ─┐                       ┌─ attachments (table)
                └─► internal/blobs ◄────┘
                    random IDs, atomic write (temp + rename) under a directory,
                    remove file, sweep files without a record,
                    common response headers (nosniff, CSP sandbox, cache)
```

`internal/images` is refactored to use `internal/blobs` without behaviour changes.
Alternative considered: a single `files` table with a `kind` column. Rejected for the reasons
above. The shared helper gives the code reuse without coupling the lifecycles.

### 2. Data model (migration `0022_attachments.sql`)

```sql
CREATE TABLE attachments (
  id           TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size         INTEGER NOT NULL CHECK (size > 0),
  created_at   TEXT NOT NULL,
  deleted_at   TEXT,            -- deletion mark
  keep         INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX attachments_user ON attachments (user_id, created_at);

CREATE TABLE note_attachments (
  note_id       TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
  attachment_id TEXT NOT NULL REFERENCES attachments (id),
  position      INTEGER NOT NULL,   -- first appearance in the content
  PRIMARY KEY (note_id, attachment_id)
);
CREATE INDEX note_attachments_attachment ON note_attachments (attachment_id);
```

`note_attachments.attachment_id` has no `ON DELETE`: the foreign key itself prevents deleting an
attachment in use, as a safety net behind the 409 check. `position` lets the notes API return
`attachments` in content order without parsing the content. User deletion cascades rows. Files
without a row are removed by the hourly sweep.

### 3. References and ownership checked in the notes store, not in `ParseDoc`

`ParseDoc` stays pure and only checks the shape of `{"type":"attachment","id":…}`. The notes
store's create and update functions, which all four save paths already use, collect the attachment
IDs from the document in order and, inside their transaction:

1. `SELECT id FROM attachments WHERE user_id = ? AND id IN (…)` and fail with the existing
   invalid-document error when any is missing;
2. replace the note's `note_attachments` rows;
3. `UPDATE attachments SET deleted_at = NULL WHERE id IN (…) AND deleted_at IS NOT NULL`.

Alternative: pass a lookup into `ParseDoc`. Rejected because it would make every caller set up the
lookup, and the check would run outside the transaction that writes the references.

The `notes` package queries the `attachments` table directly with SQL and only imports
`internal/attachments` for its pure helpers (`URL`, the type table). The `attachments` package reads
`note_attachments` and never imports `notes`, so there is no import cycle. A note's attachments are
read with a JSON subquery that is part of the shared note columns, so every query that returns notes
(API, widgets, Memos, Karakeep) has them without extra round trips.

### 4. Markdown conversion takes the known attachments

`FromMarkdown`, `ReplaceContentFromMarkdown` and `ToMarkdown` get an extra
`*notes.MarkdownAttachments` parameter: Boardly's origin and the attachments that may appear, by ID.
For writing, `NoteMarkdownAttachments(origin, note)` takes the note's own attachments; for reading,
`Store.UserMarkdownAttachments` loads all of the user's attachments, because the app may paste one
from another note. The Karakeep and Memos services get the origin like the Karakeep service already
does for favicons. `nil` means "no attachments", which keeps the pure tests, `FromText` and the
importer unchanged. Attachment URLs are recognised as `(<origin>)?/api/attachments/<id>(/<anything>)?`;
unknown or foreign IDs fall through to the existing image and link rules. An image of an attachment
splits its top-level paragraph; images inside lists or quotes stay alt text.

### 5. Type detection and serving

The type table is a Go slice in `internal/attachments/types.go`, the single source for detection,
config validation, `.env.example` and the API's `allowedTypes`. Content sniffing uses
`http.DetectContentType` plus small magic checks it lacks (AVIF `ftypavif`, ICO, BMP is covered,
WAV/`audio/wave` normalised to `audio/wav`, `application/ogg` to `audio/ogg`).

Serving uses `http.ServeContent`, which gives range requests (needed for video seeking) and
`If-Modified-Since`. Headers: stored `Content-Type`, `nosniff`, `Content-Security-Policy: sandbox`,
`Cache-Control: public, max-age=31536000, immutable`, and `Content-Disposition` built with
`mime.FormatMediaType("inline"|"attachment", {"filename": name})`, which encodes non-ASCII names
as RFC 2231/5987. SVG and HTML are always downloads when opened directly. The web app still shows
SVG inside notes via `<img>`, where scripts don't run.

Upload reads the body through `http.MaxBytesReader(max+1)` into memory and then writes it out.
At the 5 MB default this is cheap. The name comes from the `name` query parameter rather than
multipart, matching the raw-body style of `POST /api/images`.

### 6. Lifecycle job

`attachments.Store.RunCleanup(ctx, time.Hour)` is started from `main.go` next to the image cleanup.
Each run executes in one transaction:

```sql
UPDATE attachments SET deleted_at = :now
 WHERE deleted_at IS NULL AND keep = 0 AND created_at <= :now - 24h
   AND NOT EXISTS (SELECT 1 FROM note_attachments WHERE attachment_id = attachments.id);

DELETE FROM attachments
 WHERE keep = 0 AND deleted_at <= :now - :days
   AND NOT EXISTS (SELECT 1 FROM note_attachments WHERE attachment_id = attachments.id)
RETURNING id;
```

The files of the returned IDs are removed after the commit, followed by a sweep for files that
have no row. `deleteAt` in the API is computed as `deleted_at + days` and is `null` when `keep`
is set.

### 7. Frontend

- **Lexical node:** `AttachmentNode` is a `DecoratorNode` holding only `id`. It renders from an
  attachment map (`id → {name, contentType, size, url}`) that is provided through context. The map
  is filled from the note's `attachments` and updated with each upload response.
  `document/lexical.ts` maps the block both ways.
- **Upload flow:** an `insertAttachments(files, position)` command inserts placeholder nodes first
  (at the drop point, the current or last remembered selection, or the root end), uploads the files
  one after another, then replaces each placeholder with an `AttachmentNode` or removes it on
  error. The last selection is remembered in a ref on blur, so a click on the dropdown doesn't
  lose the cursor. Autosave only serialises real attachment nodes, so placeholders are never saved.
- **Paste and drop:** handled by a plugin in the editor (`PASTE_COMMAND` / `DROP_COMMAND` with
  files), which is active wherever the content editor is used, including inline card editing.
- **Rendering:** `NoteRenderer` renders attachment blocks the same way for cards and widgets.
- **Shared components:** `AttachmentsButton` (Base UI Menu/Popover) on the editor page and in the
  modal, and `ImageViewer` (Base UI Dialog), which is used by the editor, widgets and
  `/attachments`.
- **Limits:** `allowedTypes` and `maxSize` come from `GET /api/attachments`, cached in the query
  client. They're used for the file picker's `accept` and for the error messages.
- **`/attachments` route:** loads `GET /api/attachments` and `GET /api/notes` to compute display
  titles with the existing helper. The filter is a search param.

## Risks / Trade-offs

- [Public URLs for private files] → IDs are random and unguessable like image IDs. The user chose
  public visibility, and the README documents it.
- [Stored XSS through HTML/SVG on Boardly's origin] → download disposition, `nosniff` and CSP
  `sandbox` on every attachment response. Both types are excluded by default and marked as risky
  in `.env.example`.
- [A client's stale content refers to an attachment that was deleted] → deletion is only possible
  with no references, so this needs an old undo stack or an offline app. The save then gets
  `invalid_document` and the existing error message, which is acceptable for such a rare case.
- [Immutable cache after deletion] → browsers may keep showing a deleted file from cache. This
  matches images and is acceptable because deletion is deliberate.
- [Upload held in memory] → bounded by the configured maximum. If larger limits are needed, the
  upload can be streamed to a temporary file later without an API change.
- [Changing the Markdown converters' signatures touches many tests] → passing `nil` keeps the
  existing tests' behaviour, so only call sites change.

## Migration Plan

The migration only adds tables, and existing notes have no attachment blocks, so no data changes.
Rolling back to an older binary leaves the extra tables unused. Notes that contain attachment
blocks would then fail validation in the old version on save. That's acceptable for a self-hosted
rollback, and the release notes mention it.
