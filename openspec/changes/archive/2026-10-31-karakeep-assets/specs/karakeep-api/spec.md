# Spec Delta

## ADDED Requirements

### Requirement: Upload assets
`POST /api/karakeep/api/assets` with a `multipart/form-data` body SHALL store the part named `file` as an attachment of the token's owner and respond 200 with `{"assetId", "contentType", "size", "fileName"}`. Here `assetId` is the attachment's ID, `fileName` its name (the part's file name) and `size` its size in bytes as a number. The content type SHALL be found as for uploads in the web app (see "Uploading attachments" in `note-attachments`), and the part's own content type is ignored. The new attachment is not in any note, so it is marked for deletion like any unused upload when no bookmark uses it within a day.

The endpoint SHALL require the token (see "Bearer authentication for app endpoints" in `app-tokens`). Failures SHALL respond with `{"error": <message>}` and store nothing:

- 400 for a body that isn't multipart, a missing `file` part, a missing or invalid file name, an empty file, a type that is unknown or not allowed ("This type of file can't be attached"), or a token with the note scope `none`,
- 413 for a file larger than `BOARDLY_ATTACHMENT_MAX_SIZE_MB`, with a message naming the limit. The server stops reading at that size plus 64 KB.

#### Scenario: Share a photo
- **WHEN** the mobile app uploads `IMG_0042.jpg`, a JPEG of 2 MB, with a token
- **THEN** the server responds 200 with an `assetId`, `contentType` `image/jpeg`, `size` 2097152 and `fileName` `IMG_0042.jpg`, and the file appears on the user's attachments page

#### Scenario: Too large
- **WHEN** the maximum is 5 MB and the app uploads a 12 MB PDF
- **THEN** the server responds 413 with an `error` saying that attachments can be at most 5 MB, and the app shows it

#### Scenario: Type not allowed
- **WHEN** the allowed types are the defaults and the app uploads an SVG file
- **THEN** the server responds 400 with an `error`, and stores nothing

#### Scenario: Token without notes
- **WHEN** a token with the note scope `none` uploads a file
- **THEN** the server responds 400, and stores nothing

### Requirement: Serve assets to the app
`GET /api/karakeep/api/assets/{id}` SHALL require the token, and respond with the file as in "Serving attachments" in `note-attachments` (content type, disposition, `nosniff`, sandbox policy, range requests) when `{id}` is an attachment of the token's owner that a note in the token's note scope refers to, or that no note refers to. Otherwise it SHALL respond 404 with `{"error": "Not found"}`.

#### Scenario: View a shared photo
- **WHEN** the mobile app opens an asset bookmark and loads `GET <server address>/api/assets/<assetId>` with its token
- **THEN** the server responds 200 with the image

#### Scenario: Attachment outside the scope
- **WHEN** a token with the root category "Inbox" requests an attachment that only a note in "Work" refers to
- **THEN** the server responds 404

### Requirement: Asset bookmarks are notes
A note in the token's note scope whose content is exactly one attachment block, of an image type or `application/pdf`, SHALL be returned as an asset bookmark. It has the fields of "Notes are text bookmarks", except:

- `content`: `{"type": "asset", "assetType", "assetId", "fileName", "size", "sourceUrl": null, "content": null}`. `assetType` is `pdf` for `application/pdf` and `image` otherwise, `assetId` is the attachment's ID, `fileName` its name and `size` its size in bytes.
- `assets`: `[{"id": <attachment id>, "assetType": "bookmarkAsset", "fileName": <name>}]`.

An asset bookmark is a note in every other way. Updating its `title`, `archived` and `favourited` works as for notes (see "Update bookmarks"). Setting its `text` replaces the content, and the note then becomes a text bookmark. Deleting it deletes the note and not the file, which is marked and deleted by the cleanup when no other note refers to it (see "Marking and deleting unused attachments" in `note-attachments`).

#### Scenario: Shared photo
- **WHEN** a note "IMG_0042.jpg" has only the attachment block of `IMG_0042.jpg`
- **THEN** its bookmark has `content.type` `asset`, `content.assetType` `image`, `content.assetId` the attachment's ID, and the mobile app shows the photo

#### Scenario: PDF
- **WHEN** a note has only the attachment block of `invoice.pdf`
- **THEN** its bookmark has `content.assetType` `pdf`

#### Scenario: Photo with text
- **WHEN** Ann adds the paragraph "Our roses" to the note of a shared photo in the web app
- **THEN** the note is returned as a text bookmark, and its text contains the photo as a Markdown image

#### Scenario: Video
- **WHEN** a note has only the attachment block of a video
- **THEN** it is returned as a text bookmark

#### Scenario: Rename a shared file
- **WHEN** the app sets the title of an asset bookmark to "Roses"
- **THEN** the note's title is "Roses", and it is still an asset bookmark

#### Scenario: Delete a shared file
- **WHEN** the app deletes an asset bookmark
- **THEN** the note is deleted, and the file is still listed on the attachments page until the cleanup deletes it

## MODIFIED Requirements

### Requirement: Create bookmarks
`bookmarks.createBookmark` with `type` `link`, `url` and an optional `title` SHALL create a link as the first node of the token's root folder and return it as a bookmark with `alreadyExists: false`. When a bookmark with the same URL already exists in the subtree (compared as in "Check a URL"), it SHALL return that bookmark with `alreadyExists: true` and create nothing. An invalid URL or title fails with `BAD_REQUEST` (the rules of link nodes apply).

`bookmarks.createBookmark` with `type` `text`, `text` and an optional `title` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, with the title and the text read as Markdown (see "Content from Markdown" in the note-document spec), and return it as a bookmark with `alreadyExists: false`. Notes are never treated as duplicates. A title longer than the note title limit, or a text whose document would be larger than allowed, fails with `BAD_REQUEST`. `sourceUrl` is ignored. A token with the note scope `none` fails with `BAD_REQUEST`.

`bookmarks.createBookmark` with `type` `asset`, `assetId` and an optional `title` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, whose content is only the attachment block of that file, and return it as a bookmark (see "Asset bookmarks are notes") with `alreadyExists: false`. The note's title is `title` when it is given and not empty, otherwise the file's name, cut to the note title limit. `assetId` must be an attachment of the token's owner (see "Upload assets"), otherwise the call fails with `BAD_REQUEST`. `assetType`, `fileName` and `sourceUrl` are ignored, because Boardly knows the file's type and name. A token with the note scope `none` fails with `BAD_REQUEST`.

`source`, `crawlPriority`, `createdAt` and `precrawledArchiveId` are ignored; a non-empty `note` or `summary` fails with `BAD_REQUEST`. For a text or asset bookmark, `archived` sets the new note's archived flag and `favourited` its pinned flag; for a link, `archived` or `favourited` `true` fails with `BAD_REQUEST`.

#### Scenario: Save a page from the extension
- **WHEN** the extension saves `https://example.org` with title "Example"
- **THEN** a link "Example" is the first node in the token's root folder, and the result has `alreadyExists: false`

#### Scenario: Save a page twice
- **WHEN** the extension saves a URL that is already a link in the subtree
- **THEN** the existing bookmark is returned with `alreadyExists: true` and no link is added

#### Scenario: Save a note
- **WHEN** the mobile app creates a bookmark of type `text` with the text "- [ ] buy milk" and the token's root category is "Inbox"
- **THEN** a note with a task list "buy milk" is created in "Inbox", and the result is a text bookmark with `alreadyExists: false`

#### Scenario: Save a note with the top level
- **WHEN** a token with the note scope `all` creates a text bookmark
- **THEN** the note is uncategorized

#### Scenario: Save a favourite note
- **WHEN** the mobile app creates a bookmark of type `text` with `favourited: true`
- **THEN** the new note is pinned, and the result has `favourited` `true`

#### Scenario: Save an archived link
- **WHEN** the extension creates a bookmark of type `link` with `archived: true`
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

#### Scenario: Save a file
- **WHEN** the mobile app uploads `IMG_0042.jpg` and creates a bookmark of type `asset` with its `assetId` and `assetType` `image`, and the token's root category is "Inbox"
- **THEN** a note "IMG_0042.jpg" with only that photo is created in "Inbox", `/notes/<id>` shows the photo, and the result is an asset bookmark with `alreadyExists: false`

#### Scenario: Unknown file
- **WHEN** the app creates a bookmark of type `asset` with an `assetId` that isn't one of the owner's attachments
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

### Requirement: Notes are text bookmarks
A note in the token's note scope (see "Token subtree scope" in the app-tokens spec) SHALL also be a Karakeep bookmark; its ID is the note's ID. Unless it is an asset bookmark (see "Asset bookmarks are notes"), it SHALL be returned in Karakeep's bookmark shape with:

- `id`, `userId` (the token owner), `title` (the note's title or `null`), `createdAt` and `modifiedAt` (the note's creation and last-change time, as dates),
- `tags`: an empty array,
- `content`: `{"type": "text", "text", "sourceUrl": null}`, where `text` is the note's content as Markdown (see "Content as Markdown" in the note-document spec),
- `archived` the note's archived flag and `favourited` its pinned flag; `note`, `summary`, `source`, `taggingStatus`, `summarizationStatus` and `embeddingStatus` `null`; `assets` an empty array.

`bookmarks.getBookmark` SHALL return a note in scope like a link; a note outside the scope fails with `NOT_FOUND`.

#### Scenario: Note as a bookmark
- **WHEN** a token with the note scope `all` asks for a note "Groceries" with the task list `- [ ] milk`
- **THEN** `bookmarks.getBookmark` returns a bookmark with title "Groceries", `content.type` `text`, `content.text` `- [ ] milk` and no tags

#### Scenario: Pinned and archived note
- **WHEN** a note in scope is pinned and archived
- **THEN** its bookmark has `favourited` `true` and `archived` `true`

#### Scenario: Note outside the scope
- **WHEN** a token with the root category "Inbox" asks for a note in "Work"
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Linkwarden token has no notes
- **WHEN** a Linkwarden token calls the Linkwarden API
- **THEN** no note is returned or changed
