# Design

## Context

- The user's data is spread over SQLite tables that all hang off `users` (`favourite_nodes`, `tags`, `favourite_node_tags`, `dashboards`, `dashboard_widgets`, `note_categories`, `notes`, `note_attachments`, `attachments`, `media_sources`, `images`) plus a few columns on `users` (`slug`, `share_folder_id`, `note_category_id`). The `accounts` package already knows this full list for account deletion.
- Image and attachment files live in `blobs.Dir` directories (`ImageFiles`, `AttachmentFiles`), named by ID. Records live in the database.
- The database runs in WAL mode, so a read transaction sees a consistent snapshot while writes continue.
- `/api/me*` routes go through `requireUser`, which only accepts a browser session. App tokens can't reach them.
- `ratelimit.Limiter` is a token bucket keyed by string. It can't express "at most N in the last 60 minutes" exactly.
- Config is read in `config.Load` from `BOARDLY_*` variables. Invalid values stop the server with an error naming the variable.

## Goals / Non-Goals

**Goals:**
- One read of the user's data that is consistent across all tables.
- A ZIP that streams to the client without temp files or holding all file contents in memory.
- A document whose shape is stable and versioned, so a later import change can read version 1.

**Non-Goals:**
- Import of the Boardly format (later change).
- Background or emailed exports, export history, progress reporting from the server.
- Persisting rate-limit counts across restarts.
- Exporting other formats (Netscape HTML, Markdown).

## Decisions

### A dedicated `export` package that reads with SQL
New package `backend/internal/export` with a `Service` that takes the `*sql.DB` and both `blobs.Dir`s. It opens one read-only transaction (`BeginTx` with `ReadOnly: true`), reads every table for the user, and builds the document structs from those rows.

*Why not reuse the store packages' list functions?* They each open their own queries and some shape results for the UI (e.g. note list paging, tree with favicons). Reading them one after another isn't a snapshot, and adding a `Tx` parameter to each would touch many packages. The export reads raw rows and maps them itself. The cost is a second place that knows the schema. A test that compares the export with the table list in `accounts` reduces that risk (see Risks).

Icons and note content are rendered in the same shape as the API by reusing `notes.IconOf`, not by copying the logic.

### Document types are separate, versioned structs
The JSON shape is defined by structs in `export` (`DocumentV1`, `FolderV1`, …), not by the API response types. API types can change without silently changing export version 1. Field names are camelCase like the API.

Widgets: `config` is embedded as raw JSON (`json.RawMessage`), as stored. Note content: the stored document, also raw JSON.

### Writing order: files first, `boardly.json` last
With `files=true` the handler sets the headers and writes a `zip.Writer` directly to the `ResponseWriter`. It copies each image and attachment file into the archive (`zip.Store` for already-compressed types like images and PDFs, `Deflate` otherwise) and records the path of each file it could open. A file that can't be opened is skipped and its `file` stays `null`. Then it writes `boardly.json` with the final `file` values.

*Alternative:* JSON first. Then missing files couldn't be marked in the document without checking every file twice.

Since the `200` is already sent when streaming starts, a failure halfway can't become an error response. The handler logs it and aborts the connection (`panic(http.ErrAbortHandler)`), so the client sees a failed download instead of a ZIP that looks complete but is cut off.

With `files=false`, the document is encoded in full before writing, so errors still give a normal JSON error response.

### File names inside the archive
`images/<id><ext>`: the extension comes from a small fixed map of the content types images can have (`image/png`, `image/jpeg`, `image/gif`, `image/webp`, plus whatever link previews store), with `.bin` as fallback.
`attachments/<id>/<name>`: the name is cleaned by taking `path.Base`, replacing `/`, `\`, control characters and `:*?"<>|` with `_`, trimming dots and spaces at the ends, cutting to 200 bytes on a UTF-8 boundary, and falling back to `file` when nothing is left. The ID directory keeps names unique.

### Rate limit: in-memory sliding window per user
New type `export.Limiter` (or `ratelimit.Window`, if it fits better there) holding, per user ID, the timestamps of the exports in the last 24 hours, at most `max(perDay, perHour)` entries. `Allow(userID, now)` drops entries older than 24 hours, counts those in the last hour, and either records `now` and returns ok, or returns the wait time: the time until the oldest entry that blocks drops out of its window. Users with no entries left are removed from the map. A mutex guards it. The limiter is checked after the request is validated and before anything is read.

*Why not two token buckets?* A bucket of 3 per hour lets up to 6 exports pass within 60 minutes (burst plus refill), which isn't what the setting says.

### Config
`config.Exports.PerHour` and `PerDay`, read from `BOARDLY_EXPORTS_PER_HOUR` (default 3) and `BOARDLY_EXPORTS_PER_DAY` (default 10) with the same non-negative integer parsing as `BOARDLY_ACCOUNT_DELETION_DAYS`. `0` turns that window off.

### Route
`GET /api/me/export` in `router.go`, wrapped in `requireUser`. It doesn't check the deletion state, so it works while a deletion is scheduled. No CSRF concern: it's a GET with no side effects other than the rate-limit count, and other sites can't read the response.

### Frontend: fetch to a blob, then save
The button uses `fetch` (with the existing API helper's error handling), reads the response as a `Blob`, takes the file name from `Content-Disposition`, and saves it with a temporary object URL and an `<a download>` click. This lets the section show progress and the `429` message with the time from `Retry-After`.

*Alternative:* a plain `<a href download>` link. It streams to disk without holding the file in the page, but a `429` or other error would show up as a downloaded error file or nothing at all. Browsers keep large blobs on disk, so memory is acceptable for the expected sizes.

## Risks / Trade-offs

- [A new table is added later and forgotten in the export] → A test lists the tables that reference `users` in the migrated schema and fails when one isn't either exported or on an explicit "not exported" list (passkeys, app tokens, sessions, …).
- [The export knows the schema, separate from the stores] → Kept to plain reads. Tests seed data through the stores' public functions and check the export, so a schema change that breaks the export fails in tests.
- [Large ZIP holds a read transaction open while streaming files] → The transaction is closed once all rows are read, before any file is copied. In WAL mode a long reader would only delay checkpoints anyway.
- [An attachment or image is deleted between reading the rows and copying the file] → Counted as a file that can't be read: skipped, `file` stays `null`.
- [Blob in the browser for very large exports] → Accepted for now. If it becomes a problem, switch the "with files" variant to a direct link after a preflight check of the limit.
- [Rate limit resets on restart] → Accepted. It protects against accidental or casual repeated use, not a determined user on a self-hosted server.

## Migration Plan

No database migration. The new variables have defaults, so existing deployments need no change. Rollback is removing the route and section.
