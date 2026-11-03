# data-import Specification

## Purpose
Lets a signed-in user import a Boardly export, as a JSON file or a ZIP archive with files, into their account: chosen parts of it are added with new IDs, references between the imported items are kept, and nothing that exists is changed or deleted.

## Requirements

### Requirement: Import endpoint
`POST /api/import` SHALL import a Boardly export (see the `data-export` capability) into the signed-in user's account. The request body is the file itself, sent as `application/octet-stream`: the export document as JSON, or the export archive as ZIP, told apart by the content of the body. The query parameters are:

- `parts`: the parts to import, comma-separated, each one of `favourites`, `dashboards`, `notes` and `media`, at least one,
- `folderId`: the ID of the import folder, required when `parts` contains `favourites`,
- `categoryId`: the ID of the import category, optional, used only when `parts` contains `notes`. Missing or empty means the top level,
- `skipDuplicates`: `true` or `false`, optional. Missing means `true`.

The file SHALL be at most 10 MB, and a larger body fails with `413` and code `file_too_large`. A missing or empty `parts`, an unknown part, a `skipDuplicates` other than `true` or `false`, or an empty body fails with `400` and code `invalid_request`. A body that isn't JSON or ZIP, a ZIP without `boardly.json`, or a document whose `format` isn't `"boardly"` or whose `version` isn't `1`, fails with `400` and code `invalid_import_file`. A `folderId` that isn't a folder of the user fails with `400` and code `invalid_folder`, and a `categoryId` that isn't a category of the user with `400` and code `invalid_category`. Form posts are rejected with `403` like for every API endpoint. The endpoint requires a signed-in user, and app tokens can't use it.

#### Scenario: Import a JSON export
- **WHEN** a signed-in user posts a Boardly JSON export with `parts` `favourites,dashboards,notes,media` and the ID of their root folder
- **THEN** the response is `200` with the import report, and the favourites, dashboards, notes and media sources of the file are added to the account

#### Scenario: Import a ZIP export
- **WHEN** the user posts a Boardly ZIP export with `parts` `notes`
- **THEN** the notes are added with their attachments from the archive

#### Scenario: Unknown version
- **WHEN** the user posts a document with `format` `"boardly"` and `version` `2`
- **THEN** the response is `400` with code `invalid_import_file`, and nothing is imported

#### Scenario: Not a Boardly export
- **WHEN** the user posts a Karakeep JSON export to `/api/import`
- **THEN** the response is `400` with code `invalid_import_file`

#### Scenario: Unknown part
- **WHEN** the user posts a file with `parts` `favourites,passkeys`
- **THEN** the response is `400` with code `invalid_request`

#### Scenario: Favourites without folder
- **WHEN** the user posts a file with `parts` `favourites` and no `folderId`
- **THEN** the response is `400` with code `invalid_folder`

#### Scenario: Too large
- **WHEN** the user posts a ZIP of 11 MB
- **THEN** the response is `413` with code `file_too_large`, and nothing is imported

#### Scenario: Signed out
- **WHEN** a visitor without a session posts to the endpoint
- **THEN** the response is `401`

### Requirement: Imports only add
An import SHALL only add data to the account. It SHALL NOT change or delete existing favourites, tags, dashboards, notes, categories, media sources, images or attachments, except that tags are attached to imported links and that the default folder and default note category may be set (see "Profile defaults"). Every imported item SHALL get a new ID, whatever its ID in the file. The import is all or nothing: when it fails, no data is added.

#### Scenario: Restore into an empty account
- **WHEN** a new user imports the export of their old account with all parts into their root folder and the top-level category
- **THEN** their account has the favourites, dashboards, notes and media sources of the export

#### Scenario: Import the own export again
- **WHEN** Ann imports her own export with all parts and `skipDuplicates` `true`
- **THEN** no link and no note is added, every dashboard is added once more with a new slug, no media source is added, and nothing that existed is changed

#### Scenario: Failure adds nothing
- **WHEN** an import fails with an error from the database after some notes were written
- **THEN** the response is an error, and the account has none of the file's data

### Requirement: Import parts
Only the parts in `parts` SHALL be imported:

- `favourites`: the favourites tree below its root, and the tags of the file, including unused ones, matched by name to the user's tags,
- `dashboards`: the dashboards with their widgets,
- `notes`: the note categories, the notes, and the attachments the notes use,
- `media`: the media sources.

An image of the file SHALL be imported only when an imported link uses it as preview, or an imported category, note, media source or widget uses it as icon or image. An attachment SHALL be imported only when an imported note uses it. An image or attachment whose ID is already one of the user's own images or attachments SHALL NOT be imported again: the existing one is used, so importing an export into the account it came from doesn't copy its files.

