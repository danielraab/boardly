# images Specification

## Purpose
Stores and serves images for favourites and widgets: uploaded images, and favicons that the server fetches and shares between all users.

## Requirements

### Requirement: Image storage
The server SHALL store each image as a file under `<data dir>/images/` and keep its ID, content type, size, source (`favicon`, `upload` or `preview`) and, for uploads and previews, the user they belong to. Image IDs SHALL be random and unguessable. The data directory is set by `BOARDLY_DATA_DIR` and defaults to `./data`.

#### Scenario: Stored on disk
- **WHEN** a user uploads an image
- **THEN** a file for it exists under `<data dir>/images/`, and it is still served after a restart

#### Scenario: Preview stored on disk
- **WHEN** a link gets a preview image
- **THEN** a file for it exists under `<data dir>/images/` with the source `preview` and the link owner as its user

### Requirement: Serving images
`GET /api/images/{id}` SHALL respond with the image's bytes and stored content type to anyone, with or without a session. The response SHALL have `X-Content-Type-Options: nosniff`, `Content-Security-Policy: sandbox`. Uploads and previews SHALL be cacheable for a year as immutable, because their content never changes. Favicons SHALL be cacheable for one day, because a refresh replaces their content under the same ID. An unknown ID gets 404 `not_found`.

#### Scenario: Visitor loads an image
- **WHEN** a signed-out visitor requests an existing image's URL
- **THEN** the server responds 200 with the image and its content type

#### Scenario: Preview
- **WHEN** a client requests the URL of a link's preview image
- **THEN** the server responds 200 with the JPEG and a cache lifetime of a year

#### Scenario: Unknown image
- **WHEN** a client requests `/api/images/doesnotexist`
- **THEN** the server responds 404 with `not_found`

### Requirement: Uploading images
`POST /api/images` SHALL accept one file from a signed-in user, sent as the raw request body with `Content-Type: application/octet-stream`, and respond 201 with `{ "id": "<image id>" }`. The type SHALL be detected from the file's content, not its name. Only PNG, JPEG, WebP, GIF and ICO are accepted, and anything else gets 400 `unsupported_image`. A file larger than 1 MB gets 413 `image_too_large`. A request without a session gets 401.

#### Scenario: Upload a PNG
- **WHEN** Ann uploads a 20 KB PNG file
- **THEN** the server responds 201 with an image ID that serves the same bytes as `image/png`

#### Scenario: SVG
- **WHEN** Ann uploads an SVG file
- **THEN** the server responds 400 with `unsupported_image`

#### Scenario: Too large
- **WHEN** Ann uploads a 2 MB JPEG
- **THEN** the server responds 413 with `image_too_large`

### Requirement: Fetching favicons
The server SHALL find a site's favicon by loading the URL's page and using the best icon declared with `<link rel="icon">`, `<link rel="shortcut icon">` or `<link rel="apple-touch-icon">`. If there is none or it can't be loaded, it SHALL fall back to `/favicon.ico` of the URL's origin. A favicon must pass the same type and size rules as an upload. A page larger than 512 KB is read only up to that size, at most 5 redirects are followed, and the whole fetch has a time limit of 10 seconds.

#### Scenario: Declared icon
- **WHEN** a page declares `<link rel="icon" href="/static/logo.png">`
- **THEN** that PNG becomes the origin's favicon

#### Scenario: Fallback
- **WHEN** a page declares no icon and `/favicon.ico` exists
- **THEN** `/favicon.ico` becomes the origin's favicon

#### Scenario: SVG only
- **WHEN** a site only offers an SVG icon
- **THEN** no favicon is stored for its origin

### Requirement: Shared favicon cache
Favicons SHALL be stored once per origin (scheme, host and port) and shared by all users. A stored favicon SHALL be kept until it is refreshed by hand (see "Manual favicon refresh"); it is not fetched again automatically, however old it is. The server SHALL NOT fetch favicons on start for links that have none. After a failed fetch, the origin SHALL NOT be fetched again automatically for 24 hours.

#### Scenario: Two users, same site
- **WHEN** Ann and Bob both add links to `https://github.com/...`
- **THEN** the server fetches GitHub's favicon once, and both links show the same image

#### Scenario: Failed fetch
- **WHEN** a favicon fetch for an origin failed an hour ago and another link to that origin is added
- **THEN** the server doesn't fetch again, and the link has no favicon

#### Scenario: Old favicon
- **WHEN** GitHub's favicon was stored 60 days ago and Ann adds another link to `https://github.com/`
- **THEN** the server doesn't fetch it again, and the link shows the stored favicon

#### Scenario: Restart
- **WHEN** the server starts and some links have no favicon
- **THEN** it makes no favicon requests for them

### Requirement: Favicon lookup API
`POST /api/favicons` with `{ "url": "<http or https URL>" }` SHALL respond 200 with `{ "id": "<image id>" }` of the favicon of the URL's origin, from the cache or after fetching it. When no favicon can be found, it responds 404 `favicon_not_found`. A URL that isn't an absolute `http` or `https` URL gets 400 `invalid_url`. A blocked address gets 400 `blocked_address`. It requires a session.

#### Scenario: Cached
- **WHEN** Ann asks for the favicon of `https://github.com/settings` and GitHub's favicon is cached
- **THEN** the server responds 200 with its ID without fetching

