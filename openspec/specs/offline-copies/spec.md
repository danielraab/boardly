# offline-copies Specification

## Purpose
Keeps an offline copy of every favourite link, a readable version of the page's article and a screenshot of how it looked, so the owner can still read what they saved when the page changes or disappears.

## Requirements

### Requirement: Offline copy of a link
A link SHALL have at most one offline copy. Folders have none. An offline copy has a state and, once a capture has succeeded, its capture time, the page title it found, and up to two parts: a readable article and a screenshot. The state is:

- `pending`, while a capture is queued or running,
- `ready`, when the last capture succeeded,
- `failed`, when the last capture got neither an article nor a screenshot. The copy keeps a reason, which is one of `fetch_failed`, `blocked_address`, `too_large`, `not_html` or `no_content`.

A link that has never had a capture queued has no offline copy. A copy belongs only to its link and its owner. It is never shared with another link or another user, even when they have the same URL.

#### Scenario: Copy with both parts
- **WHEN** Ann saves `https://go.dev/blog/generics` and the screenshot sidecar is configured
- **THEN** after the capture, the link's copy is `ready` with a capture time, the page title, an article and a screenshot

#### Scenario: Folder
- **WHEN** Ann asks for the offline copy of a folder
- **THEN** there is none (see "Offline copy API")

### Requirement: When a copy is captured
The server SHALL queue a capture of a link's offline copy, in the same transaction as the change, when:

- the link is created in Boardly, through the `/api/favourites` API or by a connected app (Karakeep or Linkwarden API), or
- the link's URL is changed.

A URL change SHALL remove the link's current copy, including its article and screenshot, before the new capture is queued. Links created by a bookmark import or a data import SHALL NOT get a capture queued. The capture SHALL run in the background, without delaying the response. When the link's URL changes again, or the link is deleted, before a capture finishes, its result SHALL be discarded. Pending captures of the same link SHALL be combined into one, for the link's latest URL.

#### Scenario: Link created
- **WHEN** Ann adds the link `https://go.dev/tour` on `/favourites`
- **THEN** the response does not wait for the capture, the link's copy is `pending`, and later it is `ready`

#### Scenario: Connected app
- **WHEN** the Karakeep extension saves a link with an app token
- **THEN** a capture is queued for that link

#### Scenario: URL changed
- **WHEN** a link with a `ready` copy has its URL changed
- **THEN** the old article and screenshot are gone, the copy is `pending`, and the new capture is for the new URL

#### Scenario: Bookmark import
- **WHEN** Ann imports a Netscape bookmark file with 500 links
- **THEN** none of the imported links has an offline copy, and no capture is queued for them

#### Scenario: URL changed during a capture
- **WHEN** a link's URL changes while the capture for its old URL is still running
- **THEN** the result for the old URL is not stored, and the copy reflects the capture of the new URL

### Requirement: Readable article
A capture SHALL load the link's page with the limits of page-metadata fetching (at most 5 redirects, 10 seconds, the private address protection). It SHALL read at most `BOARDLY_OFFLINE_MAX_SIZE_MB` megabytes (default `5`). A larger page gives no article, and a response that isn't HTML gives no article. From the page, the server SHALL extract the main content as a readable article and keep only safe content in it:

- headings, paragraphs, lists, quotes, code, tables, links and images stay;
- scripts, styles, frames, embeds, forms, inline event handlers and `javascript:` URLs are removed;
- links and image sources are made absolute against the page URL, and only `http` and `https` ones are kept. Links open in a new tab.

Images in the article SHALL stay links to their original location: the server SHALL NOT download them. An article of fewer than 200 characters of text counts as no article. The page title is taken from the article, or from the page's `<title>`.

#### Scenario: Article with images
- **WHEN** a page has an article with a paragraph and `<img src="/img/diagram.png">`
- **THEN** the stored article has the paragraph and an image with the source `<page origin>/img/diagram.png`, and no request for that image was made by the server

#### Scenario: Scripts removed
- **WHEN** the article contains `<script>` and an element with `onclick`
- **THEN** the stored article contains neither

#### Scenario: Not HTML
- **WHEN** a link points to a PDF
- **THEN** the copy has no article

