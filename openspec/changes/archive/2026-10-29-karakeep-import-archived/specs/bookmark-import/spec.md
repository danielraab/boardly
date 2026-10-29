## MODIFIED Requirements

### Requirement: Karakeep JSON format
A Karakeep JSON export SHALL be read as an object with `bookmarks` and `lists`:

- A list is `{"id", "name", "parentId", "type"}`. A list of `type` `manual` becomes a folder, and its parent is the list with the ID in `parentId`, or the import folder when `parentId` is `null` or unknown. A parent may appear after its child in the file. A list of `type` `smart` is not created and is counted as a smart list. Manual lists also become categories for notes (see "Categories from lists").
- A bookmark is `{"createdAt", "title", "tags", "lists", "content", "note", "summary", "archived"}`. `createdAt` is in Unix seconds, `tags` are tag names, and `lists` are list IDs. A bookmark with `content.type` `link` becomes a link with the URL `content.url`. It goes into the folder of the first ID in `lists` that is a manual list, or into the import folder when there is none.
- A bookmark with `content.type` `text` becomes a note with the Markdown text `content.text` (see "Imported notes"). It goes into the category of the first ID in `lists` that is a manual list, or into the import category when there is none.
- A bookmark of another `content.type`, such as `asset`, is not imported and is counted as "not a link".
- For a link, `note` and, when present, `summary` go into the description (see "Description from notes"), and `archived` is ignored. For a text bookmark, `archived` `true` makes the note archived (see "Imported notes"); a missing `archived` counts as `false`. `favourited`, which Karakeep doesn't export, is ignored for every bookmark.

Unknown fields SHALL be ignored. A file that isn't a JSON object with a `bookmarks` array is not a Karakeep export.

#### Scenario: Nested lists
- **WHEN** the file has the list "Containers" with `parentId` of "Docker", "Docker" with `parentId` of "Dev", and "Dev" with `parentId` `null`, in this order
- **THEN** the folders "Dev/Docker/Containers" are created below the import folder

#### Scenario: Bookmark in two lists
- **WHEN** a bookmark has `lists` with the IDs of "dev" and "Linux", in this order
- **THEN** the link is imported once, into "dev"

#### Scenario: Smart list
- **WHEN** the file has a smart list "Youtube" with the query `url:youtube`
- **THEN** no folder "Youtube" is created, and the report counts one smart list

#### Scenario: Text bookmark
- **WHEN** a bookmark has `content` `{"type": "text", "text": "serienjunkies.team"}` and no lists
- **THEN** a note with the content "serienjunkies.team" is created in the import category, and it is not counted as "not a link"

#### Scenario: Asset bookmark
- **WHEN** a bookmark has `content` `{"type": "asset", "assetType": "image"}`
- **THEN** it is not imported and is counted as "not a link"

#### Scenario: Archived bookmark
- **WHEN** a link bookmark has `archived` `true`
- **THEN** it is imported like any other link

#### Scenario: Archived text bookmark
- **WHEN** a text bookmark has `archived` `true`
- **THEN** its note is imported archived, into the category it would go to anyway

### Requirement: Imported notes
Each imported note SHALL have:
- the bookmark's title as title, trimmed and cut to 200 characters; an empty title becomes no title,
- as content, the Markdown text converted as in "Content from Markdown" in the note-document spec, followed by the bookmark's `note` as a paragraph starting with `Note: ` and its `summary` as a paragraph starting with `Summary: `, each only when not empty, and then a last paragraph with the bookmark's tags, each written as `#` followed by the tag name with whitespace replaced by `-`, separated by spaces, only when there are tags,
- no icon and no colour, not pinned, and archived when the bookmark is archived,
- the bookmark's creation date as creation time and time of last change, or the import time when the bookmark has none or it can't be read,
- version 1.

A note whose content would be larger than the document size limit is not imported and is counted as a note too large to import. Within the import, notes are created in the order of the file.

#### Scenario: Text bookmark with tags
- **WHEN** a text bookmark has the title "Pasta", the text "- [ ] buy basil", the tags "cooking" and "weekend plans", and `createdAt` `1790365701`
- **THEN** the note "Pasta" has a task list with an unchecked "buy basil", then a paragraph "#cooking #weekend-plans", and its creation time and time of last change are 2026-09-25T19:48:21Z

#### Scenario: Karakeep note on a text bookmark
- **WHEN** a text bookmark has the text "Door code 1234" and the note "ask Bob first"
- **THEN** the note's content is a paragraph "Door code 1234" followed by a paragraph "Note: ask Bob first"

#### Scenario: Archived note
- **WHEN** a text bookmark with the text "old plan" has `archived` `true`
- **THEN** the note is archived, isn't pinned, and is shown with "Archived" on `/notes`

#### Scenario: Archived duplicate
- **WHEN** `skipDuplicates` is `true`, the user has a note that isn't archived, and the file has the same note as an archived text bookmark
- **THEN** the bookmark is skipped as a duplicate, and the existing note stays unarchived

#### Scenario: Too large
- **WHEN** a text bookmark has 700 KB of text
- **THEN** no note is created for it, and the report counts one note too large to import

