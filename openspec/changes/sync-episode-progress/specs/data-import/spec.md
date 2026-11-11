## MODIFIED Requirements

### Requirement: Import endpoint
`POST /api/import` SHALL import a Boardly export (see the `data-export` capability) into the signed-in user's account. The request body is the file itself, sent as `application/octet-stream`: the export document as JSON, or the export archive as ZIP, told apart by the content of the body. The query parameters are:

- `parts`: the parts to import, comma-separated, each one of `favourites`, `dashboards`, `notes`, `media` and `feeds`, at least one,
- `folderId`: the ID of the import folder, required when `parts` contains `favourites`,
- `categoryId`: the ID of the import category, optional, used only when `parts` contains `notes`. Missing or empty means the top level,
- `skipDuplicates`: `true` or `false`, optional. Missing means `true`.

The file SHALL be at most 10 MB, and a larger body fails with `413` and code `file_too_large`. A missing or empty `parts`, an unknown part, a `skipDuplicates` other than `true` or `false`, or an empty body fails with `400` and code `invalid_request`. A body that isn't JSON or ZIP, a ZIP without `boardly.json`, or a document whose `format` isn't `"boardly"` or whose `version` isn't `1`, `2` or `3`, fails with `400` and code `invalid_import_file`. A `folderId` that isn't a folder of the user fails with `400` and code `invalid_folder`, and a `categoryId` that isn't a category of the user with `400` and code `invalid_category`. Form posts are rejected with `403` like for every API endpoint. The endpoint requires a signed-in user, and app tokens can't use it.

#### Scenario: Import a JSON export
- **WHEN** a signed-in user posts a Boardly JSON export with `parts` `favourites,dashboards,notes,media` and the ID of their root folder
- **THEN** the response is `200` with the import report, and the favourites, dashboards, notes and media sources of the file are added to the account

#### Scenario: Import a ZIP export
- **WHEN** the user posts a Boardly ZIP export with `parts` `notes`
- **THEN** the notes are added with their attachments from the archive

#### Scenario: Unknown version
- **WHEN** the user posts a document with `format` `"boardly"` and `version` `4`
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

#### Scenario: Export of version 2
- **WHEN** the user posts a document with `format` `"boardly"` and `version` `2`, whose feed items have no `listen`
- **THEN** the response is `200`, and no listen record is created

### Requirement: Import parts
Only the parts in `parts` SHALL be imported:

- `favourites`: the favourites tree below its root, and the tags of the file, including unused ones, matched by name to the user's tags,
- `dashboards`: the dashboards with their widgets,
- `notes`: the note categories, the notes, and the attachments the notes use,
- `media`: the media sources,
- `feeds`: the feed subscriptions, and the feed items with flags or a listen record.

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

### Requirement: Data import report
A successful import SHALL return a report with, for each imported part:

- `favourites`: the bookmark import's counts of links, folders, duplicates, invalid URLs and shortened descriptions, and the number of created tags,
- `dashboards`: the number of added dashboards and of skipped widgets,
- `notes`: the number of imported notes and created categories, and of duplicate notes,
- `media`: the number of added media sources, of duplicates and of invalid ones,
- `feeds`: the number of added subscriptions, of duplicates and of invalid ones, and the number of restored and of skipped items with flags or a listen record,

and `files`: the number of imported images, attachments and offline copy parts, and of skipped files. A part that wasn't imported is `null`.

#### Scenario: Report of a full import
- **WHEN** an export with 10 links, 2 dashboards, 5 notes and 3 stations, one of them already saved, is imported with all parts
- **THEN** the report has 10 links, 2 dashboards, 5 notes, 2 media sources and 1 duplicate media source

#### Scenario: Offline copy files counted
- **WHEN** a ZIP export with one link whose offline copy has an article and a screenshot is imported with `favourites`
- **THEN** the report's `files` counts 2 imported files

### Requirement: Imported feeds
With `feeds`, every subscribed feed of the file SHALL be added after the user's subscriptions, in the order of the file, with its custom title and icon. A feed the user already subscribes to, by URL, SHALL be skipped and counted as a duplicate, and references to it point to the existing subscription. A feed whose URL isn't an absolute `http` or `https` URL is skipped and counted as invalid. An import SHALL NOT fetch anything: a feed the server doesn't have yet is created from its URL alone, with the title in the file ignored, and is checked and filled by its first refresh, under the fetch rules of the `feeds` capability. Until then, its title is the host of its URL.

An item of the file with flags or a listen record SHALL set the user's flags and listen record on the stored item of the same feed and identity. A flag the user already has stays as it is, and a listen record the user already has on the item stays as it is. A listen record of the file whose values "Listen API" of the `feeds` capability wouldn't accept, or on an item without audio, is left out, and the item's flags are still restored. An item with neither flags nor a listen record is skipped and counted as skipped. An import SHALL NOT change the content of a stored item that another source already provided. An item the server doesn't have yet SHALL be created visible only to the importing user, until a refresh of the feed returns the same identity and replaces its content, after which it is an ordinary item. An item that exists visible only to another user SHALL be skipped and counted as skipped. An item whose feed isn't in the file is skipped and counted as skipped.

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

#### Scenario: Restore the listen history
- **WHEN** a new user imports an export with an episode stopped at 12:30 and a finished episode
- **THEN** the first episode is in their "In progress" view at 12:30, and the second in "Finished"

#### Scenario: Existing record stays
- **WHEN** Ann has an episode at 40:00 and imports her older export with the same episode at 10:00
- **THEN** her record stays at 40:00
