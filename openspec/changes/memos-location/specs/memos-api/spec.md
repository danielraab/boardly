# Spec Delta

## ADDED Requirements

### Requirement: Trailing location is the memo's location
A note's trailing location SHALL be a location block (see "Document format" in `note-document`) that is the last block of the content before its trailing attachments (see "Trailing attachments are the memo's attachments"), or the last block when there are none. A note has at most one trailing location. A memo SHALL return its note's trailing location in `location` as `{"placeholder": <label or "">, "latitude": <lat>, "longitude": <lng>}`, and its `content` SHALL leave the block out. A memo of a note without a trailing location has no `location`. Other location blocks stay in `content` as Markdown (see "Content as Markdown" in `note-document`).

When a memo is written with a location (see "Create memos" and "Update memos"), the note's content SHALL be the blocks read from the content, followed by a location block for the location, followed by the trailing attachments. The location block gets `latitude` and `longitude`, rounded to 6 decimals, and the `placeholder` as its label: trimmed, with line breaks replaced by spaces and cut to 200 characters. An empty placeholder gives no label. A `latitude` outside -90 to 90, a `longitude` outside -180 to 180, or a coordinate that is missing or isn't a number SHALL respond 400 with code `3`, and nothing changes.

#### Scenario: Location at the end
- **WHEN** a note "Trip" has the paragraph "Meet here" followed by the location "Stephansplatz" at 48.208176, 16.373819 and the attachment block of `square.jpg`
- **THEN** its memo has the content `# Trip\n\nMeet here`, the location `{"placeholder": "Stephansplatz", "latitude": 48.208176, "longitude": 16.373819}` and the attachment `square.jpg`

#### Scenario: Location in the middle
- **WHEN** a note has the location "Stephansplatz" followed by the paragraph "Meet here"
- **THEN** its memo's content has `[Stephansplatz](geo:48.208176,16.373819)` followed by `Meet here`, and the memo has no `location`

#### Scenario: Location without label
- **WHEN** a note ends with a location without a label at 48.2, 16.37
- **THEN** its memo has the location `{"placeholder": "", "latitude": 48.2, "longitude": 16.37}`

#### Scenario: Two locations at the end
- **WHEN** a note ends with the locations "A" and then "B"
- **THEN** the memo's `location` is "B", and its content ends with `[A](geo:…)`

#### Scenario: Out of range
- **WHEN** MoeMemos creates a memo with the location latitude 95
- **THEN** the server responds 400 with code `3`, and no note is created

## MODIFIED Requirements

### Requirement: Memos are notes in the token's note scope
A memo SHALL be a note in the token's note scope (see "Token subtree scope" in `app-tokens`); its name is `memos/<note id>`. Every endpoint in this spec SHALL see and change only these notes. A memo SHALL be returned as:

- `name`, `creator` (`users/<owner id>`), `createTime` and `displayTime` (the note's creation time), `updateTime` (the note's time of last change),
- `state`: `ARCHIVED` for an archived note, else `NORMAL`; `pinned`: the note's pinned flag,
- `content`: the note's content without its trailing location and trailing attachments as Markdown (see "Content as Markdown" in `note-document`), preceded by `# <title>` and a blank line when the note has a title,
- `attachments`: the note's trailing attachments (see "Trailing attachments are the memo's attachments" and "Memo attachments"),
- `location`: the note's trailing location (see "Trailing location is the memo's location"), left out when it has none,
- `visibility` `PRIVATE`; `tags` and `relations` `[]`.

#### Scenario: Titled note
- **WHEN** a note "Groceries" has the task list `- [ ] milk`
- **THEN** its memo has the content `# Groceries\n\n- [ ] milk`, no attachments and no location

#### Scenario: Untitled note
- **WHEN** a note without a title has the paragraph "call the plumber"
- **THEN** its memo has the content `call the plumber`

#### Scenario: Pinned and archived
- **WHEN** a note is pinned and archived
- **THEN** its memo has `pinned` `true` and `state` `ARCHIVED`

#### Scenario: Note outside the scope
- **WHEN** a token with the root category "Inbox" asks to change a note in "Work"
- **THEN** the server responds 404

