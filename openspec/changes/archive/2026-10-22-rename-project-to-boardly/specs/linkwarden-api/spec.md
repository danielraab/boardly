## MODIFIED Requirements

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

- `id`, `name` (the title, or `""`), `type` `"url"`, `description` `""`, `url`, `collectionId` (the integer ID of the folder the link is directly in), `createdById` `1`, `createdAt` and `updatedAt` (the node's creation and last-change time),
- `tags`: the link's tags as tag objects (see "Tags in the token's subtree", without `_count`),
- `collection`: the link's collection (see "Collections are folders", without `parent`, `members` and `_count`),
- `pinnedBy` `[]`,
- `preview`: a non-empty value when Boardly knows a favicon in PNG, JPEG, GIF or WebP for the link's origin, else `"unavailable"`; `image`, `pdf`, `readable` and `monolith` `"unavailable"`,
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

### Requirement: Update links
`PUT /api/v1/links/{id}` SHALL change the link and respond 200 with `{"response": <link>}`. `name` sets the title (`null` or empty removes it), and `url` sets the URL when it is a non-empty string. `tags` (required) replaces the link's tags with the given names, and tags that are then unused are removed. `collection.id` (required) moves the link to the first position of that collection's folder when it differs from the current one; it must be in the subtree, else 400. The rules of link nodes apply, and violations respond 400. Fields that Boardly doesn't have SHALL respond 400 when they would set a value: a non-empty `description`, `icon`, `iconWeight` or `color`, and `pinnedBy` with an entry that has an `id`. A rejected request changes nothing.

#### Scenario: Rename and tag
- **WHEN** the app sends the link back with `name` "Docs" and the tags "dev" and "go"
- **THEN** the link is titled "Docs" in Boardly and has exactly the tags "dev" and "go"

#### Scenario: Move to another collection
- **WHEN** the app sets `collection.id` of a link in "Later" to the ID of "Recipes"
- **THEN** the link is the first node of "Recipes" and no longer in "Later"

#### Scenario: Pin
- **WHEN** the app sends `pinnedBy: [{"id": 1}]`
- **THEN** the server responds 400 and the link is unchanged

### Requirement: Change tags
`DELETE /api/v1/tags/{id}` SHALL remove the tag from every link in the subtree and respond 200 with `{"response": <tag>}`. Links outside the subtree keep it, and a tag left on no link is removed. `POST /api/v1/tags` SHALL respond 403 with a message that tags are created by adding them to a link, because a Boardly tag only exists while a link has it.

#### Scenario: Delete a tag
- **WHEN** the app deletes "dev", which is on one link inside and one link outside the subtree
- **THEN** the link inside no longer has "dev" and the link outside still has it

#### Scenario: Create a tag without a link
- **WHEN** the app creates the tag "later" from its tag screen
- **THEN** the server responds 403 and no tag is created

### Requirement: Link previews
`GET /api/v1/archives/{id}?format=1&preview=true` (also with `format=0`, and ignoring `updatedAt`) SHALL respond 200 with the image of the favicon Boardly knows for the link's origin, with its own content type, when that favicon is PNG, JPEG, GIF or WebP. A link without such a favicon, any other `format`, or a request without `preview=true` responds 404. Uploading an archive (`POST /api/v1/archives/{id}`) responds 400 with a message that Boardly doesn't store screenshots or files.

#### Scenario: Preview from the favicon
- **WHEN** the app loads the preview of a link to `https://github.com/` and Boardly has GitHub's PNG favicon
- **THEN** the response is that PNG

#### Scenario: Readable view
- **WHEN** the app asks for `format=3`
- **THEN** the server responds 404

#### Scenario: Screenshot upload from the extension
- **WHEN** the extension saves a link with "upload screenshot" turned on
- **THEN** the link is created and the following archive upload responds 400

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
Endpoints for Linkwarden features that Boardly doesn't have SHALL NOT be implemented and respond 404 (see "Linkwarden wire format"), among them SSO login (`/api/v1/auth/…`), registration, billing, highlights, preservation and re-preservation, uploads, RSS, collection sharing and members, public collections, bulk changes, tag merging, user and dashboard settings, and access token management. Password login (`POST /api/v1/session`, public) SHALL respond 401 with `{"response": "Boardly has no passwords. Log in with an access token from Settings → Connected apps."}`, so the clients show why. Messages for unsupported fields name the field and say that Boardly doesn't support it.

#### Scenario: Highlights
- **WHEN** the app opens a link and calls `GET /api/v1/links/{id}/highlights`
- **THEN** the server responds 404, and the link itself still loads

#### Scenario: Password login
- **WHEN** a client calls `POST /api/v1/session` with a username and password
- **THEN** the server responds 401 and the message explains to use an access token
