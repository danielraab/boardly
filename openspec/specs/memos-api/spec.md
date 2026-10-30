# memos-api Specification

## Purpose
Defines the Memos v1 REST endpoints that the MoeMemos Android app uses to log in, sync, write, pin, archive and delete memos, their wire format, and how memos map onto the notes in a Memos app token's note scope.

## Requirements

### Requirement: Memos wire format
MoeMemos talks to `<server address>/api/v1/…`. Boardly SHALL answer it under `/api/memos/api/v1/`. Request and response bodies are JSON with Memos' camelCase field names. Times SHALL be RFC 3339 strings in UTC with millisecond precision, and the same time SHALL always be written the same way. Errors SHALL be Memos errors `{"code": <gRPC code>, "message": <text>, "details": []}` with the matching HTTP status: `3` with 400 for invalid input, `5` with 404 for something that doesn't exist or is outside the token's scope, `16` with 401 for a missing or invalid token (see "Bearer authentication for app endpoints" in `app-tokens`). Any other path under `/api/memos/` SHALL respond 404 with code `5`, among them `GET /api/memos/api/v1/status`, so that MoeMemos doesn't take Boardly for a Memos server with the legacy API.

#### Scenario: Error format
- **WHEN** MoeMemos calls `DELETE /api/memos/api/v1/memos/doesnotexist`
- **THEN** the server responds 404 with `{"code": 5, "message": …, "details": []}`

#### Scenario: Legacy status endpoint
- **WHEN** MoeMemos calls `GET /api/memos/api/v1/status` while connecting
- **THEN** the server responds 404, and MoeMemos goes on to `instance/profile`

#### Scenario: Stable times
- **WHEN** MoeMemos lists the same unchanged memo twice
- **THEN** its `updateTime` is the same string both times

### Requirement: Memos instance profile
`GET /api/memos/api/v1/instance/profile` SHALL be public and respond 200 with `{"version": "0.31.0", "mode": "prod", "instanceUrl": <BOARDLY_BASE_URL>}`. `0.31.0` is the newest Memos version that MoeMemos supports, so MoeMemos accepts Boardly without an "unsupported version" warning.

#### Scenario: Connect
- **WHEN** MoeMemos calls `instance/profile` without a token
- **THEN** the server responds 200 with `version` `0.31.0`

### Requirement: Memos user endpoints
Boardly SHALL answer the user endpoints MoeMemos needs, for the token owner only. A user is returned as `{"name": "users/<user id>", "role": "USER", "username": <email>, "email": <email>, "displayName": <display name, or the email when there is none>, "avatarUrl": "", "description": "", "state": "NORMAL", "createTime", "updateTime"}`.

- `GET /api/v1/auth/me` SHALL respond with `{"user": <the token owner>}`.
- `GET /api/v1/users/{id}` SHALL respond with the token owner when `{id}` is their ID, and 404 otherwise.
- `GET /api/v1/users/{id}/settings/GENERAL` SHALL respond with `{"name": "users/<id>/settings/GENERAL", "generalSetting": {"locale": "", "memoVisibility": "PRIVATE", "theme": ""}}` for the token owner, and 404 otherwise.
- `GET /api/v1/users/{id}:getStats` SHALL respond with `{"name": "users/<id>/stats", "tagCount": {}}` for the token owner, and 404 otherwise.

#### Scenario: Log in
- **WHEN** Ann enters the server address and an active Memos token in MoeMemos
- **THEN** `auth/me` returns `users/<Ann's ID>` with Ann's email as `username`, and MoeMemos shows Ann's display name

#### Scenario: Other user
- **WHEN** Ann's token asks for `GET /api/memos/api/v1/users/<Bob's ID>`
- **THEN** the server responds 404

### Requirement: Memos are notes in the token's note scope
A memo SHALL be a note in the token's note scope (see "Token subtree scope" in `app-tokens`); its name is `memos/<note id>`. Every endpoint in this spec SHALL see and change only these notes. A memo SHALL be returned as:

- `name`, `creator` (`users/<owner id>`), `createTime` and `displayTime` (the note's creation time), `updateTime` (the note's time of last change),
- `state`: `ARCHIVED` for an archived note, else `NORMAL`; `pinned`: the note's pinned flag,
- `content`: the note's content without its trailing attachments as Markdown (see "Content as Markdown" in `note-document`), preceded by `# <title>` and a blank line when the note has a title,
- `attachments`: the note's trailing attachments (see "Trailing attachments are the memo's attachments" and "Memo attachments"),
- `visibility` `PRIVATE`; `tags` and `relations` `[]`; no `location`.

#### Scenario: Titled note
- **WHEN** a note "Groceries" has the task list `- [ ] milk`
- **THEN** its memo has the content `# Groceries\n\n- [ ] milk` and no attachments

