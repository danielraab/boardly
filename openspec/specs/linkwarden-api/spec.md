# linkwarden-api Specification

## Purpose
Defines the Linkwarden REST endpoints that the Linkwarden mobile app and browser extension use to save, find and organise links, their wire format, and how Linkwarden's links, collections, tags, user and dashboard map onto the folder that an app token is bound to.

## Requirements

### Requirement: Linkwarden wire format
The Linkwarden mobile app and browser extension talk to `<server address>/api/v1/…` and `<server address>/api/v2/…`. Boardly SHALL answer them under `/api/linkwarden/api/v1/` and `/api/linkwarden/api/v2/`. Request bodies are JSON. Responses SHALL be JSON with Linkwarden's envelopes: `{"response": <value>}` for single objects, plain lists and errors, and `{"data": <value>, "message": "Success", "success": true}` for search, the tag list and the dashboard. An error SHALL respond with an HTTP status 400, 401, 403 or 404 and `{"response": "<message>"}`, because the clients show `response` as the error text. Times SHALL be ISO 8601 strings in UTC with milliseconds (`2026-03-01T12:00:00.000Z`). Folders, links and tags SHALL appear with their integer ID (see the favourites spec); the token owner SHALL appear as the user `1`, so `ownerId` and `createdById` are always `1`. A path or method under `/api/linkwarden/` that isn't named in this spec SHALL respond 404 with `{"response": "Not found."}`.

#### Scenario: Envelope of a single link
- **WHEN** the app calls `GET /api/linkwarden/api/v1/links/7` for a link in the subtree
- **THEN** the server responds 200 with `{"response": {…}}`, where `response.id` is `7`

#### Scenario: Error text
- **WHEN** a client asks for a link ID that doesn't exist
- **THEN** the server responds 404 with `{"response": "Link not found."}`

#### Scenario: Unknown endpoint
- **WHEN** the app calls `GET /api/linkwarden/api/v1/rss`
- **THEN** the server responds 404 with `{"response": "Not found."}`

### Requirement: Links are links in the token's subtree
A Linkwarden link SHALL be a link node in the subtree of the token's root folder, at any depth. Every endpoint in this spec SHALL see and change only nodes in that subtree (see "Token subtree scope"). A link SHALL be returned as:

- `id`, `name` (the title, or `""`), `type` `"url"`, `description` (the link's description, or `""`), `url`, `collectionId` (the integer ID of the folder the link is directly in), `createdById` `1`, `createdAt` and `updatedAt` (the node's creation and last-change time),
- `tags`: the link's tags as tag objects (see "Tags in the token's subtree", without `_count`),
- `collection`: the link's collection (see "Collections are folders", without `parent`, `members` and `_count`),
- `pinnedBy` `[]`,
- `preview`: a non-empty value when the link has a preview image, or else when Boardly knows a favicon in PNG, JPEG, GIF or WebP for the link's origin, else `"unavailable"`; `image`, `pdf`, `readable` and `monolith` `"unavailable"`,
- `lastPreserved` the creation time, so the clients don't wait for a preservation that never happens,
- `icon`, `iconWeight`, `color`, `textContent`, `metaDescription`, `indexVersion` and `importDate` `null`; `clientSide` and `aiTagged` `false`.

#### Scenario: Link in a subfolder
- **WHEN** a token is rooted at "Reading" and "Reading/Later" contains the link "Go" with tag "dev"
- **THEN** `GET /api/v1/links/{id}` returns the link with `name` "Go", `collectionId` the ID of "Later" and the tag "dev"

#### Scenario: Link outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the server responds 404

#### Scenario: Nothing to preserve
- **WHEN** the app loads a link that was created a second ago
- **THEN** `lastPreserved` equals `createdAt` and every preserved format is `"unavailable"`

#### Scenario: Description
- **WHEN** a link has the description "A tour of Go"
- **THEN** the returned link has `description` "A tour of Go" and `metaDescription` `null`

