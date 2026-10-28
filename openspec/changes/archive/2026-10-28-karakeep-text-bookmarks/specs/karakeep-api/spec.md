# Spec Delta

## ADDED Requirements

### Requirement: Notes are text bookmarks
A note in the token's note scope (see "Token subtree scope" in the app-tokens spec) SHALL also be a Karakeep bookmark; its ID is the note's ID. It SHALL be returned in Karakeep's bookmark shape with:

- `id`, `userId` (the token owner), `title` (the note's title or `null`), `createdAt` and `modifiedAt` (the note's creation and last-change time, as dates),
- `tags`: an empty array,
- `content`: `{"type": "text", "text", "sourceUrl": null}`, where `text` is the note's content as Markdown (see "Content as Markdown" in the note-document spec),
- `archived` and `favourited` `false`; `note`, `summary`, `source`, `taggingStatus`, `summarizationStatus` and `embeddingStatus` `null`; `assets` an empty array.

`bookmarks.getBookmark` SHALL return a note in scope like a link; a note outside the scope fails with `NOT_FOUND`.

#### Scenario: Note as a bookmark
- **WHEN** a token with the note scope `all` asks for a note "Groceries" with the task list `- [ ] milk`
- **THEN** `bookmarks.getBookmark` returns a bookmark with title "Groceries", `content.type` `text`, `content.text` `- [ ] milk` and no tags

#### Scenario: Note outside the scope
- **WHEN** a token with the root category "Inbox" asks for a note in "Work"
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Linkwarden token has no notes
- **WHEN** a Linkwarden token calls the Linkwarden API
- **THEN** no note is returned or changed

## MODIFIED Requirements

### Requirement: List bookmarks
`bookmarks.getBookmarks` SHALL return the bookmarks of the subtree, links and notes in the token's note scope together, as `{"bookmarks", "nextCursor"}`, sorted by creation time, newest first (`sortOrder` `desc`, the default) or oldest first (`asc`), and by ID for equal times. With `listId`, it SHALL return only the links directly in that list's folder; with `tagId`, only links with that tag. With `archived: true` or `favourited: true` it SHALL return no bookmarks, because nothing in Boardly is archived or favourited. `limit` is 1 to 100 and defaults to 20. When more bookmarks follow, `nextCursor` is `{"createdAt", "id"}` of the last returned bookmark, and passing it as `cursor` returns the next page; otherwise it is `null`. An unknown or out-of-scope `listId` or `tagId` fails with `NOT_FOUND`.

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
- **WHEN** the app calls `bookmarks.getBookmarks` with `archived: true`
- **THEN** the result has no bookmarks and `nextCursor` `null`

### Requirement: Search bookmarks
`bookmarks.searchBookmarks` with `text` SHALL return the bookmarks of the subtree and the notes in scope that match every whitespace-separated term of the text, ignoring case: a term starting with `#` matches a link with a tag of that name and never a note, and any other term matches when the link's title, URL or one of its tags contains it, or when the note's title or its content as Markdown contains it. Karakeep's other query qualifiers are not supported and are searched as plain text. An empty text matches every bookmark. Results are sorted newest first for `sortOrder` `relevance` (the default) and `desc`, and oldest first for `asc`. `limit` is 1 to 100 and defaults to 20. The result is `{"bookmarks", "nextCursor"}` with `nextCursor` `{"ver": 1, "offset"}` when more results follow, else `null`.

#### Scenario: Search by words
- **WHEN** the subtree has links titled "Go tour" and "Rust book" and the app searches for "go"
- **THEN** only "Go tour" is returned

#### Scenario: Search by tag
- **WHEN** the app searches for "#dev"
- **THEN** only links with the tag "dev" are returned

#### Scenario: Search in note text
- **WHEN** a note in scope has the text "call the plumber" and the app searches for "plumber"
- **THEN** that note is returned

#### Scenario: Tag line in a note
- **WHEN** a note's text contains "#dev" and the app searches for "#dev"
- **THEN** the note is not returned

### Requirement: Create bookmarks
`bookmarks.createBookmark` with `type` `link`, `url` and an optional `title` SHALL create a link as the first node of the token's root folder and return it as a bookmark with `alreadyExists: false`. When a bookmark with the same URL already exists in the subtree (compared as in "Check a URL"), it SHALL return that bookmark with `alreadyExists: true` and create nothing. An invalid URL or title fails with `BAD_REQUEST` (the rules of link nodes apply).

