# Spec Delta

## MODIFIED Requirements

### Requirement: Import endpoint
`POST /api/import` SHALL import a Boardly export (see the `data-export` capability) into the signed-in user's account. The request body is the file itself, sent as `application/octet-stream`: the export document as JSON, or the export archive as ZIP, told apart by the content of the body. The query parameters are:

- `parts`: the parts to import, comma-separated, each one of `favourites`, `dashboards`, `notes`, `media` and `feeds`, at least one,
- `folderId`: the ID of the import folder, required when `parts` contains `favourites`,
- `categoryId`: the ID of the import category, optional, used only when `parts` contains `notes`. Missing or empty means the top level,
- `skipDuplicates`: `true` or `false`, optional. Missing means `true`.

The file SHALL be at most 10 MB, and a larger body fails with `413` and code `file_too_large`. A missing or empty `parts`, an unknown part, a `skipDuplicates` other than `true` or `false`, or an empty body fails with `400` and code `invalid_request`. A body that isn't JSON or ZIP, a ZIP without `boardly.json`, or a document whose `format` isn't `"boardly"` or whose `version` isn't `1` or `2`, fails with `400` and code `invalid_import_file`. A `folderId` that isn't a folder of the user fails with `400` and code `invalid_folder`, and a `categoryId` that isn't a category of the user with `400` and code `invalid_category`. Form posts are rejected with `403` like for every API endpoint. The endpoint requires a signed-in user, and app tokens can't use it.

#### Scenario: Import a JSON export
- **WHEN** a signed-in user posts a Boardly JSON export with `parts` `favourites,dashboards,notes,media` and the ID of their root folder
- **THEN** the response is `200` with the import report, and the favourites, dashboards, notes and media sources of the file are added to the account

#### Scenario: Import a ZIP export
- **WHEN** the user posts a Boardly ZIP export with `parts` `notes`
- **THEN** the notes are added with their attachments from the archive

#### Scenario: Unknown version
- **WHEN** the user posts a document with `format` `"boardly"` and `version` `3`
- **THEN** the response is `400` with code `invalid_import_file`, and nothing is imported

#### Scenario: Export of the earlier version
- **WHEN** the user posts a document with `format` `"boardly"` and `version` `1`, which has no `feeds`
- **THEN** the response is `200`, and no feed is imported

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
An import SHALL only add data to the account. It SHALL NOT change or delete existing favourites, tags, dashboards, notes, categories, media sources, feed subscriptions, feed items, images or attachments, except that tags are attached to imported links and that the default folder and default note category may be set (see "Profile defaults"). Every imported item SHALL get a new ID, whatever its ID in the file. The import is all or nothing: when it fails, no data is added.

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
- `media`: the media sources,
- `feeds`: the feed subscriptions and the flagged feed items.

An image of the file SHALL be imported only when an imported link uses it as preview, or an imported category, note, media source, feed subscription or widget uses it as icon or image. An attachment SHALL be imported only when an imported note uses it. An image or attachment whose ID is already one of the user's own images or attachments SHALL NOT be imported again: the existing one is used, so importing an export into the account it came from doesn't copy its files.

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
Every reference in an imported item to another item of the file SHALL point to that item's new ID when that item was imported in the same import, or was matched to an existing item of the user: a reused folder or category, a skipped duplicate link or note, a matched tag, a feed subscription the user already has. This applies to the parent of folders and categories, the category of notes, the attachments in a note's content, the preview image of links, the icons of categories, notes, media sources and feed subscriptions, and every ID in a widget's configuration. Any other ID in a widget's configuration SHALL be kept as it is in the file. An icon whose image is not imported SHALL be removed. An attachment in a note's content that is not imported SHALL be removed from the content.

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

#### Scenario: Widget follows its feed
- **WHEN** an export has a feed widget showing a feed and is imported with `parts` `feeds,dashboards`
- **THEN** the imported widget shows the imported subscription of that feed

#### Scenario: Icon without image
- **WHEN** a JSON export without files has a note whose icon is an uploaded image, and it is imported with `parts` `notes`
- **THEN** the note is imported without an icon

#### Scenario: Attachment without file
- **WHEN** a JSON export without files has a note with the attachment "invoice.pdf", and it is imported
- **THEN** the note is imported without that attachment in its content

### Requirement: Data import report
A successful import SHALL return a report with, for each imported part:

- `favourites`: the bookmark import's counts of links, folders, duplicates, invalid URLs and shortened descriptions, and the number of created tags,
- `dashboards`: the number of added dashboards and of skipped widgets,
- `notes`: the number of imported notes and created categories, and of duplicate notes,
- `media`: the number of added media sources, of duplicates and of invalid ones,
- `feeds`: the number of added subscriptions, of duplicates and of invalid ones, and the number of restored and of skipped flagged items,

