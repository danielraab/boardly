# Proposal

## Why

Users can bring bookmarks and notes into Boardly, but there is no way to get their data back out. A self-hosted app should let its users keep a backup of their own data and take it with them, and the "Delete account" section already points users to the "Import / Export" tab to export their data first, where no export exists yet.

## What Changes

- New endpoint `GET /api/me/export` that downloads all data of the signed-in user in one versioned Boardly format (`"format": "boardly"`, `"version": 1`) with the IDs the API uses, so references between items stay intact:
  - profile (email, display name, slug, default folder, default note category),
  - the favourites tree with tags,
  - dashboards with their widgets,
  - notes, note categories and media sources,
  - the records of uploaded images, link preview images and attachments.
- Passkeys, app tokens, favicons, sessions, sign-in links and background jobs are not part of the export.
- `files=false` (the default) downloads a plain `boardly-export-….json`. With `files=true`, the download is a `.zip` that holds the same JSON as `boardly.json` plus the image and attachment files.
- The export is allowed while an account deletion is scheduled.
- Exports are rate-limited per user with two limits: `BOARDLY_EXPORTS_PER_HOUR` (default `3`) and `BOARDLY_EXPORTS_PER_DAY` (default `10`), counted over sliding windows. An export over the limit gets `429 rate_limited` with `Retry-After`.
- The "Import / Export" settings tab gets an "Export" section: a checkbox "Include files (images and attachments)", off by default, and an "Export everything" button. The section shows progress, errors, and when the next export is possible after a rate-limit error.
- Importing this format is not part of this change.

## Capabilities

### New Capabilities
- `data-export`: The export of all data of a user: the endpoint, the contents and format of the JSON and the ZIP, what is left out, the rate limit and its configuration, and the "Export" section in the settings.

### Modified Capabilities
- `bookmark-import`: The "Import form" requirement drops the sentence that excludes an "Export" section from the "Import / Export" tab. The section is specified in `data-export`.

## Impact

- Backend: new export package that reads a consistent snapshot of the user's data and writes the JSON or streams the ZIP; new route in `httpapi`; per-user sliding-window limiter; two new settings in `config`.
- Frontend: `ImportExportTab.tsx` gets the export section; new API helper; en/de translations.
- README: feature description, the two environment variables, and a description of the export format.
- No database migration. No new Go dependencies (`archive/zip` and `encoding/json` from the standard library).
- The existing hint in the "Delete account" section already links to the tab and needs no change.