#### Scenario: Dashboards only
- **WHEN** Ann imports an export with `parts` `dashboards`
- **THEN** the dashboards are added, and no favourites, tags, notes, categories, media sources or attachments are added

#### Scenario: Own files reused
- **WHEN** Ann imports her own ZIP export with `parts` `notes`, and her note "Plan" with the attachment "invoice.pdf" still exists
- **THEN** no attachment is added, and "Plan" is skipped as a duplicate

#### Scenario: Only used images
- **WHEN** an export with an uploaded note icon and a link preview image is imported with `parts` `favourites`
- **THEN** the link preview image is imported and the note icon is not

### Requirement: References between imported items
Every reference in an imported item to another item of the file SHALL point to that item's new ID when that item was imported in the same import, or was matched to an existing item of the user: a reused folder or category, a skipped duplicate link or note, a matched tag. This applies to the parent of folders and categories, the category of notes, the attachments in a note's content, the preview image of links, the icons of categories, notes and media sources, and every ID in a widget's configuration. Any other ID in a widget's configuration SHALL be kept as it is in the file. An icon whose image is not imported SHALL be removed. An attachment in a note's content that is not imported SHALL be removed from the content.

#### Scenario: Widget follows its folder
- **WHEN** an export has a link list widget showing the folder "Dev" and is imported with `parts` `favourites,dashboards`
- **THEN** the imported widget shows the imported folder "Dev"

#### Scenario: Reused folder
- **WHEN** Ann already has the folder "Dev" in the import folder and imports an export with "Dev" and a widget showing it
- **THEN** no second "Dev" is created, and the imported widget shows her existing "Dev"

#### Scenario: Reference kept for an item not imported
- **WHEN** Ann imports her own export with `parts` `dashboards`, and a widget shows her folder "Dev"
- **THEN** the imported widget still shows her existing folder "Dev"

#### Scenario: Reference to an item that doesn't exist
- **WHEN** a new user imports an export with `parts` `dashboards`, and a widget shows a folder of the old account
- **THEN** the widget is imported and shows that its folder doesn't exist, as for a deleted folder

#### Scenario: Icon without image
- **WHEN** a JSON export without files has a note whose icon is an uploaded image, and it is imported with `parts` `notes`
- **THEN** the note is imported without an icon

#### Scenario: Attachment without file
- **WHEN** a JSON export without files has a note with the attachment "invoice.pdf", and it is imported
- **THEN** the note is imported without that attachment in its content

### Requirement: Imported favourites
With `favourites`, the tree below the file's root SHALL be imported into the import folder following the bookmark import's rules for the folder structure, imported links, import order and duplicate URLs (see the `bookmark-import` capability). Each link keeps its URL, title, description, tags, creation time and, when the image is imported, its preview image. Imported links get the same background fetches as links of a bookmark import.

#### Scenario: Duplicate link
- **WHEN** Ann has `https://go.dev/` and imports an export with `https://go.dev` and `skipDuplicates` `true`
- **THEN** the link is skipped and counted as a duplicate, and references to it point to Ann's existing link

#### Scenario: Description and preview kept
- **WHEN** a ZIP export has a link with a description and a preview image
- **THEN** the imported link has that description and preview image

### Requirement: Imported notes
With `notes`, the categories of the file SHALL be imported below the import category, keeping their tree, following the bookmark import's rules for categories, and the notes into their imported categories, or into the import category when they have none. Duplicate notes are skipped by the bookmark import's rules for duplicate notes. Each note keeps its title, icon, colour, content, pinned and archived flags, and creation and change time. Each category keeps its title, icon and colour.

#### Scenario: Category tree kept
- **WHEN** an export has the category "Work" with the subcategory "Projects" holding the note "Plan", imported into the top level
- **THEN** "Work/Projects" exists and holds "Plan"

#### Scenario: Pinned and archived kept
- **WHEN** an export has a pinned note and an archived note
- **THEN** the imported notes are pinned and archived

### Requirement: Imported dashboards
With `dashboards`, every dashboard of the file SHALL be added with its title, layout settings and widgets in their order. An imported dashboard SHALL NOT be public and SHALL NOT be the default dashboard. Its slug is the slug of the file when the user has no dashboard with it, otherwise the slug followed by `-2`, `-3` and so on, the first one that is free. A widget whose configuration this server doesn't accept SHALL be left out and counted in the report.

#### Scenario: Never public
- **WHEN** an export has the public default dashboard "Home"
- **THEN** the imported "Home" is neither public nor the default dashboard

