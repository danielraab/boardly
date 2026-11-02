# Tasks

## 1. Configuration

- [x] 1.1 Add `Exports.PerHour` / `Exports.PerDay` to `config.Config`, read from `BOARDLY_EXPORTS_PER_HOUR` (default 3) and `BOARDLY_EXPORTS_PER_DAY` (default 10) as non-negative integers; invalid values return an error naming the variable. Verify with config tests for defaults, `0`, `-1` and `many`.

## 2. Rate limiter

- [x] 2.1 Implement the per-user sliding-window limiter (`Allow(userID, now)` → ok or wait duration; `0` disables a window; empty users are dropped). Verify with unit tests: hourly limit, daily limit, window passing (3 exports at 10:00–10:05, allowed at 11:01), per-user isolation, wait time equals time until the oldest blocking entry expires, disabled windows.

## 3. Export document

- [x] 3.1 Create `backend/internal/export` with the version 1 document structs (profile, favourites tree, tags, dashboards with widget tree, note categories, notes, media sources, images, attachments) and a `Service` holding the DB and both blob dirs. Verify `go build ./...`.
- [x] 3.2 Read all of a user's rows in one read-only transaction and build the document: tree and widget order by position, tags on links by name, note attachments in order, `notes.IconOf` for icons, raw JSON for widget config and note content, upload and preview images only, attachments without `deleted_at`, times in UTC RFC 3339. Verify with a test that seeds two users through the stores and checks Ann's document contains all her items with matching IDs and none of Bob's.
- [x] 3.3 Add the schema-coverage test: list every table with a foreign key to `users` in the migrated test database and fail unless it is exported or on the explicit "not exported" list (passkeys, app tokens, sessions, webauthn challenges, jobs, …). Verify the test passes and fails when a table is removed from both lists.

## 4. Archive

- [x] 4.1 Implement the ZIP writer: files first (`images/<id><ext>`, `attachments/<id>/<clean name>`), stored or deflated by type, skipped with `file: null` when unreadable, `boardly.json` last with `withFiles: true`. Verify with tests: expected entries and paths, unchanged file bytes, `../../etc/passwd` stays inside `attachments/<id>/`, a missing blob is left out and marked `null`.

## 5. Endpoint

- [x] 5.1 Add `GET /api/me/export` behind `requireUser`: validate `files` (`400 invalid_request`, not counted), check the limiter (`429 rate_limited` with `Retry-After`), set `Content-Type`, `Content-Disposition` (`boardly-export-<slug|user>-<UTC date>.<json|zip>`) and `Cache-Control: no-store`; encode JSON fully before writing, stream ZIP and abort the connection on a mid-stream error. Wire the service, limiter and config in `main.go`/`router.go`. Verify with httptest: 200 JSON and ZIP with file names, 400 for `files=yes`, 401 signed out, 401 with only an app token, 200 while deletion is scheduled, 429 after the configured number of exports with a correct `Retry-After`, another user unaffected.

## 6. Frontend

- [x] 6.1 Add the API helper that fetches the export, returns the blob and file name from `Content-Disposition`, and maps `429` to an error carrying the retry time from `Retry-After`. Verify with a Vitest unit test.
- [x] 6.2 Add the "Export" section to `ImportExportTab.tsx` after the import section: description, "Include files (images and attachments)" checkbox (off) with ZIP hint, "Export everything" button with in-progress state, saving via object URL, limit-reached message with local time, general error message. Add en/de strings. Verify in `ImportExport.test.tsx`: default sends `files=false`, checked sends `files=true`, button disabled while running, 429 with `Retry-After: 1200` shows the time 20 minutes later.

## 7. Documentation and checks

- [x] 7.1 Update README: export feature in the features list, the two environment variables in the configuration table, and a short description of the export format (JSON vs. ZIP, contents, what is left out, version). Verify by reading the rendered sections.
- [x] 7.2 Run `make check` and `make test` and confirm both pass; run `openspec validate export-all-data --strict`.