#### Scenario: Untitled note
- **WHEN** a note without a title has the paragraph "call the plumber"
- **THEN** its memo has the content `call the plumber`

#### Scenario: Pinned and archived
- **WHEN** a note is pinned and archived
- **THEN** its memo has `pinned` `true` and `state` `ARCHIVED`

#### Scenario: Note outside the scope
- **WHEN** a token with the root category "Inbox" asks to change a note in "Work"
- **THEN** the server responds 404

### Requirement: Title from a leading heading
When the content of a memo is written (on create or update), Boardly SHALL read it as Markdown (see "Content from Markdown" in `note-document`). When the first block is a level-1 heading whose text, trimmed, is 1 to 200 characters long, that text SHALL become the note's title and the heading SHALL be left out of the content. Otherwise the note SHALL have no title and keep every block as content. The text of the content is then written as in "Content replaced from Markdown" for updates, so tasks keep their IDs.

#### Scenario: Rename by heading
- **WHEN** the note "Groceries" is edited in MoeMemos and its first line becomes `# Shopping`
- **THEN** the note's title is "Shopping", and its content doesn't contain the heading

#### Scenario: Heading removed
- **WHEN** the first line `# Groceries` is removed in MoeMemos
- **THEN** the note has no title

#### Scenario: Second-level heading
- **WHEN** a memo is created with the content `## Week\n\n- [ ] review`
- **THEN** the note has no title, and its content starts with a heading of level 2 "Week"

#### Scenario: Round trip
- **WHEN** MoeMemos saves a memo's content unchanged
- **THEN** the note's title and content are unchanged, and its tasks keep their IDs

### Requirement: List memos
`GET /api/v1/memos` SHALL return `{"memos", "nextPageToken"}` with the memos in scope, newest first by creation time, then by ID. The parameters are:

- `state`: `NORMAL` (the default) for notes that aren't archived, `ARCHIVED` for archived notes.
- `pageSize`: 1 to 1000; missing or `0` means 50, and larger values mean 1000.
- `pageToken`: the `nextPageToken` of the previous page; empty or missing starts at the beginning. An invalid token responds 400.
- `filter`: a Memos filter made of conditions joined with `&&`. `creator == "users/<id>"` matches every memo when `<id>` is the token owner's ID and no memo otherwise. `visibility in [<values>]` and `visibility == <value>` match every memo when the values include `PRIVATE` and no memo otherwise. Other conditions are ignored.

`nextPageToken` SHALL be a non-empty token when more memos follow, else `""`.

#### Scenario: Sync
- **WHEN** the scope has 250 notes that aren't archived and MoeMemos lists with `pageSize=200`, `state=NORMAL` and the filter `creator == "users/<owner id>"`, then again with the returned token
- **THEN** the first page has the 200 newest memos and a token, and the second page has the other 50 and `nextPageToken` `""`

#### Scenario: Archive list
- **WHEN** MoeMemos lists with `state=ARCHIVED`
- **THEN** only memos of archived notes are returned

#### Scenario: Explore
- **WHEN** MoeMemos lists with the filter `visibility in ["PUBLIC", "PROTECTED"]`
- **THEN** the result has no memos

#### Scenario: Notes outside the scope
- **WHEN** a token has the root category "Home", and the user has notes in "Home/Garden", "Work" and uncategorized
- **THEN** the list has only the notes of "Home" and "Home/Garden"

### Requirement: Create memos
`POST /api/v1/memos` with `content` and optional `visibility`, `attachments`, `createTime` and `location` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, with the title and content from `content` (see "Title from a leading heading") followed by the listed attachments (see "Trailing attachments are the memo's attachments"), neither pinned nor archived, and respond 200 with the memo. `createTime` SHALL become the note's creation time, so memos written offline keep their date; a time in the future, or a missing one, means now. `visibility` and `location` are ignored. A missing `content`, an unknown attachment (see "Memo attachments"), or a content whose document would be larger than allowed, responds 400, and nothing is created.

#### Scenario: Write a memo
- **WHEN** MoeMemos creates a memo "milk\neggs" with a token whose root category is "Inbox"
- **THEN** a note without a title with that text is created in "Inbox", and `/notes` shows it

#### Scenario: Written offline
- **WHEN** MoeMemos creates a memo with `createTime` of yesterday 08:00 UTC
- **THEN** the note's creation time is yesterday 08:00 UTC

#### Scenario: Memo with a photo
- **WHEN** MoeMemos uploads `roses.jpg` and then creates a memo "Roses are out" with that attachment
- **THEN** a note is created with the paragraph "Roses are out" followed by the attachment block of `roses.jpg`, `/notes/<id>` shows the photo, and the returned memo lists `roses.jpg` in `attachments`

