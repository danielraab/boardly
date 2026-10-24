# Spec Delta

## MODIFIED Requirements

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

### Requirement: Unsupported Linkwarden features
Endpoints for Linkwarden features that Boardly doesn't have SHALL NOT be implemented and respond 404 (see "Linkwarden wire format"), among them SSO login (`/api/v1/auth/…`), registration, billing, highlights, preservation and re-preservation, file uploads other than the screenshot upload in "Link previews", RSS, collection sharing and members, public collections, bulk changes, tag merging, user and dashboard settings, and access token management. Password login (`POST /api/v1/session`, public) SHALL respond 401 with `{"response": "Boardly has no passwords. Log in with an access token from Settings → Connected apps."}`, so the clients show why. Messages for unsupported fields name the field and say that Boardly doesn't support it.

#### Scenario: Highlights
- **WHEN** the app opens a link and calls `GET /api/v1/links/{id}/highlights`
- **THEN** the server responds 404, and the link itself still loads

#### Scenario: Password login
- **WHEN** a client calls `POST /api/v1/session` with a username and password
- **THEN** the server responds 401 and the message explains to use an access token
