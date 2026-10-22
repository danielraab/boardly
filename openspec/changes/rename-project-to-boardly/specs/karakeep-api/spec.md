## MODIFIED Requirements

### Requirement: Bookmarks are links in the token's subtree
A Karakeep bookmark SHALL be a link node in the subtree of the token's root folder, at any depth; its ID is the node's ID. Every procedure in this spec SHALL see and change only nodes in that subtree (see "Token subtree scope"). A bookmark SHALL be returned in Karakeep's bookmark shape with:

- `id`, `userId` (the token owner), `title` (the link's title or `null`), `createdAt` and `modifiedAt` (the node's creation and last-change time, as dates),
- `tags`: the link's tags as `{"id", "name", "attachedBy": "human"}`,
- `content`: `{"type": "link", "url", "title", "favicon", "crawlStatus": "success", "crawledAt"}`, where `favicon` is the URL of the favicon Boardly knows for the link's origin, or `null`, and `crawledAt` is the creation time,
- `archived` and `favourited` `false`; `note`, `summary`, `source`, `taggingStatus`, `summarizationStatus` and `embeddingStatus` `null`; `assets` an empty array.

Because the crawl status is `success` and no status is `pending`, the clients don't poll a bookmark for updates.

#### Scenario: Link in a subfolder
- **WHEN** a token is rooted at "Reading" and "Reading/Later" contains the link "Go" with tag "dev"
- **THEN** `bookmarks.getBookmark` with that link's ID returns a bookmark with title "Go", its URL in `content.url` and the tag "dev"

#### Scenario: Link outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the call fails with `NOT_FOUND`

### Requirement: List bookmarks
`bookmarks.getBookmarks` SHALL return the bookmarks of the subtree as `{"bookmarks", "nextCursor"}`, sorted by creation time, newest first (`sortOrder` `desc`, the default) or oldest first (`asc`), and by ID for equal times. With `listId`, it SHALL return only the links directly in that list's folder; with `tagId`, only links with that tag. With `archived: true` or `favourited: true` it SHALL return no bookmarks, because nothing in Boardly is archived or favourited. `limit` is 1 to 100 and defaults to 20. When more bookmarks follow, `nextCursor` is `{"createdAt", "id"}` of the last returned bookmark, and passing it as `cursor` returns the next page; otherwise it is `null`. An unknown or out-of-scope `listId` or `tagId` fails with `NOT_FOUND`.

#### Scenario: Pages
- **WHEN** the subtree has 25 links and the app calls `bookmarks.getBookmarks` without a limit, then again with the returned cursor
- **THEN** the first page has the 20 newest links and a cursor, and the second page has the other 5 and `nextCursor` `null`

#### Scenario: Bookmarks of a list
- **WHEN** the app calls `bookmarks.getBookmarks` with the ID of the folder "Later" as `listId`
- **THEN** only the links directly in "Later" are returned, not those in its subfolders

#### Scenario: Archived bookmarks
- **WHEN** the app calls `bookmarks.getBookmarks` with `archived: true`
- **THEN** the result has no bookmarks and `nextCursor` `null`

### Requirement: Create bookmarks
`bookmarks.createBookmark` with `type` `link`, `url` and an optional `title` SHALL create a link as the first node of the token's root folder and return it as a bookmark with `alreadyExists: false`. When a bookmark with the same URL already exists in the subtree (compared as in "Check a URL"), it SHALL return that bookmark with `alreadyExists: true` and create nothing. An invalid URL or title fails with `BAD_REQUEST` (the rules of link nodes apply). Bookmarks of type `text` or `asset` fail with `BAD_REQUEST`, because Boardly only stores links. `source`, `crawlPriority`, `createdAt` and `precrawledArchiveId` are ignored; a non-empty `note` or `summary`, or `archived` or `favourited` `true`, fails with `BAD_REQUEST`.

#### Scenario: Save a page from the extension
- **WHEN** the extension saves `https://example.org` with title "Example"
- **THEN** a link "Example" is the first node in the token's root folder, and the result has `alreadyExists: false`

#### Scenario: Save a page twice
- **WHEN** the extension saves a URL that is already a link in the subtree
- **THEN** the existing bookmark is returned with `alreadyExists: true` and no link is added

