# Proposal

## Why

The favourites tree already allows the same URL more than once: the web app and the Linkwarden API save duplicates. The bookmark import is the exception that always drops entries whose URL is already saved, so a user can't import a file as it is, for example to keep a link that intentionally appears in two folders.

## What Changes

- `POST /api/favourites/import` takes an optional query parameter `skipDuplicates` (`true` or `false`). Missing means `true`, so the current behaviour stays the default. Any other value fails with `400` and code `invalid_request`.
- With `skipDuplicates=false`, the import compares no URLs at all: links already in the tree and links repeated within the file are all imported. The report keeps its shape; `skipped.duplicates` is `0`.
- The import form gets a checkbox "Skip links that are already saved" / "Bereits gespeicherte Links überspringen", checked by default, that sets the parameter.
- Unchanged: the Karakeep API keeps returning the existing bookmark (`alreadyExists`), because the browser extension relies on it; the web app and the Linkwarden API already allow duplicates.

Non-goals: a duplicate finder and merging duplicates by hand. They may follow in a later change.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `bookmark-import`: the import endpoint gets the `skipDuplicates` parameter, the duplicate rule applies only when it is on, and the import form gets the checkbox.

## Impact

- Backend: `backend/internal/httpapi/import.go` (parse the parameter), `backend/internal/favourites/import.go` (skip the duplicate check when off), their tests.
- Frontend: `frontend/src/features/settings/import/api.ts`, `ImportExportTab.tsx`, en/de translations, the import form test.
- API: additive, backwards compatible.