#### Scenario: Too large
- **WHEN** a page is 8 MB and `BOARDLY_OFFLINE_MAX_SIZE_MB` is not set
- **THEN** the copy has no article

#### Scenario: Too little content
- **WHEN** a page's readable content is a single short sentence
- **THEN** the copy has no article

### Requirement: Screenshot through a browser sidecar
When `BOARDLY_BROWSER_URL` is set to the Chrome DevTools Protocol endpoint of a headless browser (`ws://`, `wss://`, `http://` or `https://`), a capture SHALL also open the link's page in that browser with a 1280×800 viewport and take a screenshot of the viewport. The screenshot is stored as a JPEG, converted like a preview image (see "Preview image limits and conversion" in the link-previews spec) but at its full width of 1280 pixels. The screenshot SHALL take at most 30 seconds. When `BOARDLY_BROWSER_URL` is not set, no browser is used and copies have no screenshot. A browser that can't be reached, or a screenshot that fails or times out, leaves the copy without a screenshot and is logged as a warning. The article is kept. An invalid `BOARDLY_BROWSER_URL` SHALL stop the server on start with an error that names the variable.

#### Scenario: No sidecar
- **WHEN** `BOARDLY_BROWSER_URL` is not set and Ann saves a link
- **THEN** the copy is `ready` with an article and no screenshot

#### Scenario: Sidecar down
- **WHEN** `BOARDLY_BROWSER_URL` points to a browser that isn't running
- **THEN** the copy has an article, no screenshot, and the server logs a warning

#### Scenario: Viewport only
- **WHEN** the page is 6000 pixels high
- **THEN** the screenshot is 1280×800

#### Scenario: Invalid configuration
- **WHEN** the server starts with `BOARDLY_BROWSER_URL=chrome:9222`
- **THEN** it exits with an error naming `BOARDLY_BROWSER_URL`

### Requirement: Browser requests follow the private address protection
Unless `BOARDLY_FETCH_ALLOW_PRIVATE` is `true`, every request the browser makes for a capture SHALL be blocked when its host is or resolves to an address the private address protection forbids. This covers the page, its redirects, subresources and requests made by the page's scripts. Requests with a scheme other than `http`, `https`, `data` and `blob` SHALL be blocked too. A page whose own address is blocked gets no screenshot.

#### Scenario: Subresource on a private address
- **WHEN** a public page contains `<img src="http://10.0.0.5/admin.png">`
- **THEN** the browser makes no request to `10.0.0.5`, and the screenshot is still taken

#### Scenario: Page on a private address
- **WHEN** a link points to `http://192.168.1.10/` and private addresses are not allowed
- **THEN** the copy has no article and no screenshot, and is `failed` with `blocked_address`

### Requirement: Result of a capture
A capture that gets an article or a screenshot, or both, SHALL make the copy `ready`. It replaces the previous article and screenshot of the copy and sets the capture time and the title. A capture that gets neither SHALL make the copy `failed` with its reason. When the copy had parts from an earlier capture, it keeps them and their capture time. A failed capture is not retried automatically.

#### Scenario: Refresh fails
- **WHEN** a link's copy is `ready` from yesterday and a refresh fails because the site is down
- **THEN** the copy is `failed` with `fetch_failed`, and yesterday's article, screenshot and capture time are still served

#### Scenario: Only a screenshot
- **WHEN** a page has too little text for an article but the sidecar takes a screenshot
- **THEN** the copy is `ready` with only a screenshot

### Requirement: Offline copy API
The following routes SHALL require a session of the link's owner. App tokens can't use them, and a link of another user or an unknown ID gets 404 `not_found`, the same as a folder:

- `GET /api/favourites/{id}/offline` responds 200 with `{ "status", "capturedAt", "title", "hasArticle", "hasScreenshot", "error" }`. `status` is `none`, `pending`, `ready` or `failed`. `capturedAt` and `title` are `null` when no capture has succeeded, and `error` is the reason of a failed capture or `null`.
- `GET /api/favourites/{id}/offline/article` responds with the article as `text/html; charset=utf-8`, or 404 `not_found` when the copy has none.
- `GET /api/favourites/{id}/offline/screenshot` responds with the screenshot as `image/jpeg`, or 404 `not_found` when the copy has none.
- `POST /api/favourites/{id}/offline` queues a new capture of the link's current URL and responds 202 with the copy as for `GET`, with `status` `pending`. Parts of an earlier capture stay until the new one succeeds. Each user SHALL be allowed at most 30 such requests per hour, and more get 429 `rate_limited` with `Retry-After`.