#### Scenario: Save a note
- **WHEN** the mobile app creates a bookmark of type `text`
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

### Requirement: Update bookmarks
`bookmarks.updateBookmark` with `bookmarkId` SHALL change the link's `title` (`null` or empty removes it) and `url` when they are given, and return the updated bookmark. The rules of link nodes apply, and violations fail with `BAD_REQUEST`. Fields that Boardly doesn't have SHALL fail with `BAD_REQUEST` when they would set a value: `archived` or `favourited` `true`, a non-empty `note`, `summary`, `description`, `author`, `publisher` or `text`, and any `datePublished`, `dateModified`, `createdAt` or `assetContent`.

#### Scenario: Rename a bookmark
- **WHEN** the app sets the title of a bookmark to "Docs"
- **THEN** the link is shown as "Docs" in Boardly

#### Scenario: Archive a bookmark
- **WHEN** the app sets `archived: true`
- **THEN** the call fails with `BAD_REQUEST` and the link is unchanged

### Requirement: Lists are folders in the token's subtree
A Karakeep list SHALL be a folder below the token's root folder, at any depth; the root folder itself is not a list. A list's ID is the folder's ID, and its parent is the parent folder, or `null` when the parent is the token's root folder. A bookmark is in a list when its link is directly in that folder, so a bookmark is in at most one list. A list SHALL be returned as `{"id", "name", "description": null, "icon": "📁", "parentId", "type": "manual", "query": null, "public": false, "hasCollaborators": false, "userRole": "owner"}`. The procedures are:

- `lists.list`: `{"lists"}` with every list of the subtree.
- `lists.get` with `listId`: the list.
- `lists.stats`: `{"stats"}`, a map from each list's ID to the number of links directly in it.
- `lists.getListsOfBookmark` with `bookmarkId`: `{"lists"}` with the bookmark's list, or empty when the link is directly in the token's root folder.
- `lists.getPendingInvitations`: always an empty array, because Boardly has no list sharing.

An unknown or out-of-scope `listId` or `bookmarkId` fails with `NOT_FOUND`.

#### Scenario: Nested folders
- **WHEN** a token is rooted at "Reading", which contains the folder "Later" with the subfolder "Videos"
- **THEN** `lists.list` returns "Later" with `parentId` `null` and "Videos" with `parentId` the ID of "Later"

#### Scenario: Count of a list
- **WHEN** "Later" contains 3 links and the folder "Videos" with 2 links
- **THEN** `lists.stats` maps "Later" to 3 and "Videos" to 2

### Requirement: Change tags
`tags.update` with `tagId` and `name` SHALL rename the tag and return `{"id", "name"}`. Because tags belong to the whole tree, renaming a tag that is also on links outside the subtree SHALL fail with `FORBIDDEN`, and a name that another tag of the user already has (ignoring case) SHALL fail with `CONFLICT`. `tags.delete` with `tagId` SHALL remove the tag from every link in the subtree; links outside the subtree keep it, and a tag left on no link is removed. `tags.create` SHALL fail with `FORBIDDEN` and a message that tags are created by adding them to a bookmark, because a Boardly tag only exists while a link has it.

#### Scenario: Rename a tag used only in the subtree
- **WHEN** the app renames "dev" to "development" and "dev" is only on links in the subtree
- **THEN** every link that had "dev" shows "development"

#### Scenario: Rename a tag also used elsewhere
- **WHEN** the app renames a tag that is also on a link outside the subtree
- **THEN** the call fails with `FORBIDDEN` and the tag is unchanged

#### Scenario: Delete a tag
- **WHEN** the app deletes the tag "dev", which is on one link inside and one link outside the subtree
- **THEN** the link inside no longer has "dev" and the link outside still has it

### Requirement: Unsupported Karakeep features
Procedures for Karakeep features that Boardly doesn't have SHALL NOT be implemented and answer with `NOT_FOUND` (see "Karakeep tRPC transport"), among them highlights, reading progress, AI summaries, list sharing and collaborators, RSS feeds, account deletion, user settings changes and API key management. Error messages for unsupported fields name the field and say that Boardly doesn't support it.

#### Scenario: Highlights
- **WHEN** the mobile app opens a bookmark and calls `highlights.getForBookmark`
- **THEN** the call fails with `NOT_FOUND`, and the bookmark itself still loads
