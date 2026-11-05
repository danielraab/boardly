# data-export Specification

## Purpose
Lets a signed-in user download all of their own data from Boardly in one versioned file, as a backup and to take it elsewhere, with or without their uploaded files, under a configurable per-user rate limit.

## Requirements

### Requirement: Export endpoint
`GET /api/me/export` SHALL download all data of the signed-in user. The query parameter `files` is `true` or `false`, and a missing `files` means `false`. Any other value gets `400` with code `invalid_request`, and the export isn't counted towards the rate limit.

- With `files=false`, the response SHALL be the export document (see "Export document") with `Content-Type: application/json` and a `Content-Disposition` of `attachment` with the file name `boardly-export-<name>-<date>.json`.
- With `files=true`, the response SHALL be a ZIP archive with `Content-Type: application/zip` and a `Content-Disposition` of `attachment` with the file name `boardly-export-<name>-<date>.zip` (see "Export archive").

`<name>` is the user's slug, or `user` when the user has no slug. `<date>` is the date of the export in UTC as `YYYY-MM-DD`. The response SHALL carry `Cache-Control: no-store`. The endpoint requires a signed-in user, and it SHALL work while a deletion of the account is scheduled. App tokens can't use it.

#### Scenario: Export without files
- **WHEN** Ann, with slug `ann`, requests `GET /api/me/export` on 2026-11-02
- **THEN** the response is `200` with a JSON document and the file name `boardly-export-ann-2026-11-02.json`

#### Scenario: Export with files
- **WHEN** Ann requests `GET /api/me/export?files=true`
- **THEN** the response is `200` with a ZIP archive and the file name `boardly-export-ann-2026-11-02.zip`

#### Scenario: Invalid files parameter
- **WHEN** Ann requests `GET /api/me/export?files=yes`
- **THEN** the response is `400` with code `invalid_request`, and the request doesn't count towards the rate limit

#### Scenario: Deletion scheduled
- **WHEN** Ann has scheduled the deletion of her account and requests `GET /api/me/export`
- **THEN** the response is `200` with her data

#### Scenario: Signed out
- **WHEN** a visitor without a session requests `GET /api/me/export`
- **THEN** the response is `401`

#### Scenario: App token
- **WHEN** a request to `GET /api/me/export` carries only an app token
- **THEN** the response is `401`, and no data is exported

### Requirement: Export document
The export document SHALL be a JSON object with:

- `format`: `"boardly"`,
- `version`: `1`,
- `exportedAt`: the time of the export,
- `withFiles`: whether the export contains the files,
- `profile`: the email address, display name, slug, creation time, the ID of the default folder for new links and the ID of the default category for new notes,
- `favourites`: the user's favourites tree, starting at the root folder, with every folder's children in their order. A folder has its ID, title, creation and change time and children. A link has its ID, URL, title, description, tag names, preview image ID, creation and change time, and `offlineCopy`. `offlineCopy` is `null` when the link has no offline copy with an article or a screenshot. Otherwise it is an object with `capturedAt`, `title`, `article` and `screenshot`. `article` and `screenshot` are the paths of these files inside the export archive, or `null` when the copy doesn't have that part, the export has no files, or the file couldn't be read,
- `tags`: all of the user's tags with ID, name and creation time, including tags no link uses,
- `dashboards`: every dashboard with its ID, slug, title, whether it is the default and whether it is public, its layout settings, creation and change time, and its widgets as a tree in their order, each with ID, kind, width and configuration as stored,
- `noteCategories`: every note category with ID, parent ID, title, icon, colour, and creation and change time,
- `notes`: every note, archived ones included, with ID, category ID, title, icon, colour, content document as stored, pinned and archived flags, the IDs of its attachments in their order, and creation and change time,
- `mediaSources`: every media source with ID, kind, title, URL, icon and position,
- `images`: every image the user uploaded or that was fetched as a link preview for them, with ID, source (`upload` or `preview`), content type, size, creation time, and `file`,
- `attachments`: every attachment of the user that isn't deleted, with ID, file name, content type, size, creation time, and `file`.