#### Scenario: Preview without favicon
- **WHEN** a link has a preview image and Boardly knows no favicon for its origin
- **THEN** the returned link has a non-empty `preview`

### Requirement: Search links
`GET /api/v1/search` SHALL return links of the subtree as `{"data": {"links", "nextCursor"}, …}`, 50 per page. The parameters are:

- `sort`: `0` newest first (the default), `1` oldest first, `2` name A–Z, `3` name Z–A. Names are the title, or the URL when there is none, ignoring case. Equal values are ordered by ID.
- `collectionId`: only links directly in that collection's folder. `tagId`: only links with that tag. An unknown or out-of-scope ID returns no links.
- `pinnedOnly=true`: no links, because nothing is pinned in Boardly.
- `searchQueryString`: links that match every whitespace-separated term, ignoring case. A term `url:<value>` matches a link whose URL equals the value, comparing without the fragment and without a trailing slash of the path; `name:<value>` matches a title that contains the value; `tag:<value>` matches a link with a tag of that name; `collection:<value>` matches a link whose collection's name contains the value. A value MAY be in double quotes. Any other term, including Linkwarden's other qualifiers, matches when the link's title, URL or one of its tags contains it.
- `cursor`: the ID of the last link of the previous page; the page starts after that link in the chosen order. `0` or missing starts at the beginning.

`nextCursor` SHALL be the ID of the last returned link when more links follow, else `null`.

#### Scenario: Pages
- **WHEN** the subtree has 60 links and the app calls `/api/v1/search?cursor=0&sort=0`, then again with the returned `nextCursor`
- **THEN** the first page has the 50 newest links, and the second page has the other 10 and `nextCursor` `null`

#### Scenario: Links of a collection
- **WHEN** the app searches with the ID of the folder "Later" as `collectionId`
- **THEN** only the links directly in "Later" are returned, not those in its subfolders

#### Scenario: Already saved check
- **WHEN** the subtree has a link `https://example.org/docs/` and the extension searches `url:https://example.org/docs#intro`
- **THEN** that link is returned

#### Scenario: Words
- **WHEN** the subtree has links titled "Go tour" and "Rust book" and the app searches "go"
- **THEN** only "Go tour" is returned

### Requirement: Create links
`POST /api/v1/links` with `url`, an optional `name`, `description`, `tags` and `collection` SHALL create a link as the first node of the chosen collection's folder and respond 200 with `{"response": <link>}`. The collection is chosen in this order:

1. `collection.id`: that collection, which must be in the subtree, else 400 with "Collection is not accessible.";
2. `collection.name` other than `Unorganized`: the first folder directly in the token's root folder with that name, ignoring case, or a new folder with that name at the end of the token's root folder;
3. otherwise the token's root collection.

Tags are given by `name`, and the rules of tags on links apply (an existing tag in another case is reused). An empty `name` stores no title, and an empty `description` stores no description. The rules of link nodes and link descriptions apply, and violations respond 400. A missing `url` or a `type` other than `url` respond 400. `image` is ignored. A URL that already exists in the subtree is saved again, as in Linkwarden without duplicate prevention.

#### Scenario: Save from the extension
- **WHEN** the extension saves `https://example.org` with name "Example" and the collection `{"name": "Unorganized"}`
- **THEN** a link "Example" is the first node of the token's root folder

#### Scenario: Save with a description
- **WHEN** the extension saves a link with the description "Read on the weekend"
- **THEN** the link has that description, and the page fetch doesn't replace it

#### Scenario: Save into a collection
- **WHEN** the app saves a link with `collection.id` of "Later"
- **THEN** the link is the first node of "Later"

#### Scenario: New collection by name
- **WHEN** the extension saves a link with the collection `{"name": "Recipes"}` and there is no such folder
- **THEN** a folder "Recipes" is created at the end of the token's root folder and contains the link

#### Scenario: PDF upload
- **WHEN** the app creates a link with `type` `pdf`
- **THEN** the server responds 400 and nothing is created