`bookmarks.createBookmark` with `type` `text`, `text` and an optional `title` SHALL create a note in the token's root category, or uncategorized for the note scope `all`, with the title and the text read as Markdown (see "Content from Markdown" in the note-document spec), and return it as a bookmark with `alreadyExists: false`. Notes are never treated as duplicates. A title longer than the note title limit, or a text whose document would be larger than allowed, fails with `BAD_REQUEST`. `sourceUrl` is ignored. A token with the note scope `none` fails with `BAD_REQUEST`.

Bookmarks of type `asset` fail with `BAD_REQUEST`, because Boardly doesn't store files. `source`, `crawlPriority`, `createdAt` and `precrawledArchiveId` are ignored; a non-empty `note` or `summary`, or `archived` or `favourited` `true`, fails with `BAD_REQUEST`.

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

#### Scenario: Save a file
- **WHEN** the mobile app creates a bookmark of type `asset`
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

### Requirement: Update bookmarks
`bookmarks.updateBookmark` with `bookmarkId` SHALL change a link's `title` (`null` or empty removes it), `url` and `description` (`null` or empty removes it) when they are given, and return the updated bookmark. The rules of link nodes and link descriptions apply, and violations fail with `BAD_REQUEST`. Changing the URL removes the preview and fetches the new page's metadata, as for any URL change.

For a note, it SHALL change the `title` (`null` or empty removes it) and the `text` when they are given, and return the updated note as a bookmark. The text replaces the content as described in "Content replaced from Markdown" in the note-document spec, so tasks keep their IDs. A `text` equal to the note's current content as Markdown leaves the content unchanged. Every change counts as a change of the note (version and time of last change). `url` or `description` on a note fail with `BAD_REQUEST`, and `text` on a link fails with `BAD_REQUEST`.

Fields that Boardly doesn't have SHALL fail with `BAD_REQUEST` when they would set a value: `archived` or `favourited` `true`, a non-empty `note`, `summary`, `author` or `publisher`, and any `datePublished`, `dateModified`, `createdAt` or `assetContent`.

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
- **WHEN** the app sets `archived: true`
- **THEN** the call fails with `BAD_REQUEST` and the link is unchanged

#### Scenario: Note
- **WHEN** the app sets the note of a bookmark to "Remember this"
- **THEN** the call fails with `BAD_REQUEST` and the link is unchanged

#### Scenario: Edit a note's text
- **WHEN** a note has the tasks "milk" (checked) and "eggs", and the app sets its text to `- [x] milk\n- [ ] eggs\n- [ ] bread`
- **THEN** "milk" and "eggs" keep their task IDs, "bread" gets a new ID, and the note's version goes up by one

#### Scenario: URL on a note
- **WHEN** the app sets the `url` of a text bookmark
- **THEN** the call fails with `BAD_REQUEST` and the note is unchanged

### Requirement: Delete bookmarks
`bookmarks.deleteBookmark` with `bookmarkId` SHALL delete the link, or the note when the ID is a note in scope. Tags that are no longer on any link are removed, as for any delete.

#### Scenario: Delete
- **WHEN** the extension deletes a bookmark it just saved
- **THEN** the link is no longer in the tree

#### Scenario: Delete a note
- **WHEN** the mobile app deletes a text bookmark
- **THEN** the note is deleted and no longer shown on `/notes`

### Requirement: Tag bookmarks
`bookmarks.updateTags` with `bookmarkId`, `attach` and `detach` SHALL add and remove tags on the link and return `{"attached", "detached"}` with the tag IDs. A tag to attach is given by `tagId` (a tag visible to the token) or `tagName` (an existing tag of the user, matched regardless of case, or a new one); a tag to detach is given by `tagId` or `tagName`. The rules of tags on links apply. An unknown or out-of-scope `tagId` fails with `NOT_FOUND`. For a note, a call with empty `attach` and `detach` returns empty `attached` and `detached`, and any tag to attach or detach fails with `BAD_REQUEST` with a message that notes have no tags.

