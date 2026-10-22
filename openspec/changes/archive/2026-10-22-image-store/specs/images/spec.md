# Spec Delta

## Purpose

Stores and serves images for favourites and widgets: uploaded images, and favicons that the server fetches and shares between all users.

## ADDED Requirements

### Requirement: Image storage
The server SHALL store each image as a file under `<data dir>/images/` and keep its ID, content type, size, source (`favicon` or `upload`) and, for uploads, the uploading user. Image IDs SHALL be random and unguessable. The data directory is set by `FAMODASH_DATA_DIR` and defaults to `./data`.

#### Scenario: Stored on disk
- **WHEN** a user uploads an image
- **THEN** a file for it exists under `<data dir>/images/`, and it is still served after a restart

### Requirement: Serving images
`GET /api/images/{id}` SHALL respond with the image's bytes and stored content type to anyone, with or without a session. The response SHALL have `X-Content-Type-Options: nosniff`, `Content-Security-Policy: sandbox`. Uploads SHALL be cacheable for a year as immutable, because their content never changes. Favicons SHALL be cacheable for one day, because a refresh replaces their content under the same ID. An unknown ID gets 404 `not_found`.

#### Scenario: Visitor loads an image
- **WHEN** a signed-out visitor requests an existing image's URL
- **THEN** the server responds 200 with the image and its content type

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
Favicons SHALL be stored once per origin (scheme, host and port) and shared by all users. A favicon older than 30 days SHALL be fetched again the next time it is needed, and the old one is served until the new one is stored. After a failed fetch, the origin SHALL NOT be fetched again for 24 hours.

#### Scenario: Two users, same site
- **WHEN** Ann and Bob both add links to `https://github.com/...`
- **THEN** the server fetches GitHub's favicon once, and both links show the same image

#### Scenario: Failed fetch
- **WHEN** a favicon fetch for an origin failed an hour ago and another link to that origin is added
- **THEN** the server doesn't fetch again, and the link has no favicon

### Requirement: Favicon lookup API
`POST /api/favicons` with `{ "url": "<http or https URL>" }` SHALL respond 200 with `{ "id": "<image id>" }` of the favicon of the URL's origin, from the cache or after fetching it. When no favicon can be found, it responds 404 `favicon_not_found`. A URL that isn't an absolute `http` or `https` URL gets 400 `invalid_url`. A blocked address gets 400 `blocked_address`. It requires a session.

#### Scenario: Cached
- **WHEN** Ann asks for the favicon of `https://github.com/settings` and GitHub's favicon is cached
- **THEN** the server responds 200 with its ID without fetching

#### Scenario: Private address
- **WHEN** private addresses are not allowed and Ann asks for the favicon of `http://192.168.1.10:8096`
- **THEN** the server responds 400 with `blocked_address` and makes no request to that address

### Requirement: Private address protection
Unless `FAMODASH_FETCH_ALLOW_PRIVATE` is `true`, the server SHALL NOT connect to loopback, private, link-local, unique-local, carrier-grade NAT, multicast or unspecified addresses, in IPv4 or IPv6, when it fetches favicons. The check SHALL apply to the address actually connected to after DNS resolution, and to every redirect. `FAMODASH_FETCH_ALLOW_PRIVATE` defaults to `false`.

#### Scenario: Name that resolves to a private address
- **WHEN** private addresses are not allowed and a link's host name resolves to `10.0.0.5`
- **THEN** no connection is made and no favicon is stored

#### Scenario: Redirect to a private address
- **WHEN** a public page redirects to `http://127.0.0.1/`
- **THEN** the redirect isn't followed

#### Scenario: Allowed
- **WHEN** `FAMODASH_FETCH_ALLOW_PRIVATE=true` and a link points to `http://192.168.1.10:8096`
- **THEN** the server fetches that site's favicon

### Requirement: Cleanup of unused uploads
An uploaded image that no favourite and no dashboard widget refers to SHALL be deleted, file and record, once it is older than 24 hours. The server checks this on start and then once an hour. Favicons are not deleted by this cleanup.

#### Scenario: Unused upload
- **WHEN** Ann uploaded an image 25 hours ago and never used it
- **THEN** the image is deleted and its URL responds 404

#### Scenario: Recent upload
- **WHEN** Ann uploaded an image 1 hour ago for a draft she hasn't saved yet
- **THEN** the image is kept
