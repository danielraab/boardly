# Spec Delta

## MODIFIED Requirements

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
