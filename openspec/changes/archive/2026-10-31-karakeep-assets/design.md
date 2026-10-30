# Design

## Context

Boardly answers Karakeep's tRPC procedures under `/api/karakeep/api/trpc/` (`internal/trpc`, `internal/karakeep`). Text bookmarks are notes (`karakeep/notes.go`). Today `createBookmark` with `type` `asset` and any `assetContent` fail with `BAD_REQUEST`, and `Bookmark.Assets` is always `[]`.

What the Karakeep mobile app does (`apps/mobile/lib/upload.ts`, `sharing.tsx`, `lib/hooks.ts` and `packages/shared/types` in karakeep-app/karakeep, October 2026):

- **Share a file:** the share sheet takes the first shared file, whatever its MIME type, and sends `POST <address>/api/assets` as `multipart/form-data` with a part named `file`, the file name, the MIME type and the `Authorization: Bearer <key>` header.
- **Upload response:** any 2xx with `{assetId, contentType, size: number, fileName}`. For errors it shows `body.error` when the body is JSON `{error}`, else "Upload failed (HTTP n)".
- **Create:** then `bookmarks.createBookmark({type: "asset", assetId, assetType: contentType === "application/pdf" ? "pdf" : "image", source: "mobile"})`, with no title and no file name.
- **View:** loads `<address>/api/assets/<assetId>` with the same headers, in an image viewer (`expo-image`) or a PDF viewer.

Boardly's `<address>` is `<BOARDLY_BASE_URL>/api/karakeep`, so the two REST paths are `/api/karakeep/api/assets` and `/api/karakeep/api/assets/{id}`, next to the tRPC route on the same mux, behind the same app-token middleware.

## Goals / Non-Goals

**Goals:**
- Sharing a photo or PDF to Karakeep creates a note with that file, and the app shows it as an image or PDF bookmark.

**Non-Goals:**
- `bookmarks.attachAsset`, `replaceAsset` and `detachAsset` (adding files to existing bookmarks, banner images). These stay unimplemented and return `NOT_FOUND`.
- Asset bookmarks in the Karakeep JSON import.
- Assets on link bookmarks (screenshots, archives).
- Choosing the target category per share. Notes go to the token's root category, like text bookmarks.

## Decisions

### A shared file becomes a note with one attachment block

Asset bookmarks map to notes, as text bookmarks already do. The content is `{v:1, blocks:[{type:"attachment", id}]}`, and the title is the file name, so the note isn't "Untitled" on `/notes`.

- *Alternative: a link node pointing to the attachment URL.* This would mix files into the favourites tree, and links don't hold attachments, so the cleanup would delete the file.
- *Alternative: a dedicated "Files" category.* It isn't needed. The token's root category already says where the app's notes go.

### Asset or text is derived from the content

A note is an asset bookmark exactly when its content is one attachment block of `image/*` or `application/pdf`, the two kinds the app can show. Nothing is stored about how the note was created. So a photo note made in the web app also shows as a photo in Karakeep. A shared photo that gets text added in Boardly turns into a text bookmark, which the app can still show as Markdown with the image link. The check needs the attachment's content type, which `Note.Attachments` already has.

### Upload handler

The handler uses `r.MultipartReader()` and streams the `file` part through `attachments.Store.ReadLimited`, so the size limit applies without buffering the whole form. `http.MaxBytesReader` at `MaxBytes + 64 KB` caps the whole body. Then it calls `Store.Create(ctx, owner, part.FileName(), data)`. Errors are written as `{"error": msg}` by a small `karakeepREST` wrapper, separate from the tRPC error format and from the session API's error format, so the app shows the message. Status codes: 400 for an invalid request, a name error or an unsupported type, 413 for `ErrTooLarge`, 404 for not found.

The token's note scope is checked first (`none` → 400). Asset bookmarks are notes, so a token that can't create notes would only leave orphaned files.

### Serving

`GET …/assets/{id}` checks that the attachment belongs to the token's owner and is in scope (referenced by an in-scope note, or by no note at all), using `attachments.Store.Get` and its `NoteIDs`. Then it uses the same response code as `GET /api/attachments/{id}/{name}` (headers, range requests through `http.ServeContent`). Extract that code into a shared helper in `httpapi/attachments.go` rather than duplicating it. Allowing unreferenced files lets the app preview a file right after uploading it. That only exposes the owner's own unused uploads to the owner's token.

The public `/api/attachments/...` URL could have served the app instead. But the app builds the asset URL itself from `<address>/api/assets/<id>`, and we can't give it a different one.

### Create

`createBookmark` gets an `asset` branch next to `text`. It resolves `assetId` against the owner's attachments (`attachments.Store.Get`, or the existing `UserMarkdownAttachments` map), builds the one-block document and creates the note through `notes.Store.CreateNote`. `CreateNote` validates the block and writes the reference in the same transaction. The `"file bookmarks"` entry stays in the unsupported-fields check for `assetContent` on updates, but its message no longer claims that Boardly doesn't store files.

## Risks / Trade-offs

- **[A share whose MIME type isn't allowed, such as HEIC from some phones]** → The upload returns 400 with "This type of file can't be attached", and the app shows it. Allowing more types is a `note-attachments` concern.
- **[The app labels every non-PDF file `image`]** → `assetType` from the client is ignored. A shared `.docx` (when allowed) becomes a note with a file card and is read back as a text bookmark.
- **[Upload without a following create]** (app killed) → The unused file is marked after a day and deleted by the cleanup.
- **[Title from the file name, like `IMG_20261030_0815.jpg`]** → Accepted. The user can rename the note in either app.

## Migration Plan

No data migration. Rollback is a redeploy. Notes created from shared files stay normal notes with an attachment.