### Requirement: Update links
`PUT /api/v1/links/{id}` SHALL change the link and respond 200 with `{"response": <link>}`. `name` sets the title (`null` or empty removes it), `url` sets the URL when it is a non-empty string, and `description` sets the description when it is given (`null` or empty removes it). `tags` (required) replaces the link's tags with the given names, and tags that are then unused are removed. `collection.id` (required) moves the link to the first position of that collection's folder when it differs from the current one; it must be in the subtree, else 400. Changing the URL removes the preview and fetches the new page's metadata, as for any URL change. The rules of link nodes and link descriptions apply, and violations respond 400. Fields that Boardly doesn't have SHALL respond 400 when they would set a value: a non-empty `icon`, `iconWeight` or `color`, and `pinnedBy` with an entry that has an `id`. A rejected request changes nothing.

#### Scenario: Rename and tag
- **WHEN** the app sends the link back with `name` "Docs" and the tags "dev" and "go"
- **THEN** the link is titled "Docs" in Boardly and has exactly the tags "dev" and "go"

#### Scenario: Change the description
- **WHEN** the app sends the link back with `description` "Updated notes"
- **THEN** the link has the description "Updated notes"

#### Scenario: Move to another collection
- **WHEN** the app sets `collection.id` of a link in "Later" to the ID of "Recipes"
- **THEN** the link is the first node of "Recipes" and no longer in "Later"

#### Scenario: Pin
- **WHEN** the app sends `pinnedBy: [{"id": 1}]`
- **THEN** the server responds 400 and the link is unchanged

### Requirement: Delete links
`DELETE /api/v1/links/{id}` SHALL delete the link and respond 200 with `{"response": <the deleted link>}`. Tags that are no longer on any link are removed, as for any delete.

#### Scenario: Delete
- **WHEN** the app deletes a link
- **THEN** the link is no longer in the tree

### Requirement: Collections are folders
A Linkwarden collection SHALL be a folder in the subtree, including the token's root folder, because every Linkwarden link is in a collection. A link is in the collection of the folder it is directly in. A collection SHALL be returned as `{"id", "name", "description": "", "icon": null, "iconWeight": null, "color": "#0ea5e9", "parentId", "isPublic": false, "ownerId": 1, "createdById": 1, "createdAt", "updatedAt", "parent", "members": [], "_count": {"links"}}`. `parentId` and `parent` (`{"id", "name"}`) are the parent folder, or `null` for the token's root collection. `_count.links` counts the links directly in the folder. The name is the folder's name; the user's top-level folder has no name and SHALL be called "Unorganized".

`GET /api/v1/collections` SHALL respond with `{"response": [<collections>]}`: every collection of the subtree, parents before their children and siblings in tree order.

#### Scenario: Nested folders
- **WHEN** a token is rooted at "Reading", which contains the folder "Later" with the subfolder "Videos"
- **THEN** the list has "Reading" with `parentId` `null`, "Later" with the parent "Reading" and "Videos" with the parent "Later"

#### Scenario: Whole tree
- **WHEN** a token is rooted at the top-level folder
- **THEN** the root collection is called "Unorganized"

#### Scenario: Link count
- **WHEN** "Later" contains 3 links and the folder "Videos" with 2 links
- **THEN** "Later" has `_count.links` 3 and "Videos" 2

### Requirement: Create collections
`POST /api/v1/collections` with `name` and an optional `parentId` SHALL create a folder at the end of the parent collection's folder, or of the token's root folder when `parentId` is missing, and respond 200 with `{"response": <collection>}`. The rules of folder nodes apply. An unknown or out-of-scope `parentId` responds 400. `color`, `icon` and `iconWeight` are ignored, and a non-empty `description` responds 400.

#### Scenario: Create a collection
- **WHEN** the app creates the collection "Recipes" without a parent
- **THEN** a folder "Recipes" is the last node of the token's root folder

### Requirement: Delete collections
`DELETE /api/v1/collections/{id}` SHALL delete the folder and move its contents (links and subfolders) to its parent, in its place, so that no link is lost, and respond 200 with `{"response": <the deleted collection>}`. This differs from Linkwarden, which deletes the links too. App tokens rooted at the deleted folder are revoked, as for any folder delete. Deleting the token's root collection responds 403.