### Requirement: Update memos
`PATCH /api/v1/memos/{id}` SHALL change the note of a memo in scope and respond 200 with the memo. With `updateMask`, a comma-separated list of field names in snake case, only the listed fields of the body are applied; without it, every field present in the body is applied. The fields are:

- `content`: the title and the content before the trailing attachments, as in "Title from a leading heading". The note's trailing attachments stay unless `attachments` is applied too. Content equal to the memo's current content leaves the note's title and content unchanged.
- `attachments`: the note's trailing attachments, replacing the current ones (see "Trailing attachments are the memo's attachments"); an empty list removes them. Removing an attachment from a note doesn't delete the file.
- `pinned`: the note's pinned flag.
- `state`: `ARCHIVED` archives the note, `NORMAL` unarchives it; other values respond 400.
- `visibility`, `update_time`, `create_time`, `display_time` and `location` are ignored.

The change SHALL be applied to the note's current version, so a change from MoeMemos always wins, and it counts as a change of the note (version and time of last change) when anything changed. A memo that doesn't exist or is outside the scope responds 404. An unknown attachment (see "Memo attachments") responds 400. A request that fails changes nothing.

#### Scenario: Tick off a task
- **WHEN** MoeMemos changes `- [ ] milk` to `- [x] milk` in a memo
- **THEN** the task "milk" is checked in Boardly, keeps its ID, and the note's version goes up by one

#### Scenario: Pin
- **WHEN** MoeMemos sends `{"pinned": true}` for a memo
- **THEN** the note is pinned and shown first on `/notes`

#### Scenario: Archive
- **WHEN** MoeMemos sends `{"state": "ARCHIVED"}` for a memo
- **THEN** the note is archived, and the next list with `state=NORMAL` doesn't contain it

#### Scenario: Open in the web editor
- **WHEN** Ann has a note open in the web editor and changes it in MoeMemos, then types in the web editor
- **THEN** MoeMemos' change is saved, and the web editor says the note was changed elsewhere

#### Scenario: Attachment on update
- **WHEN** a memo has the content "Roses" and no attachments, and MoeMemos sends the same content with the attachments `roses.jpg`
- **THEN** the note ends with the attachment block of `roses.jpg`, and its version goes up by one

#### Scenario: Remove a photo
- **WHEN** a memo has the attachments `roses.jpg` and `plan.pdf`, and MoeMemos sends its content with the attachments `plan.pdf`
- **THEN** the note ends with the attachment block of `plan.pdf` only, and `roses.jpg` still exists

#### Scenario: Edit text only
- **WHEN** a memo has the attachment `roses.jpg`, and MoeMemos sends `{"content": "Roses are out"}` without `attachments`
- **THEN** the note's text is "Roses are out", and it still ends with `roses.jpg`

#### Scenario: Unchanged memo
- **WHEN** MoeMemos sends a memo's current content and attachments unchanged
- **THEN** the note's version and time of last change stay the same

#### Scenario: Inline attachment kept
- **WHEN** a note has an image block followed by a paragraph, and MoeMemos sends its content unchanged with an empty attachment list
- **THEN** the image block stays where it is

### Requirement: Delete memos
`DELETE /api/v1/memos/{id}` SHALL delete the note of a memo in scope and respond 200 with `{}`. A memo that doesn't exist or is outside the scope responds 404.

#### Scenario: Delete
- **WHEN** MoeMemos deletes a memo
- **THEN** the note is deleted and no longer shown on `/notes`

### Requirement: Trailing attachments are the memo's attachments
The attachment blocks at the end of a note's content, after its last block of another type, SHALL be the note's trailing attachments. A note whose content is only attachment blocks has all of them as trailing attachments. A memo SHALL list its note's trailing attachments in `attachments`, in their order and each once, and its `content` SHALL leave them out. Attachment blocks with other blocks after them stay in `content` as Markdown (see "Content as Markdown" in `note-document`).

When a memo is written with content and attachments (see "Create memos" and "Update memos"), the note's content SHALL be the blocks read from the content followed by an attachment block for each listed attachment, in the order given and each once. Attachment blocks at the end of the written content therefore become trailing attachments as well.

#### Scenario: Photo at the end
- **WHEN** a note "Garden" has the paragraph "roses" followed by the attachment blocks of `roses.jpg` and `plan.pdf`
- **THEN** its memo has the content `# Garden\n\nroses` and the attachments `roses.jpg` and `plan.pdf`, in this order

#### Scenario: Attachment in the middle
- **WHEN** a note has the attachment block of `roses.jpg`, then the paragraph "roses"
- **THEN** its memo's content has `![roses.jpg](<BOARDLY_BASE_URL>/api/attachments/<id>/roses.jpg)` followed by `roses`, and `attachments` is empty

#### Scenario: Only attachments
- **WHEN** a note without a title has only the attachment block of `roses.jpg`
- **THEN** its memo has the content `` and the attachment `roses.jpg`

