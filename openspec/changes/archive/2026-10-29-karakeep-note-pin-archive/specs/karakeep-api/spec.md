## MODIFIED Requirements

### Requirement: List bookmarks
`bookmarks.getBookmarks` SHALL return the bookmarks of the subtree, links and notes in the token's note scope together, as `{"bookmarks", "nextCursor"}`, sorted by creation time, newest first (`sortOrder` `desc`, the default) or oldest first (`asc`), and by ID for equal times. With `listId`, it SHALL return only the links directly in that list's folder; with `tagId`, only links with that tag. With `archived: true` it SHALL return only archived notes, and with `archived: false` it SHALL leave archived notes out; with `favourited: true` it SHALL return only pinned notes, and with `favourited: false` it SHALL leave pinned notes out. Links are never archived or favourited. Both filters can be combined, and without them every bookmark is returned. `limit` is 1 to 100 and defaults to 20. When more bookmarks follow, `nextCursor` is `{"createdAt", "id"}` of the last returned bookmark, and passing it as `cursor` returns the next page; otherwise it is `null`. An unknown or out-of-scope `listId` or `tagId` fails with `NOT_FOUND`.

#### Scenario: Pages
- **WHEN** the subtree has 25 links and the app calls `bookmarks.getBookmarks` without a limit, then again with the returned cursor
- **THEN** the first page has the 20 newest links and a cursor, and the second page has the other 5 and `nextCursor` `null`

#### Scenario: Links and notes
- **WHEN** the subtree has a link created yesterday and the scope a note created today
- **THEN** `bookmarks.getBookmarks` returns the note first and the link second

#### Scenario: Bookmarks of a list
- **WHEN** the app calls `bookmarks.getBookmarks` with the ID of the folder "Later" as `listId`
- **THEN** only the links directly in "Later" are returned, not those in its subfolders and no notes

#### Scenario: Archived bookmarks
- **WHEN** the subtree has links, the scope has the archived note "Old plan" and the note "Groceries", and the app calls `bookmarks.getBookmarks` with `archived: true`
- **THEN** the result has only "Old plan" and `nextCursor` `null`

#### Scenario: Not archived
- **WHEN** the app calls `bookmarks.getBookmarks` with `archived: false` for the same subtree and scope
- **THEN** the result has the links and "Groceries", and not "Old plan"

#### Scenario: Favourites
- **WHEN** the scope has the pinned note "Groceries" and the app calls `bookmarks.getBookmarks` with `favourited: true`
- **THEN** the result has only "Groceries"

### Requirement: Create bookmarks
`bookmarks.createBookmark` with `type` `link`, `url` and an optional `title` SHALL create a link as the first node of the token's root folder and return it as a bookmark with `alreadyExists: false`. When a bookmark with the same URL already exists in the subtree (compared as in "Check a URL"), it SHALL return that bookmark with `alreadyExists: true` and create nothing. An invalid URL or title fails with `BAD_REQUEST` (the rules of link nodes apply).

`bookmarks.createBookmark` with `type` `text`, `text` and an optional `title` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, with the title and the text read as Markdown (see "Content from Markdown" in the note-document spec), and return it as a bookmark with `alreadyExists: false`. Notes are never treated as duplicates. A title longer than the note title limit, or a text whose document would be larger than allowed, fails with `BAD_REQUEST`. `sourceUrl` is ignored. A token with the note scope `none` fails with `BAD_REQUEST`.

Bookmarks of type `asset` fail with `BAD_REQUEST`, because Boardly doesn't store files. `source`, `crawlPriority`, `createdAt` and `precrawledArchiveId` are ignored; a non-empty `note` or `summary` fails with `BAD_REQUEST`. For a text bookmark, `archived` sets the new note's archived flag and `favourited` its pinned flag; for a link, `archived` or `favourited` `true` fails with `BAD_REQUEST`.

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
- **WHEN** the mobile app creates a bookmark of type `asset`
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

### Requirement: Update bookmarks
`bookmarks.updateBookmark` with `bookmarkId` SHALL change a link's `title` (`null` or empty removes it), `url` and `description` (`null` or empty removes it) when they are given, and return the updated bookmark. The rules of link nodes and link descriptions apply, and violations fail with `BAD_REQUEST`. Changing the URL removes the preview and fetches the new page's metadata, as for any URL change.