#### Scenario: Delete a collection with links
- **WHEN** the app deletes "Later", which contains 3 links and the subfolder "Videos"
- **THEN** "Later" is gone and its 3 links and "Videos" are in its parent folder

#### Scenario: Delete the root collection
- **WHEN** the app deletes the token's root collection
- **THEN** the server responds 403 and nothing is deleted

### Requirement: Tags in the token's subtree
A tag SHALL be visible to a token when it is on at least one link in the subtree. A tag SHALL be returned as `{"id", "name", "ownerId": 1, "archiveAsScreenshot": null, "archiveAsMonolith": null, "archiveAsPDF": null, "archiveAsReadable": null, "archiveAsWaybackMachine": null, "aiTag": null, "aiGenerated": false, "createdAt", "updatedAt", "_count": {"links"}}`, where `updatedAt` is the creation time and `_count.links` only counts links in the subtree.

`GET /api/v1/tags` SHALL respond with `{"data": {"tags", "nextCursor"}, …}`, 50 per page. `search` keeps tags whose name contains it, ignoring case. `sort` is `0` newest first (the default), `1` oldest first, `2` name A–Z, `3` name Z–A, `4` most links first, `5` fewest links first; equal values are ordered by ID. `cursor` and `nextCursor` work as for search. `GET /api/v1/tags/{id}` SHALL respond with `{"response": <tag>}`. An unknown or out-of-scope tag responds 404.

#### Scenario: Tags of the subtree
- **WHEN** the user has the tag "dev" on a link in the subtree and the tag "work" only on links outside it
- **THEN** the tag list has "dev" and not "work"

#### Scenario: Search tags
- **WHEN** the extension asks for `/api/v1/tags?sort=2&cursor=0&search=de`
- **THEN** the result has "dev" and other matching tags by name, and `nextCursor` `null` when there are at most 50

### Requirement: Change tags
`DELETE /api/v1/tags/{id}` SHALL remove the tag from every link in the subtree and respond 200 with `{"response": <tag>}`. Links outside the subtree keep it, and a tag left on no link is removed. `POST /api/v1/tags` SHALL respond 403 with a message that tags are created by adding them to a link, because a Boardly tag only exists while a link has it.

#### Scenario: Delete a tag
- **WHEN** the app deletes "dev", which is on one link inside and one link outside the subtree
- **THEN** the link inside no longer has "dev" and the link outside still has it

#### Scenario: Create a tag without a link
- **WHEN** the app creates the tag "later" from its tag screen
- **THEN** the server responds 403 and no tag is created

### Requirement: Link previews
`GET /api/v1/archives/{id}?format=1&preview=true` (also with `format=0`, and ignoring `updatedAt`) SHALL respond 200 with the link's preview image when it has one, or else with the image of the favicon Boardly knows for the link's origin when that favicon is PNG, JPEG, GIF or WebP, each with its own content type. A link without either, any other `format`, or a request without `preview=true` responds 404.

