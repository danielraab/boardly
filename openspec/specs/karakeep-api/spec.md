# karakeep-api Specification

## Purpose
Defines the Karakeep tRPC procedures that the Karakeep browser extension and mobile app use to save, find and organise bookmarks, and how Karakeep's bookmarks, lists and tags map onto the folder that an app token is bound to.

## Requirements

### Requirement: Bookmarks are links in the token's subtree
A Karakeep bookmark SHALL be a link node in the subtree of the token's root folder, at any depth; its ID is the node's ID. Every procedure in this spec SHALL see and change only nodes in that subtree (see "Token subtree scope"). A bookmark SHALL be returned in Karakeep's bookmark shape with:

- `id`, `userId` (the token owner), `title` (the link's title or `null`), `createdAt` and `modifiedAt` (the node's creation and last-change time, as dates),
- `tags`: the link's tags as `{"id", "name", "attachedBy": "human"}`,
- `content`: `{"type": "link", "url", "title", "description", "imageUrl", "favicon", "crawlStatus": "success", "crawledAt"}`, where `description` is the link's description or `null`, `favicon` is the URL of the favicon Boardly knows for the link's origin, or `null`, `imageUrl` is the URL of the link's preview image, or else the favicon's URL, or else `null`, and `crawledAt` is the creation time. Image URLs are absolute URLs of `GET /api/images/{id}`,
- `archived` and `favourited` `false`; `note`, `summary`, `source`, `taggingStatus`, `summarizationStatus` and `embeddingStatus` `null`; `assets` an empty array.

Because the crawl status is `success` and no status is `pending`, the clients don't poll a bookmark for updates.

#### Scenario: Link in a subfolder
- **WHEN** a token is rooted at "Reading" and "Reading/Later" contains the link "Go" with tag "dev"
- **THEN** `bookmarks.getBookmark` with that link's ID returns a bookmark with title "Go", its URL in `content.url` and the tag "dev"

#### Scenario: Link outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Description and preview
- **WHEN** a link has the description "A tour of Go" and a preview image
- **THEN** its bookmark has `content.description` "A tour of Go" and `content.imageUrl` the absolute URL of the preview image

#### Scenario: Favicon as fallback
- **WHEN** a link has no preview image and Boardly knows the favicon of its origin
- **THEN** its bookmark has `content.imageUrl` equal to `content.favicon`

#### Scenario: Neither
- **WHEN** a link has no description, no preview and no known favicon
- **THEN** `content.description`, `content.imageUrl` and `content.favicon` are `null`

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

### Requirement: Search bookmarks
`bookmarks.searchBookmarks` with `text` SHALL return the bookmarks of the subtree that match every whitespace-separated term of the text, ignoring case: a term starting with `#` matches a link with a tag of that name, and any other term matches when the link's title, URL or one of its tags contains it. Karakeep's other query qualifiers are not supported and are searched as plain text. An empty text matches every bookmark. Results are sorted newest first for `sortOrder` `relevance` (the default) and `desc`, and oldest first for `asc`. `limit` is 1 to 100 and defaults to 20. The result is `{"bookmarks", "nextCursor"}` with `nextCursor` `{"ver": 1, "offset"}` when more results follow, else `null`.

#### Scenario: Search by words
- **WHEN** the subtree has links titled "Go tour" and "Rust book" and the app searches for "go"
- **THEN** only "Go tour" is returned

#### Scenario: Search by tag
- **WHEN** the app searches for "#dev"
- **THEN** only links with the tag "dev" are returned

### Requirement: Check a URL
`bookmarks.checkUrl` with `url` SHALL return `{"bookmarkId"}` with the ID of a bookmark in the subtree whose URL equals the given URL, comparing without the fragment and without a trailing slash of the path, or `null` when there is none. The extension uses it for its "already saved" badge.

#### Scenario: Saved page
- **WHEN** the subtree has a link `https://example.org/docs/` and the extension checks `https://example.org/docs#intro`
- **THEN** the result is that link's ID

#### Scenario: Page saved outside the subtree
- **WHEN** the only link with that URL is outside the subtree
- **THEN** the result is `null`

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
`bookmarks.updateBookmark` with `bookmarkId` SHALL change the link's `title` (`null` or empty removes it), `url` and `description` (`null` or empty removes it) when they are given, and return the updated bookmark. The rules of link nodes and link descriptions apply, and violations fail with `BAD_REQUEST`. Changing the URL removes the preview and fetches the new page's metadata, as for any URL change. Fields that Boardly doesn't have SHALL fail with `BAD_REQUEST` when they would set a value: `archived` or `favourited` `true`, a non-empty `note`, `summary`, `author`, `publisher` or `text`, and any `datePublished`, `dateModified`, `createdAt` or `assetContent`.

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

### Requirement: Delete bookmarks
`bookmarks.deleteBookmark` with `bookmarkId` SHALL delete the link. Tags that are no longer on any link are removed, as for any delete.

#### Scenario: Delete
- **WHEN** the extension deletes a bookmark it just saved
- **THEN** the link is no longer in the tree

