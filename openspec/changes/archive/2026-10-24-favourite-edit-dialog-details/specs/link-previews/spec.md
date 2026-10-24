# Spec Delta

## MODIFIED Requirements

### Requirement: Link description
A link SHALL have an optional description of at most 2048 characters after trimming. An empty description SHALL be stored as no description. Folders have no description. The description is set by the page fetch (see "Fetching page metadata"), by connected apps (see the Karakeep and Linkwarden specs) and by the user in the link's edit dialog. In the `/api/favourites` API, a link node has `description` set when it has one and leaves it out otherwise, and `PATCH /api/favourites/{id}` accepts `description`, where an empty string clears it. A description longer than the limit gets 400 `invalid_description`, and a description on a folder gets 400 `invalid_field`.

#### Scenario: Description from a connected app
- **WHEN** the Linkwarden app saves a link with the description "Notes on Go generics"
- **THEN** the link has that description, and the Linkwarden, Karakeep and `/api/favourites` APIs return it

#### Scenario: Too long
- **WHEN** a connected app sends a description of 3000 characters
- **THEN** the request is rejected and the link is unchanged

#### Scenario: Description changed in Boardly
- **WHEN** the user sends `PATCH /api/favourites/{id}` with `{ "description": "My notes" }` for a link
- **THEN** the response and the tree show the link with the description "My notes"

#### Scenario: Description cleared
- **WHEN** the user sends `PATCH /api/favourites/{id}` with `{ "description": "" }` for a link with a description
- **THEN** the link has no description and the API leaves `description` out

### Requirement: Link preview image
A link SHALL have at most one preview image. The preview belongs to that link only: it is never shared with another link or user, also not when they have the same URL. Folders have no preview. In the `/api/favourites` API, a link node has `previewImageId` set to its preview's image ID when it has one and leaves it out otherwise. `PATCH /api/favourites/{id}` with `"removePreview": true` SHALL remove the link's preview; on a folder it gets 400 `invalid_field`. There is no upload for previews in Boardly's UI or the `/api/favourites` API.

#### Scenario: Same URL, two users
- **WHEN** Ann and Bob both save `https://example.org/article`
- **THEN** each link gets its own preview image with a different image ID

#### Scenario: Preview in the tree
- **WHEN** a link has a preview and the user loads `GET /api/favourites`
- **THEN** the link node has `previewImageId` with the preview's image ID

#### Scenario: Remove a preview
- **WHEN** the user sends `PATCH /api/favourites/{id}` with `{ "removePreview": true }` for a link with a preview
- **THEN** the link has no preview and the API leaves `previewImageId` out

### Requirement: Fetched values only fill empty fields
A fetched description SHALL only be stored when the link has no description at that moment, and a fetched preview only when the link has no preview at that moment. A description or preview that the user, a connected app or an earlier fetch set is never overwritten by a fetch. Changing a link's URL SHALL remove its preview before the new fetch starts, and SHALL keep its description. The server SHALL NOT remove a description or preview in any other case except the user clearing or removing it and deleting the link. Removing a preview SHALL NOT start a new fetch.

#### Scenario: Description written by the user
- **WHEN** a link has the description "My notes" and its URL is changed to a page with `og:description` "Something else"
- **THEN** the link keeps the description "My notes"

#### Scenario: URL changed
- **WHEN** a link has a preview and its URL is changed to another page with an `og:image`
- **THEN** the old preview is removed and the link gets the new page's image as preview

#### Scenario: Screenshot before the fetch finished
- **WHEN** the Linkwarden extension uploads a screenshot for a new link before the page fetch for that link has finished
- **THEN** the link keeps the screenshot as preview, and the fetched image is not stored

#### Scenario: Removed preview stays removed
- **WHEN** the user removes the preview of a link whose page has an `og:image`
- **THEN** the link has no preview and no fetch is started for it

### Requirement: Previews are deleted with their link
When a link's preview is replaced or removed, or the link is deleted (also as part of a folder), its preview image SHALL be deleted, file and record, by the image cleanup (see "Cleanup of unused uploads" in the images spec), so at the latest about two hours later. Its URL responds 404 afterwards.

#### Scenario: Link deleted
- **WHEN** Ann deletes a link that has a preview
- **THEN** the preview's image URL responds 404 at the latest two hours later

#### Scenario: Preview replaced
- **WHEN** a link's fetched preview is replaced by a screenshot from the Linkwarden extension
- **THEN** the old preview's image URL responds 404 at the latest two hours later, and the new one is served

#### Scenario: Preview removed by the user
- **WHEN** Ann removes a link's preview in its edit dialog
- **THEN** the preview's image URL responds 404 at the latest two hours later