#### Scenario: Slug taken
- **WHEN** Ann has dashboards with the slugs `home` and `home-2` and imports an export with the dashboard slug `home`
- **THEN** the imported dashboard has the slug `home-3`

#### Scenario: Unknown widget kind
- **WHEN** an export has a widget of a kind this server doesn't know
- **THEN** the dashboard is imported without that widget, and the report counts one skipped widget

### Requirement: Imported media sources
With `media`, every media source of the file SHALL be added after the user's media sources, in the order of the file, with its kind, title, URL and icon. A media source whose URL equals the URL of one of the user's media sources, or of one imported before it, SHALL be skipped and counted as a duplicate. A media source of a kind this server doesn't know is skipped and counted as invalid.

#### Scenario: Duplicate station
- **WHEN** Ann has the station `https://radio.example.org/stream` and imports an export with the same URL
- **THEN** no second station is added, and the report counts one duplicate media source

### Requirement: Files from the archive
An image or attachment of a ZIP export SHALL be imported from the file named in its `file` and SHALL meet this server's rules for uploaded images and attachments (allowed types and maximum size). A file that doesn't, that is missing from the archive, or whose `file` is `null`, SHALL be skipped and counted in the report, and the items that use it are imported without it (see "References between imported items"). A JSON export has no files, so no image or attachment is imported from it.

#### Scenario: Attachment type not allowed here
- **WHEN** a ZIP export has a note with a `.docx` attachment and this server doesn't allow that type
- **THEN** the note is imported without the attachment, and the report counts one skipped file

#### Scenario: Missing file
- **WHEN** a ZIP export lists an image whose file isn't in the archive
- **THEN** the image is skipped and counted in the report

### Requirement: Profile defaults
When the user has no default folder for new links and the file's default folder was imported or matched, the default folder SHALL be set to it. When the user has no default category for new notes and the file's default category was imported or matched, the default category SHALL be set to it. The display name and the slug of the user SHALL NOT be changed.

#### Scenario: Default category set
- **WHEN** a new user without a default category imports an export whose default category is "Inbox"
- **THEN** their default category is the imported "Inbox"

#### Scenario: Existing default kept
- **WHEN** Ann has a default folder and imports an export with another default folder
- **THEN** her default folder doesn't change

### Requirement: Data import report
A successful import SHALL return a report with, for each imported part:

- `favourites`: the bookmark import's counts of links, folders, duplicates, invalid URLs and shortened descriptions, and the number of created tags,
- `dashboards`: the number of added dashboards and of skipped widgets,
- `notes`: the number of imported notes and created categories, and of duplicate notes,
- `media`: the number of added media sources, of duplicates and of invalid ones,

and `files`: the number of imported images and attachments, and of skipped files. A part that wasn't imported is `null`.

#### Scenario: Report of a full import
- **WHEN** an export with 10 links, 2 dashboards, 5 notes and 3 stations, one of them already saved, is imported with all parts
- **THEN** the report has 10 links, 2 dashboards, 5 notes, 2 media sources and 1 duplicate media source

### Requirement: Boardly format in the import form
When the format "Boardly export (JSON or ZIP)" / "Boardly-Export (JSON oder ZIP)" is chosen in the import form, the form SHALL show a checkbox for each part, all checked at first: "Favourites" / "Favoriten", "Dashboards" / "Dashboards", "Notes" / "Notizen" and "Media sources" / "Medienquellen". The import folder is shown only while "Favourites" is checked, and the import category only while "Notes" is checked, with a hint that notes without a category go into it. The "Import" button is disabled while no part is checked, and while "Favourites" is checked without an import folder. The import is sent to `/api/import` with the checked parts. After a successful import, the section SHALL show the report, and favourites, tags, dashboards, notes, categories and media sources are reloaded wherever they are shown. A `413` SHALL show that the file can have at most 10 MB, with the hint "Export without files to get a smaller JSON file." / "Exportiere ohne Dateien, um eine kleinere JSON-Datei zu erhalten."

#### Scenario: Parts checked at first
- **WHEN** the user chooses the format "Boardly export (JSON or ZIP)"
- **THEN** all four part checkboxes are checked, and the import folder and the import category are shown

#### Scenario: No folder without favourites
- **WHEN** the user unchecks "Favourites"
- **THEN** the import folder isn't shown, the button is enabled without a folder, and `parts` doesn't contain `favourites`

#### Scenario: No part
- **WHEN** the user unchecks all four parts
- **THEN** the "Import" button is disabled

#### Scenario: Too large
- **WHEN** the import fails with `413`
- **THEN** the section shows that the file can have at most 10 MB and the hint to export without files