For a note, it SHALL change the `title` (`null` or empty removes it), the `text`, `archived` (the note's archived flag) and `favourited` (the note's pinned flag) when they are given, and return the updated note as a bookmark. The text replaces the content as described in "Content replaced from Markdown" in the note-document spec, so tasks keep their IDs. A `text` equal to the note's current content as Markdown leaves the content unchanged. Every change counts as a change of the note (version and time of last change). `url` or `description` on a note fail with `BAD_REQUEST`, and `text` on a link fails with `BAD_REQUEST`.

Fields that Boardly doesn't have SHALL fail with `BAD_REQUEST` when they would set a value: on a link `archived` or `favourited` `true`, and on any bookmark a non-empty `note`, `summary`, `author` or `publisher`, and any `datePublished`, `dateModified`, `createdAt` or `assetContent`.

#### Scenario: Rename a bookmark
- **WHEN** the app sets the title of a bookmark to "Docs"
- **THEN** the link is shown as "Docs" in Boardly

#### Scenario: Describe a bookmark
- **WHEN** the app sets the description of a bookmark to "Read later"
- **THEN** the link has the description "Read later", and the returned bookmark has it in `content.description`

#### Scenario: Remove the description
- **WHEN** the app sets the description of a bookmark to `null`
- **THEN** the link has no description

#### Scenario: Archive a bookmark
- **WHEN** the app sets `archived: true` on a link
- **THEN** the call fails with `BAD_REQUEST` and the link is unchanged

#### Scenario: Archive a note
- **WHEN** the app sets `archived: true` on a text bookmark with version 3
- **THEN** the note is archived, has version 4, is shown with "Archived" on `/notes`, and the returned bookmark has `archived` `true`

#### Scenario: Unfavourite a note
- **WHEN** the app sets `favourited: false` on a text bookmark of a pinned note
- **THEN** the note is no longer pinned

#### Scenario: Note
- **WHEN** the app sets the note of a bookmark to "Remember this"
- **THEN** the call fails with `BAD_REQUEST` and the link is unchanged

#### Scenario: Edit a note's text
- **WHEN** a note has the tasks "milk" (checked) and "eggs", and the app sets its text to `- [x] milk\n- [ ] eggs\n- [ ] bread`
- **THEN** "milk" and "eggs" keep their task IDs, "bread" gets a new ID, and the note's version goes up by one

#### Scenario: URL on a note
- **WHEN** the app sets the `url` of a text bookmark
- **THEN** the call fails with `BAD_REQUEST` and the note is unchanged

### Requirement: User settings and statistics
`users.settings` SHALL return fixed settings: `bookmarkClickAction` `open_original_link`, `archiveDisplayBehaviour` `show`, `timezone` `UTC`, backups off, no reader, AI or curated-tag preferences (`null`) and the tag style `lowercase-hyphens`. `users.stats` SHALL return Karakeep's statistics computed from the subtree and the notes in scope: the number of bookmarks (links of type `link` and notes of type `text`), tags and lists; the number of pinned notes as favourites and of archived notes as archived bookmarks; zero highlights and assets; the 10 most frequent domains (of links); the 10 most used tags; and the bookmarking activity of links and notes of this week, month and year and by hour of day and day of week (in UTC).

#### Scenario: Statistics
- **WHEN** the subtree has 5 links, 3 of them on `example.org`, in 2 folders
- **THEN** `users.stats` reports 5 bookmarks, 2 lists and `example.org` with 3 as the top domain

#### Scenario: Statistics with notes
- **WHEN** the subtree has 5 links and the scope 2 notes
- **THEN** `users.stats` reports 7 bookmarks, and the top domains count only the links

#### Scenario: Favourites and archived in the statistics
- **WHEN** the scope has 3 notes, one pinned and two archived
- **THEN** `users.stats` reports 1 favourite and 2 archived bookmarks

### Requirement: Notes are text bookmarks
A note in the token's note scope (see "Token subtree scope" in the app-tokens spec) SHALL also be a Karakeep bookmark; its ID is the note's ID. It SHALL be returned in Karakeep's bookmark shape with:

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
