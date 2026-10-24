# Spec Delta

## MODIFIED Requirements

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

### Requirement: Cleanup of unused uploads
An uploaded image that no favourite and no dashboard widget refers to SHALL be deleted, file and record, once it is older than 24 hours. A preview image that no link refers to SHALL be deleted, file and record, once it is older than one hour. The server checks this on start and then once an hour. Favicons are not deleted by this cleanup.

#### Scenario: Unused upload
- **WHEN** Ann uploaded an image 25 hours ago and never used it
- **THEN** the image is deleted and its URL responds 404

#### Scenario: Recent upload
- **WHEN** Ann uploaded an image 1 hour ago for a draft she hasn't saved yet
- **THEN** the image is kept

#### Scenario: Preview of a deleted folder's link
- **WHEN** Ann deleted a folder with its contents two hours ago, and a link in it had a preview
- **THEN** the preview image is deleted and its URL responds 404

## ADDED Requirements

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