IDs SHALL be the IDs the Boardly API uses for the same items, so that references inside the document (default folder and category, tag names, icon and preview image IDs, widget configurations, note content and attachments) point to items in the same document. Icons have the same shape as in the API. Times are RFC 3339 timestamps in UTC. `file` is the path of the file inside the export archive, or `null` when the export has no files or the file couldn't be read. The records of images and attachments SHALL be included also when the export has no files. The state of offline copies (pending or failed) and their pending captures SHALL NOT be exported.

#### Scenario: Version and format
- **WHEN** Ann exports her data
- **THEN** the document has `format` `"boardly"`, `version` `1` and `exportedAt` set to the time of the export

#### Scenario: Favourites tree with tags
- **WHEN** Ann's tree has a folder "Dev" with the link `https://go.dev` tagged "go"
- **THEN** the document's `favourites` contains "Dev" with that link, its tag "go", and "go" is listed in `tags`

#### Scenario: References stay intact
- **WHEN** Ann's link widget on her dashboard shows the folder "Dev"
- **THEN** the folder ID in the widget's configuration is the ID of "Dev" in `favourites`

#### Scenario: Archived note
- **WHEN** Ann has an archived note
- **THEN** the note is in `notes` with `archived` `true`

#### Scenario: Records without files
- **WHEN** Ann has an attachment "invoice.pdf" and exports without files
- **THEN** `attachments` lists "invoice.pdf" with its content type and size, and `file` is `null`

#### Scenario: Deleted attachment
- **WHEN** Ann removed an attachment from her note and it is marked as deleted
- **THEN** it is not in `attachments`

#### Scenario: Only own data
- **WHEN** Ann and Bob both have notes and Ann exports her data
- **THEN** the document contains none of Bob's data

#### Scenario: Offline copy without files
- **WHEN** Ann's link "Go tour" has an offline copy with an article and a screenshot, and she exports without files
- **THEN** the link's `offlineCopy` has its `capturedAt` and `title`, and `article` and `screenshot` are `null`

#### Scenario: Link without offline copy
- **WHEN** Ann's link was imported and never captured
- **THEN** its `offlineCopy` is `null`

### Requirement: Data left out of the export
The export SHALL NOT contain passkeys, app tokens (neither their secrets nor their names), sessions, sign-in links or codes, favicons, background jobs, the state of a scheduled account deletion, or any data of other users.

#### Scenario: Passkeys and app tokens
- **WHEN** Ann has two passkeys and a Karakeep app token and exports her data
- **THEN** the document contains nothing about her passkeys or app tokens

#### Scenario: Favicons
- **WHEN** Ann's links show favicons and she exports with files
- **THEN** the archive contains no favicon files and the document lists no favicons

### Requirement: Export archive
With `files=true`, the ZIP archive SHALL contain `boardly.json`, the export document with `withFiles` `true`, and the files of the images, attachments and offline copies in the document:

- each image at `images/<id>.<extension>`, the extension following its content type,
- each attachment at `attachments/<id>/<file name>`, where the file name is the attachment's name with path separators and characters not allowed in file names replaced, and never a name that leads out of the directory,
- each offline copy's article at `offline/<link id>/article.html` and its screenshot at `offline/<link id>/screenshot.jpg`.

The `file` of each image and attachment, and the `article` and `screenshot` of each offline copy, SHALL be its path in the archive. A file that can't be read SHALL be left out of the archive, and its record keeps `null` in its place. The files SHALL keep their stored content unchanged.

#### Scenario: Image and attachment files
- **WHEN** Ann has an uploaded icon (PNG) and the attachment "invoice.pdf" and exports with files
- **THEN** the archive contains `boardly.json`, `images/<id>.png` and `attachments/<id>/invoice.pdf`, and the records' `file` are these paths

#### Scenario: Unsafe file name
- **WHEN** an attachment is named `../../etc/passwd`
- **THEN** its file is inside `attachments/<id>/` in the archive, and no path in the archive leaves the archive's root

