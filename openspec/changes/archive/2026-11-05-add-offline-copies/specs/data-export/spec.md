# Spec Delta

## MODIFIED Requirements

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