Article and screenshot responses SHALL carry `Cache-Control: private, no-cache`.

#### Scenario: Status of a ready copy
- **WHEN** Ann requests `GET /api/favourites/{id}/offline` for a link with a captured copy
- **THEN** the response is 200 with `status` `ready`, the capture time, the title, `hasArticle` `true` and `hasScreenshot` `true`

#### Scenario: Never captured
- **WHEN** Ann requests the status of an imported link
- **THEN** the response is 200 with `status` `none`

#### Scenario: Another user's link
- **WHEN** Bob requests the article of Ann's link
- **THEN** the response is 404 `not_found`

#### Scenario: Signed out
- **WHEN** a visitor without a session requests the screenshot of a link
- **THEN** the response is 401

#### Scenario: Refresh
- **WHEN** Ann posts `POST /api/favourites/{id}/offline` for an imported link
- **THEN** the response is 202 with `status` `pending`, and later the copy is `ready`

#### Scenario: Too many refreshes
- **WHEN** Ann sends a 31st refresh within an hour
- **THEN** the response is 429 `rate_limited` and no capture is queued

### Requirement: Serving the article safely
The article response SHALL carry these headers:

- `Content-Security-Policy: sandbox allow-popups allow-popups-to-escape-sandbox; default-src 'none'; img-src http: https: data:; style-src 'unsafe-inline'; frame-ancestors 'self'`. Scripts stay blocked, and links can still open in a new tab,
- `X-Content-Type-Options: nosniff`,
- `Referrer-Policy: no-referrer`.

The article document SHALL be a complete HTML document with a `<meta name="referrer" content="no-referrer">` and a small built-in reading style. The screenshot response SHALL carry `X-Content-Type-Options: nosniff` and `Content-Security-Policy: sandbox`.

#### Scenario: No scripts run
- **WHEN** the article is opened directly in a browser
- **THEN** no script runs, and it can't read Boardly's cookies or call its API

#### Scenario: No referrer
- **WHEN** the article's images load from the original site
- **THEN** the requests carry no `Referer` header

### Requirement: Offline copies are deleted with their link
When a link is deleted, also as part of a folder, or its URL changes, or a capture replaces the copy's parts, the old article and screenshot files SHALL be deleted by the hourly cleanup, so at the latest about two hours later. A deleted link's copy, including its pending capture, is removed. Deleting an account removes its pending captures.

#### Scenario: Link deleted
- **WHEN** Ann deletes a link with a `ready` copy
- **THEN** its article and screenshot routes respond 404, and their files are gone at the latest two hours later

#### Scenario: Copy replaced
- **WHEN** a refresh of a link's copy succeeds
- **THEN** the old article and screenshot files are gone at the latest two hours later

### Requirement: Cookie banners are left out of screenshots
When the browser takes a screenshot for an offline copy, it SHALL NOT load anything from the hosts of well-known consent-management providers. These requests fail in the browser as blocked requests and are never made by the server. The page of the link itself, including its redirects, is never blocked this way. Before the screenshot, the banners, dialogs and overlays of well-known consent tools SHALL be hidden, and a page they kept from scrolling SHALL be scrollable again, so the screenshot shows the page behind them. Nothing on the page is clicked and no consent is given. The article of the copy is not affected.

#### Scenario: Banner script from a consent provider
- **WHEN** a page loads its cookie banner script from `https://cdn.cookielaw.org/`
- **THEN** no request to `cdn.cookielaw.org` is made, and the screenshot shows the page without the banner

#### Scenario: Banner on the page itself
- **WHEN** a page contains a full-screen cookie dialog with a known banner id, and sets `overflow: hidden` on its `body`
- **THEN** the screenshot shows the page content, not the dialog

#### Scenario: The link is on a provider's host
- **WHEN** a link points to a page on a host on the consent-provider list
- **THEN** the page itself is loaded and gets a screenshot

#### Scenario: Unknown banner
- **WHEN** a page shows a cookie banner that is on neither list
- **THEN** the screenshot shows the banner, and the copy is still `ready`
