# link-previews Specification

## Purpose
Gives every link a description and a preview image, fetched from the linked page or sent by a connected app, so the Karakeep and Linkwarden clients can show them on their bookmark cards.

## Requirements

### Requirement: Link description
A link SHALL have an optional description of at most 2048 characters after trimming. An empty description SHALL be stored as no description. Folders have no description. The description is set by the page fetch (see "Fetching page metadata") and by connected apps (see the Karakeep and Linkwarden specs). Boardly's own UI and the `/api/favourites` API SHALL NOT show or change it.

#### Scenario: Description from a connected app
- **WHEN** the Linkwarden app saves a link with the description "Notes on Go generics"
- **THEN** the link has that description, and the Linkwarden and Karakeep APIs return it

#### Scenario: Too long
- **WHEN** a connected app sends a description of 3000 characters
- **THEN** the request is rejected and the link is unchanged

### Requirement: Link preview image
A link SHALL have at most one preview image. The preview belongs to that link only: it is never shared with another link or user, also not when they have the same URL. Folders have no preview. Boardly's own UI and the `/api/favourites` API SHALL NOT show or change it, and there is no upload for previews in Boardly's UI.

#### Scenario: Same URL, two users
- **WHEN** Ann and Bob both save `https://example.org/article`
- **THEN** each link gets its own preview image with a different image ID

### Requirement: Fetching page metadata
When a link is created, or its URL is changed, by any client, the server SHALL load the link's page in the background, without delaying the response, and read:

- the description from `<meta property="og:description">`, or else `<meta name="description">`, with whitespace collapsed and cut to 2048 characters;
- the preview image URL from `<meta property="og:image">`, or else `<meta name="twitter:image">`, resolved against the page URL. Only `http` and `https` image URLs are used.

The page load follows the limits of favicon fetching (at most 512 KB of the page are read, at most 5 redirects, 10 seconds) and the private address protection, for the page and for the image. Pending fetches and conversions survive a restart (see "Background jobs survive restarts" in the images spec). A fetch that ran and failed or was blocked leaves the link without a fetched description or preview and is not retried automatically. The result of a fetch SHALL be discarded when the link's URL has changed again or the link was deleted before the fetch finished.

#### Scenario: Open Graph tags
- **WHEN** the user adds a link to a page that declares `og:description` "A tour of Go" and `og:image` `/img/card.png`
- **THEN** after the fetch the link has the description "A tour of Go" and a preview made from `<page origin>/img/card.png`

#### Scenario: Fallback to the meta description
- **WHEN** a page has no `og:description` but `<meta name="description" content="Docs">`
- **THEN** the link gets the description "Docs"

#### Scenario: Page without metadata
- **WHEN** a page declares neither a description nor an image
- **THEN** the link has no description and no preview

#### Scenario: Private address
- **WHEN** private addresses are not allowed and a page's `og:image` points to `http://10.0.0.5/card.png`
- **THEN** no request is made to that address and the link has no preview

#### Scenario: URL changed during the fetch
- **WHEN** a link's URL is changed while the fetch for its old URL is still running
- **THEN** the result for the old URL is not stored, and the fetch for the new URL decides the description and preview

### Requirement: Fetched values only fill empty fields
A fetched description SHALL only be stored when the link has no description at that moment, and a fetched preview only when the link has no preview at that moment. A description or preview that a connected app or an earlier fetch set is never overwritten by a fetch. Changing a link's URL SHALL remove its preview before the new fetch starts, and SHALL keep its description. The server SHALL NOT remove a description or preview in any other case except deleting the link.

#### Scenario: Description written by the user
- **WHEN** a link has the description "My notes" and its URL is changed to a page with `og:description` "Something else"
- **THEN** the link keeps the description "My notes"

#### Scenario: URL changed
- **WHEN** a link has a preview and its URL is changed to another page with an `og:image`
- **THEN** the old preview is removed and the link gets the new page's image as preview

#### Scenario: Screenshot before the fetch finished
- **WHEN** the Linkwarden extension uploads a screenshot for a new link before the page fetch for that link has finished
- **THEN** the link keeps the screenshot as preview, and the fetched image is not stored

### Requirement: Preview from a connected app
A preview image sent by a connected app SHALL replace the link's current preview, whether it was fetched or sent before. It follows the same limits and conversion as a fetched preview.

#### Scenario: Screenshot after the fetch
- **WHEN** the page fetch already stored a preview for a link and the Linkwarden extension then uploads a screenshot for it
- **THEN** the screenshot becomes the link's preview and the fetched image is deleted

### Requirement: Preview image limits and conversion
A preview image SHALL be a PNG, JPEG, WebP or GIF, detected from its content, of at most `BOARDLY_PREVIEW_MAX_SIZE_MB` megabytes (default `10`) and at most `BOARDLY_PREVIEW_MAX_PIXELS` pixels, width times height (default `25000000`), as declared in the image header. Larger images and other types are not used: a fetched one is ignored, and one from a connected app is rejected. Every accepted image SHALL be converted in the background into a JPEG that is at most `BOARDLY_PREVIEW_WIDTH` pixels wide (default `800`), keeping its aspect ratio and never enlarged. The conversion applies the EXIF orientation of JPEG images, uses the first frame of an animated image and puts transparent areas on a white background. The link gets its preview only once the conversion has finished; an image that can't be converted leaves the link's preview unchanged. Preview images are served like uploads (see "Serving images" in the images spec): their content never changes, a new preview gets a new image ID. Invalid values of the three variables SHALL stop the server on start with an error that names the variable; each must be a positive whole number.

#### Scenario: Large photo
- **WHEN** a page's `og:image` is a 3000×2000 JPEG of 700 KB
- **THEN** the link's preview is an 800×533 JPEG

#### Scenario: Small image
- **WHEN** a page's `og:image` is a 600×315 PNG
- **THEN** the link's preview is a 600×315 JPEG

#### Scenario: Transparent logo
- **WHEN** a page's `og:image` is a PNG logo with a transparent background
- **THEN** the preview shows the logo on white

#### Scenario: Too large
- **WHEN** a page's `og:image` is a 12 MB PNG and `BOARDLY_PREVIEW_MAX_SIZE_MB` is not set
- **THEN** the link has no preview

#### Scenario: Too many pixels
- **WHEN** an image declares 30000×30000 pixels
- **THEN** it is not decoded and the link's preview is unchanged

#### Scenario: SVG
- **WHEN** a page's `og:image` is an SVG
- **THEN** the link has no preview

#### Scenario: Invalid configuration
- **WHEN** the server starts with `BOARDLY_PREVIEW_WIDTH=wide`
- **THEN** it exits with an error naming `BOARDLY_PREVIEW_WIDTH`

### Requirement: Previews are deleted with their link
When a link's preview is replaced or removed, or the link is deleted (also as part of a folder), its preview image SHALL be deleted, file and record, by the image cleanup (see "Cleanup of unused uploads" in the images spec), so at the latest about two hours later. Its URL responds 404 afterwards.

#### Scenario: Link deleted
- **WHEN** Ann deletes a link that has a preview
- **THEN** the preview's image URL responds 404 at the latest two hours later

#### Scenario: Preview replaced
- **WHEN** a link's fetched preview is replaced by a screenshot from the Linkwarden extension
- **THEN** the old preview's image URL responds 404 at the latest two hours later, and the new one is served