### Requirement: Create memos
`POST /api/v1/memos` with `content` and optional `visibility`, `attachments`, `createTime` and `location` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, with the title and content from `content` (see "Title from a leading heading") followed by a location block for `location` when it is given (see "Trailing location is the memo's location") and then the listed attachments (see "Trailing attachments are the memo's attachments"), neither pinned nor archived, and respond 200 with the memo. `createTime` SHALL become the note's creation time, so memos written offline keep their date; a time in the future, or a missing one, means now. `visibility` is ignored, and a `location` of `null` is the same as none. A missing `content`, an unknown attachment (see "Memo attachments"), an invalid location, or a content whose document would be larger than allowed, responds 400, and nothing is created.

#### Scenario: Write a memo
- **WHEN** MoeMemos creates a memo "milk\neggs" with a token whose root category is "Inbox"
- **THEN** a note without a title with that text is created in "Inbox", and `/notes` shows it

#### Scenario: Written offline
- **WHEN** MoeMemos creates a memo with `createTime` of yesterday 08:00 UTC
- **THEN** the note's creation time is yesterday 08:00 UTC

#### Scenario: Memo with a photo
- **WHEN** MoeMemos uploads `roses.jpg` and then creates a memo "Roses are out" with that attachment
- **THEN** a note is created with the paragraph "Roses are out" followed by the attachment block of `roses.jpg`, `/notes/<id>` shows the photo, and the returned memo lists `roses.jpg` in `attachments`

#### Scenario: Memo with a location
- **WHEN** MoeMemos creates a memo "Coffee" with the location `{"placeholder": "Café Central", "latitude": 48.2104, "longitude": 16.3655}` and the attachment `cup.jpg`
- **THEN** the note has the paragraph "Coffee", the location "Café Central" at 48.2104, 16.3655 and the attachment block of `cup.jpg`, in this order, `/notes/<id>` shows the location chip, and the returned memo has that `location`

### Requirement: Update memos
`PATCH /api/v1/memos/{id}` SHALL change the note of a memo in scope and respond 200 with the memo. With `updateMask`, a comma-separated list of field names in snake case, only the listed fields of the body are applied; without it, every field present in the body is applied. The fields are:

- `content`: the title and the content before the trailing location and trailing attachments, as in "Title from a leading heading". The note's trailing location and trailing attachments stay unless `location` or `attachments` is applied too. Content equal to the memo's current content leaves the note's title and content unchanged.
- `attachments`: the note's trailing attachments, replacing the current ones (see "Trailing attachments are the memo's attachments"); an empty list removes them. Removing an attachment from a note doesn't delete the file.
- `location`: the note's trailing location, replacing the current one (see "Trailing location is the memo's location"). `null`, or `location` listed in `updateMask` but missing from the body, removes it. A location equal to the current one changes nothing.
- `pinned`: the note's pinned flag.
- `state`: `ARCHIVED` archives the note, `NORMAL` unarchives it; other values respond 400.
- `visibility`, `update_time`, `create_time` and `display_time` are ignored.

The change SHALL be applied to the note's current version, so a change from MoeMemos always wins, and it counts as a change of the note (version and time of last change) when anything changed. A memo that doesn't exist or is outside the scope responds 404. An unknown attachment (see "Memo attachments") or an invalid location responds 400. A request that fails changes nothing.

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
- **WHEN** MoeMemos sends a memo's current content, attachments and location unchanged
- **THEN** the note's version and time of last change stay the same

#### Scenario: Inline attachment kept
- **WHEN** a note has an image block followed by a paragraph, and MoeMemos sends its content unchanged with an empty attachment list
- **THEN** the image block stays where it is

#### Scenario: Set a location
- **WHEN** a memo "Coffee" has no location, and MoeMemos sends `{"location": {"placeholder": "Café Central", "latitude": 48.2104, "longitude": 16.3655}}` with `updateMask=location`
- **THEN** the note gets the location "Café Central" after its text and before its trailing attachments, and its version goes up by one

#### Scenario: Remove the location
- **WHEN** a memo has a location, and MoeMemos sends `{}` with `updateMask=location`
- **THEN** the note's trailing location block is removed, and its other blocks stay

#### Scenario: Edit text keeps the location
- **WHEN** a memo has the location "Café Central", and MoeMemos sends `{"content": "Coffee with Ann"}` without `location`
- **THEN** the note's text is "Coffee with Ann", and it still has the location "Café Central" before its trailing attachments