and `files`: the number of imported images, attachments and offline copy parts, and of skipped files. A part that wasn't imported is `null`.

#### Scenario: Report of a full import
- **WHEN** an export with 10 links, 2 dashboards, 5 notes and 3 stations, one of them already saved, is imported with all parts
- **THEN** the report has 10 links, 2 dashboards, 5 notes, 2 media sources and 1 duplicate media source

#### Scenario: Offline copy files counted
- **WHEN** a ZIP export with one link whose offline copy has an article and a screenshot is imported with `favourites`
- **THEN** the report's `files` counts 2 imported files

### Requirement: Boardly format in the import form
When the format "Boardly export (JSON or ZIP)" / "Boardly-Export (JSON oder ZIP)" is chosen in the import form, the form SHALL show a checkbox for each part, all checked at first: "Favourites" / "Favoriten", "Dashboards" / "Dashboards", "Notes" / "Notizen", "Media sources" / "Medienquellen" and "Feeds" / "Feeds". The import folder is shown only while "Favourites" is checked, and the import category only while "Notes" is checked, with a hint that notes without a category go into it. The "Import" button is disabled while no part is checked, and while "Favourites" is checked without an import folder. The import is sent to `/api/import` with the checked parts. After a successful import, the section SHALL show the report, and favourites, tags, dashboards, notes, categories, media sources and feeds are reloaded wherever they are shown. A `413` SHALL show that the file can have at most 10 MB, with the hint "Export without files to get a smaller JSON file." / "Exportiere ohne Dateien, um eine kleinere JSON-Datei zu erhalten."

#### Scenario: Parts checked at first
- **WHEN** the user chooses the format "Boardly export (JSON or ZIP)"
- **THEN** all five part checkboxes are checked, and the import folder and the import category are shown

#### Scenario: No folder without favourites
- **WHEN** the user unchecks "Favourites"
- **THEN** the import folder isn't shown, the button is enabled without a folder, and `parts` doesn't contain `favourites`

#### Scenario: No part
- **WHEN** the user unchecks all five parts
- **THEN** the "Import" button is disabled

#### Scenario: Too large
- **WHEN** the import fails with `413`
- **THEN** the section shows that the file can have at most 10 MB and the hint to export without files

## ADDED Requirements

### Requirement: Imported feeds
With `feeds`, every subscribed feed of the file SHALL be added after the user's subscriptions, in the order of the file, with its custom title and icon. A feed the user already subscribes to, by URL, SHALL be skipped and counted as a duplicate, and references to it point to the existing subscription. A feed whose URL isn't an absolute `http` or `https` URL is skipped and counted as invalid. An import SHALL NOT fetch anything: a feed the server doesn't have yet is created from its URL alone, with the title in the file ignored, and is checked and filled by its first refresh, under the fetch rules of the `feeds` capability. Until then, its title is the host of its URL.

A flagged item of the file SHALL set the user's flags on the stored item of the same feed and identity, and a flag the user already has stays as it is. An import SHALL NOT change the content of a stored item that another source already provided. An item the server doesn't have yet SHALL be created visible only to the importing user, until a refresh of the feed returns the same identity and replaces its content, after which it is an ordinary item. An item that exists visible only to another user SHALL be skipped and counted as skipped. A flagged item whose feed isn't in the file is skipped and counted as skipped.

#### Scenario: Restore feeds
- **WHEN** a new user imports an export with two subscribed feeds and one liked item
- **THEN** they subscribe to both feeds, and the item is in their "Liked" view

#### Scenario: Feed already subscribed
- **WHEN** Ann imports her own export with `parts` `feeds`
- **THEN** no subscription is added, duplicates are counted, and her flags are unchanged

#### Scenario: A file can't change what others see
- **WHEN** Mallory imports a crafted file that gives an item of Ann's feed a new title
- **THEN** the item's title is unchanged for Ann and for Mallory, and Mallory has a flag on it

#### Scenario: A file can't add items for others
- **WHEN** Mallory imports a crafted file with a flagged item, of a feed Ann subscribes to, that the server doesn't have
- **THEN** Mallory sees the item in her view and Ann doesn't see it

#### Scenario: Imported item confirmed by the feed
- **WHEN** a user imported a flagged item the server didn't have, and a refresh returns an entry with the same identity
- **THEN** the item shows the feed's content, is visible to all subscribers and keeps the user's flag

#### Scenario: No outbound request during import
- **WHEN** a file with a feed the server doesn't have is imported
- **THEN** the import makes no request to the feed's host