#### Scenario: Same file twice at the end
- **WHEN** a note ends with the attachment blocks of `roses.jpg`, `roses.jpg`
- **THEN** its memo lists `roses.jpg` once

### Requirement: Memo attachments
A memo attachment SHALL be returned as:

- `name`: `attachments/<attachment id>`,
- `createTime`: the attachment's upload time,
- `filename`: its name, `type`: its content type, `size`: its size in bytes as a decimal string,
- `externalLink`: `<BOARDLY_BASE_URL>/api/attachments/<id>/<name>` with the name escaped for a URL path, so MoeMemos shows and downloads the file from Boardly,
- `memo`: `memos/<note id>` in a memo's `attachments`; left out elsewhere.

A request SHALL refer to an attachment by its `name`; other fields of a referenced attachment are ignored. A name that isn't `attachments/<id>` of an existing attachment of the token's owner responds 400 with code `3`. Attachments marked for deletion exist and are accepted.

#### Scenario: Shown in MoeMemos
- **WHEN** MoeMemos lists a memo with the attachment `roses.jpg`
- **THEN** the attachment has `externalLink` set to the attachment's Boardly URL, and MoeMemos loads the image from there

#### Scenario: Someone else's attachment
- **WHEN** a memo is created with an attachment name of one of Bob's attachments, using Ann's token
- **THEN** the server responds 400 with code `3`, and no note is created

### Requirement: Upload attachments
`POST /api/v1/attachments` with `filename`, `content` (the file as base64) and optional `type` and `memo` SHALL store the file as an attachment of the token's owner and respond 200 with it (see "Memo attachments"). The content type SHALL be found as for uploads in the web app (see "Uploading attachments" in `note-attachments`); `type` is ignored. `memo` is ignored; the file is attached by the memo update that MoeMemos sends next. The new attachment is not in any note, so it is marked for deletion like any unused upload when no memo refers to it within a day.

The server SHALL respond 400 with code `3` and nothing stored when `filename` is missing or not a valid attachment name, `content` is missing, empty or not base64, the file's type is unknown or not allowed (with a message saying that this type of file can't be attached), or the file is larger than `BOARDLY_ATTACHMENT_MAX_SIZE_MB` (with a message naming the limit). The server SHALL stop reading a body that is larger than the base64 of the largest allowed file plus 64 KB, and respond 400 with code `3`.

#### Scenario: Upload a photo
- **WHEN** MoeMemos uploads `IMG_0042.jpg`, a JPEG of 2 MB
- **THEN** the server responds 200 with an attachment named `attachments/<id>` with `filename` `IMG_0042.jpg`, `type` `image/jpeg` and `size` `"2097152"`, and the file appears on the user's attachments page

#### Scenario: Too large
- **WHEN** the maximum is 5 MB and MoeMemos uploads a 12 MB video
- **THEN** the server responds 400 with code `3` and a message that attachments can be at most 5 MB, and stores nothing

#### Scenario: Type not allowed
- **WHEN** the allowed types are the defaults and MoeMemos uploads an SVG file
- **THEN** the server responds 400 with code `3`, and stores nothing

### Requirement: List attachments
`GET /api/v1/attachments` SHALL respond 200 with `{"attachments", "nextPageToken": ""}`: every attachment that a note in the token's scope refers to, anywhere in its content, each once, newest first, then by ID. Attachments of notes outside the scope and attachments that no note refers to SHALL NOT be listed.

#### Scenario: Scoped list
- **WHEN** a token has the root category "Home", "Home/Garden" contains `roses.jpg`, and "Work" contains `budget.pdf`
- **THEN** the list has `roses.jpg` and not `budget.pdf`

### Requirement: Delete attachments
`DELETE /api/v1/attachments/{id}` SHALL respond 200 with `{}` without deleting the file when `{id}` is an attachment of the token's owner that a note in the token's scope refers to or that no note refers to. Otherwise it SHALL respond 404 with code `5`. A file that no note refers to any more is marked and deleted by the cleanup (see "Marking and deleting unused attachments" in `note-attachments`).

#### Scenario: Remove a photo in MoeMemos
- **WHEN** Ann removes `roses.jpg` from a memo in MoeMemos, which updates the memo and then deletes the attachment
- **THEN** both requests succeed, the note no longer contains `roses.jpg`, and the file still exists until the cleanup deletes it

#### Scenario: Used in another note
- **WHEN** `roses.jpg` is in a memo and also pasted into the note "Ideas", and MoeMemos deletes it
- **THEN** the server responds 200, and "Ideas" still shows the image

#### Scenario: Attachment outside the scope
- **WHEN** a token with the root category "Home" deletes `budget.pdf`, which only "Work" refers to
- **THEN** the server responds 404
