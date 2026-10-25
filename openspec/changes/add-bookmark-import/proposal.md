## Why

People moving to Boardly from Karakeep or Linkwarden have to re-create their bookmarks by hand, because the compatible APIs only cover saving new links from the apps. A one-time import from their export files makes switching practical and keeps folders, tags and creation dates.

## What Changes

- New one-time bookmark import from an uploaded file (no syncing, no connection to a running Karakeep or Linkwarden instance). Supported formats:
  - Karakeep JSON export (bookmarks and nested lists),
  - Linkwarden JSON export (collections, links and tags),
  - Netscape bookmark HTML (exported by Karakeep, Linkwarden and browsers).
- The user must pick an import folder in their favourites tree, and the tree root is allowed. The source's folder structure is recreated below that folder. A folder that already exists there with the same name, compared case-insensitively, is reused; missing folders are created.
- Links keep their original creation date and the order of the file. Their tags are attached as Boardly tags.
- A URL that is already anywhere in the user's tree, or appears earlier in the same file, is skipped.
- A Karakeep bookmark in several lists goes into the first one.
- Karakeep notes and summaries are appended to the link description, which is cut to 2048 characters.
- Not imported: Karakeep text and asset bookmarks, smart lists, and the archived, favourited and pinned flags. These are counted in the import report. Text and asset bookmarks can be revisited once Boardly has notes, and the flags once Boardly has such flags.
- The import runs in one transaction (all or nothing) and returns a report of what was imported and skipped.
- New "Import / Export" settings tab (`?tab=data`) with the import form. The export section is reserved for a later change.

## Capabilities

### New Capabilities
- `bookmark-import`: importing Karakeep JSON, Linkwarden JSON and Netscape HTML files into a chosen folder of the favourites tree, including the mapping rules, duplicate handling, the import API, the report and the import form.

### Modified Capabilities
- `user-profile`: the settings page gets a fourth tab, "Import / Export" (`?tab=data`).

## Impact

- Backend: new `internal/importer` package (parsers and a common import model), a bulk import operation in `internal/favourites` that sets `created_at` from the source and queues the usual favicon and preview fetches, and a new endpoint `POST /api/favourites/import` (raw file body as for image uploads, size-limited).
- Frontend: new settings tab with the import form (format choice, file, import folder via the existing `FavouritePicker`, result report), plus en and de translations.
- Background jobs: a large import queues one preview fetch per link and one favicon fetch per new origin.
- No database migration and no new dependencies. HTML is parsed with `golang.org/x/net/html`, which is already in `go.mod`.
- README: document the import.