#### Scenario: Private address
- **WHEN** private addresses are not allowed and Ann asks for the favicon of `http://192.168.1.10:8096`
- **THEN** the server responds 400 with `blocked_address` and makes no request to that address

### Requirement: Private address protection
Unless `BOARDLY_FETCH_ALLOW_PRIVATE` is `true`, the server SHALL NOT connect to loopback, private, link-local, unique-local, carrier-grade NAT, multicast or unspecified addresses, in IPv4 or IPv6, when it fetches favicons, checks a media stream URL or fetches a station logo (see the `media-library` capability). The check SHALL apply to the address actually connected to after DNS resolution, and to every redirect. `BOARDLY_FETCH_ALLOW_PRIVATE` defaults to `false`.

#### Scenario: Name that resolves to a private address
- **WHEN** private addresses are not allowed and a link's host name resolves to `10.0.0.5`
- **THEN** no connection is made and no favicon is stored

#### Scenario: Redirect to a private address
- **WHEN** a public page redirects to `http://127.0.0.1/`
- **THEN** the redirect isn't followed

#### Scenario: Allowed
- **WHEN** `BOARDLY_FETCH_ALLOW_PRIVATE=true` and a link points to `http://192.168.1.10:8096`
- **THEN** the server fetches that site's favicon

#### Scenario: Station URL on a private address
- **WHEN** private addresses are not allowed and a pasted stream URL's host resolves to `10.0.0.5`
- **THEN** no connection is made and the station isn't saved

### Requirement: Cleanup of unused uploads
An uploaded image that no favourite, no dashboard widget, no note, no note category and no media source refers to SHALL be deleted, file and record, once it is older than 24 hours. A preview image that no link refers to SHALL be deleted, file and record, once it is older than one hour. The server checks this on start and then once an hour. Favicons are not deleted by this cleanup.

#### Scenario: Unused upload
- **WHEN** Ann uploaded an image 25 hours ago and never used it
- **THEN** the image is deleted and its URL responds 404

#### Scenario: Recent upload
- **WHEN** Ann uploaded an image 1 hour ago for a draft she hasn't saved yet
- **THEN** the image is kept

#### Scenario: Preview of a deleted folder's link
- **WHEN** Ann deleted a folder with its contents two hours ago, and a link in it had a preview
- **THEN** the preview image is deleted and its URL responds 404

#### Scenario: Note icon
- **WHEN** Ann uploaded an image 25 hours ago and uses it as the icon of a note or a note category
- **THEN** the image is kept

#### Scenario: Station icon
- **WHEN** a station's logo was stored 25 hours ago and the station still uses it as its icon
- **THEN** the image is kept

### Requirement: Background jobs survive restarts
Favicon fetches, preview fetches and preview conversions that the server has started or queued SHALL be stored in the database, and SHALL run, or run again from the start, after the server restarts. None of them is dropped because many are waiting. A job SHALL be queued in the same transaction as the link change that causes it, so a saved link always has its pending jobs. Pending favicon fetches for the same origin SHALL be combined into one, and pending preview fetches for the same link into one for its latest URL. A job that was interrupted by restarts three times SHALL be dropped and logged, so a broken image can't stop the server from starting. A job that ran and failed is not repeated.

#### Scenario: Restart while fetching
- **WHEN** Ann saves a link and the server restarts before its favicon and preview were fetched
- **THEN** after the restart the server fetches them, and the link gets its favicon and preview

#### Scenario: Restart during a conversion
- **WHEN** the Linkwarden extension uploaded a screenshot and the server restarts before it was converted
- **THEN** after the restart the screenshot is converted and becomes the link's preview

#### Scenario: Many links of one site
- **WHEN** a connected app saves 50 links to `https://github.com/` in a row before GitHub's favicon is fetched
- **THEN** the server fetches GitHub's favicon once

#### Scenario: Image that keeps crashing the server
- **WHEN** a conversion was interrupted by a restart three times
- **THEN** it isn't started again, the link's preview is unchanged and the server logs a warning

### Requirement: Manual favicon refresh
`POST /api/favicons/refresh` with `{ "url": "<http or https URL>" }` SHALL fetch the favicon of the URL's origin again right away, also when a fetch failed less than 24 hours ago, and respond 200 with `{ "id": "<image id>" }`. A refresh keeps the favicon's image ID, so every link and widget that uses it shows the new image. When no favicon can be found, it responds 404 `favicon_not_found`, and a favicon stored before stays. A URL that isn't an absolute `http` or `https` URL gets 400 `invalid_url`, and a blocked address gets 400 `blocked_address`. It requires a session. Each user SHALL be allowed at most 30 refreshes per hour; more get 429 `rate_limited`.

#### Scenario: Refresh after a failure
- **WHEN** the favicon fetch for `https://example.org` failed an hour ago, the site now has a favicon, and Ann refreshes it
- **THEN** the server fetches it, responds 200 with its ID, and links to that site show it

#### Scenario: Site lost its favicon
- **WHEN** Ann refreshes the favicon of a site whose icon can no longer be loaded
- **THEN** the server responds 404 with `favicon_not_found` and the stored favicon stays

#### Scenario: Too many refreshes
- **WHEN** Ann has refreshed 30 favicons in the last hour and refreshes another
- **THEN** the server responds 429 with `rate_limited` and makes no request