#### Scenario: Add a new tag
- **WHEN** the extension attaches the tag name "later" to a bookmark and the user has no such tag
- **THEN** the tag "later" is created, the link has it and `attached` contains its ID

#### Scenario: Remove a tag
- **WHEN** the extension detaches a tag by ID
- **THEN** the link no longer has it and `detached` contains its ID

#### Scenario: Tag a note
- **WHEN** the app attaches the tag name "later" to a text bookmark
- **THEN** the call fails with `BAD_REQUEST` and no tag is created

### Requirement: Lists are folders in the token's subtree
A Karakeep list SHALL be a folder below the token's root folder, at any depth; the root folder itself is not a list. A list's ID is the folder's ID, and its parent is the parent folder, or `null` when the parent is the token's root folder. A bookmark is in a list when its link is directly in that folder, so a bookmark is in at most one list. Notes are never in a list, and categories are not lists. A list SHALL be returned as `{"id", "name", "description": null, "icon": "📁", "parentId", "type": "manual", "query": null, "public": false, "hasCollaborators": false, "userRole": "owner"}`. The procedures are:

- `lists.list`: `{"lists"}` with every list of the subtree.
- `lists.get` with `listId`: the list.
- `lists.stats`: `{"stats"}`, a map from each list's ID to the number of links directly in it.
- `lists.getListsOfBookmark` with `bookmarkId`: `{"lists"}` with the bookmark's list, or empty when the link is directly in the token's root folder or the bookmark is a note in scope.
- `lists.getPendingInvitations`: always an empty array, because Boardly has no list sharing.

An unknown or out-of-scope `listId` or `bookmarkId` fails with `NOT_FOUND`.

#### Scenario: Nested folders
- **WHEN** a token is rooted at "Reading", which contains the folder "Later" with the subfolder "Videos"
- **THEN** `lists.list` returns "Later" with `parentId` `null` and "Videos" with `parentId` the ID of "Later"

#### Scenario: Count of a list
- **WHEN** "Later" contains 3 links and the folder "Videos" with 2 links
- **THEN** `lists.stats` maps "Later" to 3 and "Videos" to 2

#### Scenario: Lists of a note
- **WHEN** the app calls `lists.getListsOfBookmark` with the ID of a note in scope
- **THEN** the result has no lists

### Requirement: Add bookmarks to lists
`lists.addToList` with `listId` and `bookmarkId` SHALL move the link into the list's folder, as its first node. Since a bookmark is in at most one list, this removes it from its previous list. `lists.removeFromList` SHALL move a link that is directly in the list's folder to the first position of the token's root folder; for a link that isn't in that list, it fails with `BAD_REQUEST`. Both fail with `BAD_REQUEST` for a note in scope, with a message that notes can't be in lists.

#### Scenario: Add to a list
- **WHEN** the extension adds a bookmark in "Later" to the list "Recipes"
- **THEN** the link is the first node of "Recipes" and no longer in "Later"

#### Scenario: Remove from a list
- **WHEN** the app removes a bookmark from the list "Recipes"
- **THEN** the link is the first node of the token's root folder

#### Scenario: Add a note to a list
- **WHEN** the app adds a text bookmark to the list "Recipes"
- **THEN** the call fails with `BAD_REQUEST` and the note is unchanged

### Requirement: User settings and statistics
`users.settings` SHALL return fixed settings: `bookmarkClickAction` `open_original_link`, `archiveDisplayBehaviour` `show`, `timezone` `UTC`, backups off, no reader, AI or curated-tag preferences (`null`) and the tag style `lowercase-hyphens`. `users.stats` SHALL return Karakeep's statistics computed from the subtree and the notes in scope: the number of bookmarks (links of type `link` and notes of type `text`), tags and lists; zero favourites, archived bookmarks, highlights and assets; the 10 most frequent domains (of links); the 10 most used tags; and the bookmarking activity of links and notes of this week, month and year and by hour of day and day of week (in UTC).

#### Scenario: Statistics
- **WHEN** the subtree has 5 links, 3 of them on `example.org`, in 2 folders
- **THEN** `users.stats` reports 5 bookmarks, 2 lists and `example.org` with 3 as the top domain

#### Scenario: Statistics with notes
- **WHEN** the subtree has 5 links and the scope 2 notes
- **THEN** `users.stats` reports 7 bookmarks, and the top domains count only the links