#### Scenario: Missing file
- **WHEN** the file of one of Ann's images is missing on the server
- **THEN** the archive is still created, the image's record has `file` `null`, and all other files are included

#### Scenario: Offline copy files
- **WHEN** Ann's link with the ID `L1` has an offline copy with an article and a screenshot, and she exports with files
- **THEN** the archive contains `offline/L1/article.html` and `offline/L1/screenshot.jpg`, and the link's `offlineCopy` points to them

### Requirement: Export rate limit
Exports SHALL be limited per user over two sliding windows: at most `BOARDLY_EXPORTS_PER_HOUR` exports in the last 60 minutes (default `3`) and at most `BOARDLY_EXPORTS_PER_DAY` exports in the last 24 hours (default `10`). Exports with and without files count towards the same limits. An export counts when it starts, also when the download is cancelled or fails afterwards. A request over either limit SHALL get `429` with code `rate_limited` and a `Retry-After` header with the number of seconds until the next export is allowed, and nothing is exported. `0` turns the limit of its window off. A value that isn't a non-negative integer SHALL stop the server from starting with an error that names the variable. The counts SHALL NOT survive a restart of the server.

#### Scenario: Hourly limit
- **WHEN** Ann has exported 3 times in the last 60 minutes with the default settings and exports again
- **THEN** the response is `429` with code `rate_limited`, and `Retry-After` is the number of seconds until her oldest of these exports is 60 minutes old

#### Scenario: Daily limit
- **WHEN** Ann has exported 10 times in the last 24 hours, never more than 3 within 60 minutes, and exports again
- **THEN** the response is `429` with code `rate_limited`

#### Scenario: Window passes
- **WHEN** Ann exported 3 times between 10:00 and 10:05 and exports again at 11:01
- **THEN** the export succeeds

#### Scenario: Per user
- **WHEN** Ann has reached the hourly limit and Bob exports his data
- **THEN** Bob's export succeeds

#### Scenario: Configured limits
- **WHEN** the server runs with `BOARDLY_EXPORTS_PER_HOUR=0` and `BOARDLY_EXPORTS_PER_DAY=5` and Ann exports for the sixth time within 24 hours
- **THEN** the response is `429` with code `rate_limited`

#### Scenario: Invalid setting
- **WHEN** the server starts with `BOARDLY_EXPORTS_PER_DAY=-1` or `=many`
- **THEN** it exits with an error naming `BOARDLY_EXPORTS_PER_DAY`

### Requirement: Export section
The "Import / Export" settings tab SHALL show, after the import section, a section "Export" / "Export" with:

- a short text that the export contains all dashboards, favourites, notes and media sources, and not passkeys or connected apps,
- a checkbox "Include files (images and attachments)" / "Dateien einschließen (Bilder und Anhänge)", unchecked at first, with a hint that the download is then a ZIP file and can be large,
- a button "Export everything" / "Alles exportieren".

The button SHALL request the export with `files` `true` when the checkbox is checked and `false` otherwise, and save the response as a file with the file name from the server. While the export runs, the button is disabled and shows that the export is in progress. A `429` response SHALL show that the export limit is reached and the local time when the next export is possible, taken from `Retry-After`. Other errors are shown in the section with a general error message.

#### Scenario: Default without files
- **WHEN** the user clicks "Export everything" without touching the checkbox
- **THEN** the export is requested with `files` `false`, and a `.json` file is saved

#### Scenario: With files
- **WHEN** the user checks "Include files (images and attachments)" and clicks "Export everything"
- **THEN** the export is requested with `files` `true`, and a `.zip` file is saved

#### Scenario: In progress
- **WHEN** the export is running
- **THEN** the button is disabled and shows that the export is in progress

#### Scenario: Limit reached
- **WHEN** the export fails with `429` and `Retry-After` `1200` at 14:00 local time
- **THEN** the section shows that the export limit is reached and that the next export is possible at 14:20