`POST /api/v1/archives/{id}?format=0` or `format=1` with a multipart form whose `file` field holds an image (the browser extension's "upload screenshot") SHALL store that image as the link's preview (see "Preview from a connected app" in the link-previews spec) and respond 200 with `{"response": <link>}`; the preview appears once its conversion has finished. A file that breaks the limits of preview images or isn't a supported image type responds 400 with a message that says why, and the link is unchanged. Any other `format` (PDF, readable view, monolith) responds 400 with a message that Boardly doesn't store these files.

#### Scenario: Preview image
- **WHEN** the app loads the preview of a link that has a preview image
- **THEN** the response is that JPEG

#### Scenario: Preview from the favicon
- **WHEN** the app loads the preview of a link to `https://github.com/` without a preview image, and Boardly has GitHub's PNG favicon
- **THEN** the response is that PNG

#### Scenario: Readable view
- **WHEN** the app asks for `format=3`
- **THEN** the server responds 404

#### Scenario: Screenshot upload from the extension
- **WHEN** the extension saves a link with "upload screenshot" turned on
- **THEN** the link is created, the archive upload responds 200, and the screenshot becomes the link's preview

#### Scenario: PDF upload
- **WHEN** a client uploads a file with `format=2`
- **THEN** the server responds 400 and the link is unchanged

#### Scenario: Screenshot too large
- **WHEN** the extension uploads a 12 MB screenshot and `BOARDLY_PREVIEW_MAX_SIZE_MB` is not set
- **THEN** the server responds 400 with a message that the image is too large, and the link keeps its preview

### Requirement: User, configuration and dashboard
- `GET /api/v1/users/me` SHALL respond with `{"response": …}`: `id` `1`, `uuid` the user's Boardly ID, `name` (the display name or `null`), `username` (the user slug), `email`, `image` `null`, `locale` `en`, `linksRouteTo` `ORIGINAL` (tapping a link opens the page, because Boardly stores no content), `preventDuplicateLinks` `false`, all `archiveAs…` `false`, `collectionOrder` `[]`, `dashboardSections` (see below), `subscription` `{"active": false, "quantity": 0}`, `parentSubscription` `{"active": false, "user": {}}`, `hasPassword` and `hasOAuthAccount` `false`, and `createdAt` and `updatedAt` of the user.
- `GET /api/v1/config` (public) SHALL respond with `{"response": …}` with every field of Linkwarden's configuration `null`, except `DISABLE_REGISTRATION` `true` and `INSTANCE_VERSION` `v2.16.3`, the Linkwarden version whose API Boardly follows. Billing (`STRIPE_ENABLED` `null`) is off, so the app never asks for a subscription.
- `GET /api/v1/logins` (public) SHALL respond with `{"buttonAuths": []}`, because there is no SSO.
- The dashboard has the fixed sections `STATS` and `RECENT_LINKS`, each `{"id", "userId": 1, "collectionId": null, "type", "order", "createdAt", "updatedAt"}`. `GET /api/v2/dashboard` SHALL respond with `{"data": {"links", "collectionLinks": {}, "numberOfPinnedLinks": 0, "numberOfTags"}, …}`: the 16 newest links of the subtree and the number of tags visible to the token.

#### Scenario: Log in with a token
- **WHEN** the mobile app logs in with an active Linkwarden token and calls `GET /api/v1/users/me`
- **THEN** the server responds 200 with the user, and then the app opens its dashboard, not the subscribe screen

#### Scenario: Version check of the extension
- **WHEN** the extension reads `INSTANCE_VERSION` from `/api/v1/config`
- **THEN** it is `v2.16.3`, so the extension uses paginated tag lists and tag search

#### Scenario: Dashboard
- **WHEN** the subtree has 20 links and 4 tags and the app opens its dashboard
- **THEN** it shows the 16 newest links and the number of tags 4

### Requirement: Unsupported Linkwarden features
Endpoints for Linkwarden features that Boardly doesn't have SHALL NOT be implemented and respond 404 (see "Linkwarden wire format"), among them SSO login (`/api/v1/auth/…`), registration, billing, highlights, preservation and re-preservation, file uploads other than the screenshot upload in "Link previews", RSS, collection sharing and members, public collections, bulk changes, tag merging, user and dashboard settings, and access token management. Password login (`POST /api/v1/session`, public) SHALL respond 401 with `{"response": "Boardly has no passwords. Log in with an access token from Settings → Connected apps."}`, so the clients show why. Messages for unsupported fields name the field and say that Boardly doesn't support it.

#### Scenario: Highlights
- **WHEN** the app opens a link and calls `GET /api/v1/links/{id}/highlights`
- **THEN** the server responds 404, and the link itself still loads

#### Scenario: Password login
- **WHEN** a client calls `POST /api/v1/session` with a username and password
- **THEN** the server responds 401 and the message explains to use an access token