### Requirement: Tag bookmarks
`bookmarks.updateTags` with `bookmarkId`, `attach` and `detach` SHALL add and remove tags on the link and return `{"attached", "detached"}` with the tag IDs. A tag to attach is given by `tagId` (a tag visible to the token) or `tagName` (an existing tag of the user, matched regardless of case, or a new one); a tag to detach is given by `tagId` or `tagName`. The rules of tags on links apply. An unknown or out-of-scope `tagId` fails with `NOT_FOUND`.

#### Scenario: Add a new tag
- **WHEN** the extension attaches the tag name "later" to a bookmark and the user has no such tag
- **THEN** the tag "later" is created, the link has it and `attached` contains its ID

#### Scenario: Remove a tag
- **WHEN** the extension detaches a tag by ID
- **THEN** the link no longer has it and `detached` contains its ID

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

### Requirement: Create and edit lists
`lists.create` with `name` and an optional `parentId` SHALL create a folder with that name at the end of the parent list's folder, or of the token's root folder when `parentId` is `null` or missing, and return the list. `lists.edit` with `listId` SHALL rename the folder when `name` is given and move it to the end of the new parent when `parentId` is given (`null` meaning the token's root folder), and return the list. The rules of folder nodes apply, and a move into the list itself or one of its sublists fails with `BAD_REQUEST`. `icon` and an empty `description` are ignored; `type` `smart`, a `query`, a non-empty `description` or `public: true` fail with `BAD_REQUEST`.

#### Scenario: Create a list
- **WHEN** the app creates the list "Recipes" without a parent
- **THEN** a folder "Recipes" is the last node of the token's root folder

#### Scenario: Smart list
- **WHEN** the app creates a list of type `smart`
- **THEN** the call fails with `BAD_REQUEST` and nothing is created

### Requirement: Delete lists
`lists.delete` with `listId` SHALL delete the folder and move its contents (links and subfolders) to its parent, in its place, whatever `deleteChildren` says, so that no link is lost, as in Karakeep, where deleting a list keeps its bookmarks. App tokens rooted at the deleted folder are revoked, as for any folder delete.

#### Scenario: Delete a list with bookmarks
- **WHEN** the app deletes the list "Later", which contains 3 links and the subfolder "Videos"
- **THEN** "Later" is gone and its 3 links and "Videos" are in its parent folder

### Requirement: Add bookmarks to lists
`lists.addToList` with `listId` and `bookmarkId` SHALL move the link into the list's folder, as its first node. Since a bookmark is in at most one list, this removes it from its previous list. `lists.removeFromList` SHALL move a link that is directly in the list's folder to the first position of the token's root folder; for a link that isn't in that list, it fails with `BAD_REQUEST`.

#### Scenario: Add to a list
- **WHEN** the extension adds a bookmark in "Later" to the list "Recipes"
- **THEN** the link is the first node of "Recipes" and no longer in "Later"

#### Scenario: Remove from a list
- **WHEN** the app removes a bookmark from the list "Recipes"
- **THEN** the link is the first node of the token's root folder

### Requirement: Tags in the token's subtree
A tag SHALL be visible to a token when it is on at least one link in the subtree. `tags.list` SHALL return `{"tags", "nextCursor"}`, each tag as `{"id", "name", "numBookmarks", "numBookmarksByAttachedType": {"human": n, "ai": 0}}`, where the counts only include links in the subtree. It SHALL filter by `nameContains` (ignoring case) and `ids`; `attachedBy` `ai` or `none` returns no tags. It SHALL sort by number of bookmarks, most first (`sortBy` `usage`, the default, and `relevance`), or by name (`name`), and page with `limit` (default: all tags) and `cursor` `{"page"}`. `tags.get` with `tagId` SHALL return the tag in the same shape. An unknown or out-of-scope `tagId` fails with `NOT_FOUND`.

#### Scenario: Tags of the subtree
- **WHEN** the user has the tag "dev" on a link in the subtree and the tag "work" only on links outside it
- **THEN** `tags.list` returns "dev" and not "work"

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

### Requirement: User settings and statistics
`users.settings` SHALL return fixed settings: `bookmarkClickAction` `open_original_link`, `archiveDisplayBehaviour` `show`, `timezone` `UTC`, backups off, no reader, AI or curated-tag preferences (`null`) and the tag style `lowercase-hyphens`. `users.stats` SHALL return Karakeep's statistics computed from the subtree: the number of bookmarks (all of type `link`), tags and lists; zero favourites, archived bookmarks, highlights and assets; the 10 most frequent domains; the 10 most used tags; and the bookmarking activity of this week, month and year and by hour of day and day of week (in UTC).

#### Scenario: Statistics
- **WHEN** the subtree has 5 links, 3 of them on `example.org`, in 2 folders
- **THEN** `users.stats` reports 5 bookmarks, 2 lists and `example.org` with 3 as the top domain

### Requirement: Unsupported Karakeep features
Procedures for Karakeep features that Boardly doesn't have SHALL NOT be implemented and answer with `NOT_FOUND` (see "Karakeep tRPC transport"), among them highlights, reading progress, AI summaries, list sharing and collaborators, RSS feeds, account deletion, user settings changes and API key management. Error messages for unsupported fields name the field and say that Boardly doesn't support it.

#### Scenario: Highlights
- **WHEN** the mobile app opens a bookmark and calls `highlights.getForBookmark`
- **THEN** the call fails with `NOT_FOUND`, and the bookmark itself still loads
