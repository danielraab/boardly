## MODIFIED Requirements

### Requirement: Export document
The export document SHALL be a JSON object with:

- `format`: `"boardly"`,
- `version`: `3`,
- `exportedAt`: the time of the export,
- `withFiles`: whether the export contains the files,
- `profile`: the email address, display name, slug, creation time, the ID of the default folder for new links and the ID of the default category for new notes,
- `favourites`: the user's favourites tree, starting at the root folder, with every folder's children in their order. A folder has its ID, title, creation and change time and children. A link has its ID, URL, title, description, tag names, preview image ID, creation and change time, and `offlineCopy`. `offlineCopy` is `null` when the link has no offline copy with an article or a screenshot. Otherwise it is an object with `capturedAt`, `title`, `article` and `screenshot`. `article` and `screenshot` are the paths of these files inside the export archive, or `null` when the copy doesn't have that part, the export has no files, or the file couldn't be read,
- `tags`: all of the user's tags with ID, name and creation time, including tags no link uses,
- `dashboards`: every dashboard with its ID, slug, title, whether it is the default and whether it is public, its layout settings, creation and change time, and its widgets as a tree in their order, each with ID, kind, width and configuration as stored,
- `noteCategories`: every note category with ID, parent ID, title, icon, colour, and creation and change time,
- `notes`: every note, archived ones included, with ID, category ID, title, icon, colour, content document as stored, pinned and archived flags, the IDs of its attachments in their order, and creation and change time,
- `mediaSources`: every media source with ID, kind, title, URL, icon and position,
- `feeds`: every feed the user subscribes to or has a flagged or listened item in, with `id` (the subscription's ID, or the feed's ID without a subscription), `url`, `title` (the feed's title), `customTitle`, `icon`, `position` and `subscribed`,
- `feedItems`: every item the user flagged or has a listen record of, with `feedId` (the `id` in `feeds`), `guid` (the item's identity), `title`, `link`, `summary`, `publishedAt`, `audio`, `readLaterAt`, `likedAt` and `listen` (`{ "positionSeconds", "durationSeconds", "startedAt", "updatedAt", "finishedAt" }`, or `null` without a record),
- `images`: every image the user uploaded or that was fetched as a link preview for them, with ID, source (`upload` or `preview`), content type, size, creation time, and `file`,
- `attachments`: every attachment of the user that isn't deleted, with ID, file name, content type, size, creation time, and `file`.

IDs SHALL be the IDs the Boardly API uses for the same items, so that references inside the document (default folder and category, tag names, icon and preview image IDs, widget configurations, note content, attachments and the feed of a feed item) point to items in the same document. Icons have the same shape as in the API. Times are RFC 3339 timestamps in UTC. `file` is the path of the file inside the export archive, or `null` when the export has no files or the file couldn't be read. The records of images and attachments SHALL be included also when the export has no files. The state of offline copies (pending or failed) and their pending captures SHALL NOT be exported.

#### Scenario: Version and format
- **WHEN** Ann exports her data
- **THEN** the document has `format` `"boardly"`, `version` `3` and `exportedAt` set to the time of the export

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

#### Scenario: Flagged items only
- **WHEN** Ann subscribes to a feed with 100 items, has liked 2 of them and has listened to none
- **THEN** `feeds` lists the feed and `feedItems` lists only those 2 items

#### Scenario: Saved item of a removed feed
- **WHEN** Ann liked an item and no longer subscribes to its feed
- **THEN** `feeds` lists the feed with `subscribed` `false`, and `feedItems` lists the item

#### Scenario: Link without offline copy
- **WHEN** Ann's link was imported and never captured
- **THEN** its `offlineCopy` is `null`

#### Scenario: Listened episode
- **WHEN** Ann stopped an episode at 12:30 that she didn't flag
- **THEN** `feedItems` lists it with `readLaterAt` and `likedAt` `null` and `listen.positionSeconds` 750

#### Scenario: Flagged item without a record
- **WHEN** Ann liked an article
- **THEN** its entry in `feedItems` has `listen` `null`

### Requirement: Data left out of the export
The export SHALL NOT contain passkeys, app tokens (neither their secrets nor their names), sessions, sign-in links or codes, favicons, background jobs, the state of a scheduled account deletion, feed items without a flag or a listen record of the user, the refresh state of feeds, or any data of other users.

#### Scenario: Passkeys and app tokens
- **WHEN** Ann has two passkeys and a Karakeep app token and exports her data
- **THEN** the document contains nothing about her passkeys or app tokens

#### Scenario: Favicons
- **WHEN** Ann's links show favicons and she exports with files
- **THEN** the archive contains no favicon files and the document lists no favicons
