# Tasks

## 1. Setup

- [x] 1.1 Add the Go dependencies: go-readability (the maintained one of `go-shiori/go-readability` and `codeberg.org/readeck/go-readability`; note the choice in design.md), `github.com/microcosm-cc/bluemonday` and `github.com/chromedp/chromedp`. Verify with `go build ./...`
- [x] 1.2 Add `BOARDLY_BROWSER_URL` (optional; `ws`, `wss`, `http` or `https` URL, else a start error naming the variable) and `BOARDLY_OFFLINE_MAX_SIZE_MB` (positive whole number, default `5`) to `backend/internal/config`. Verify with config tests for the defaults, valid values and the invalid `chrome:9222` and `0`
- [x] 1.3 Add migration `0025_offline_copies.sql` with the `offline_copies` table from design.md (cascade on `favourite_nodes`). Verify with a db test that deleting a link removes its row
- [x] 1.4 Add `jobs.KindOfflineCapture` and `OfflineCapturePayload{UserID, NodeID, URL, SetPreview}`, combined per key (`replaceOnConflict`). Verify with a jobs test that two enqueues for the same link leave one job with the latest URL

## 2. Queueing captures

- [x] 2.1 Extend `favourites.queueFetches` with an `offline` flag. It enqueues `offline_capture` (`SetPreview: true`) and upserts the `offline_copies` row to `pending` in the same transaction. On a URL change it also clears the row's parts, title and capture time. Pass `true` from `Create` and URL-changing `Update`, and `false` from `import.go`. Verify with favourites tests: create → pending row and job; URL change → parts cleared and new job; title-only update → no job; bookmark import and data import → no row and no job
- [x] 2.2 Verify that Karakeep `createBookmark` and Linkwarden link creation queue a capture, with one test each in their packages
- [x] 2.3 Delete pending `offline_capture` jobs in account deletion (`accounts.go`). Verify with the existing account-deletion test extended by an offline job

## 3. Capture: article

- [x] 3.1 Create `backend/internal/offline` with a `Store` (row read/write with the `url = ?` guard, a blob dir at `<data>/offline`) and the derived state: a `pending` row without a queued or running job reads as `failed`/`fetch_failed`. Verify with store tests for each status, including the dropped-job case
- [x] 3.2 Implement the article fetch with the guarded client (5 redirects, 10 s, `BOARDLY_OFFLINE_MAX_SIZE_MB`, HTML content types only), mapping errors to `fetch_failed`, `blocked_address`, `too_large` and `not_html`. Verify with httptest tests for each reason and for a redirect to a private address
- [x] 3.3 Implement extraction and sanitizing: readability, then the bluemonday policy (absolute `http`/`https` URLs only, `target=_blank rel=noopener noreferrer`, `referrerpolicy=no-referrer` on images), the 200-character minimum, and the title fallback to `<title>`. Wrap the result in the reading template. Verify with tests for relative image URLs made absolute and never requested, and for the removal of `<script>`, `onclick`, `javascript:` links, `<iframe>`, `<form>`, inline SVG with script, `srcdoc` and meta refresh, and for short content giving no article
- [x] 3.4 Run the `offline_capture` worker (2 workers): store the parts, the title and the capture time on success, replacing earlier blobs; set `failed` with the reason when nothing was captured, keeping earlier parts; discard the result when the URL changed or the link is gone. Verify with tests for success, a failed refresh that keeps the old parts, a URL change during the capture, and a deletion during the capture

## 4. Capture: screenshot

- [x] 4.1 Implement the CDP client: a remote allocator from `BOARDLY_BROWSER_URL`, an incognito context per capture, a 1280×800 viewport, a viewport-only JPEG and a 30 s overall timeout, behind a single-slot semaphore. Without the URL, it never connects. Verify with a unit test that no screenshot is attempted when the URL is unset, and with a test where an unreachable URL gives no screenshot and logs a warning while the article is kept
- [x] 4.2 Implement request fulfilment: with `Fetch.enable` on `*`, Boardly performs every paused request through the guarded client (10 s, 10 MB, at most 300 requests, no `Set-Cookie`) and fulfils it, or fails it for blocked addresses, schemes other than `http`/`https`/`data`/`blob`, and errors. Verify with tests of the request handler for a public and a private host, a `file:` URL and the request cap, plus an integration test behind a build tag or env var that runs against a real `chromedp/headless-shell` and checks that a page with `<img src="http://10.0.0.5/x.png">` makes no request to that address
- [x] 4.3 Convert the screenshot with `images.Converter` at width 1280 and store it in the offline blob dir. Verify that a converted 1280×800 image is stored as JPEG
- [x] 4.4 Screenshot as preview: when `SetPreview` is set, the page had no `og:image`/`twitter:image`, and a screenshot exists, write it to the images raw dir and queue `preview_convert{Replace:false}`. Verify with tests: no og image → preview set at `BOARDLY_PREVIEW_WIDTH`; og image → preview unchanged; refresh (`SetPreview:false`) → no preview; link already has a preview → kept

## 5. Cleanup

- [x] 5.1 Hook the offline blob dir into the hourly cleanup with `RemoveOrphans`, where referenced means used by `article_id` or `screenshot_id`. Verify with cleanup tests: files of a replaced copy and of a deleted link are removed after OrphanAge, and current files are kept

## 6. HTTP API

- [x] 6.1 Add `GET /api/favourites/{id}/offline` and `POST /api/favourites/{id}/offline` (202, 30 per hour per user via `ratelimit.Window`, 429 `rate_limited` with `Retry-After`). Both need a session (no app tokens), and other users' links, folders and unknown IDs get 404. Verify with httptest tests for each status, `none` for an imported link, 401 signed out, 401 with an app token, 404 for another user, a folder and an unknown ID, and the 31st POST getting 429
- [x] 6.2 Add `GET …/offline/article` and `GET …/offline/screenshot` with the headers from the spec (CSP with `allow-popups`, `frame-ancestors 'self'`, `nosniff`, `Referrer-Policy`, `Cache-Control: private, no-cache`), and 404 when the part is missing. Verify with tests that check the headers and the content types
- [x] 6.3 Document the four routes and an `OfflineCopy` schema in `backend/internal/httpapi/openapi/boardly.yaml`. Verify that `TestOpenAPIDocumentsMatchRoutes` passes

## 7. Frontend

- [x] 7.1 Add API calls and types for the offline copy to `frontend/src/features/favourites/api.ts`. Verify with `pnpm typecheck` (`make check`)
- [x] 7.2 Add the en/de i18n keys under `favourites.offline` (button, title, states, reasons, tabs, refresh, open original, rate limit, screenshot alt). Verify that the i18n key parity test passes
- [x] 7.3 Add `OfflineCopyDialog.tsx`: a Base UI Dialog shown in place of the details, the states `ready`/`pending`/`failed`/`none`, Article and Screenshot tabs with disabled empty tabs, an iframe with `sandbox="allow-popups allow-popups-to-escape-sandbox"` and `?v=<capturedAt>`, polling every 3 s while pending, Refresh (disabled while pending, rate-limit message) and Open original. Verify with Vitest tests for each state, the tab selection, polling to ready, Refresh and the 429 message, and that the iframe has no `allow-scripts` or `allow-same-origin`
- [x] 7.4 Add the "Offline copy" button between Edit and Open in `FavouriteDetails.tsx`. Closing the offline dialog returns to the details. Verify with `FavouriteDetails.test.tsx` cases for the button order and the return to the details

## 8. Export and import

- [x] 8.1 Add `offlineCopy` to links in `backend/internal/export` (document and ZIP files `offline/<linkId>/article.html` and `screenshot.jpg`, `null` paths without files or for unreadable files; no pending or failed state). Keep `version` `1`. Verify with export tests for with files, without files and no copy, and update `TestEveryUserTableIsCovered` for `offline_copies`
- [x] 8.2 Restore offline copies in `backend/internal/dataimport`: re-sanitize articles, re-convert screenshots (JPEG/PNG/WebP within the preview limits), write a `ready` row with the file's `capturedAt` for links that are not duplicates, queue no capture, and count the files in the report. Verify with import tests for a ZIP with a copy, a JSON export with `offlineCopy`, an old export without the field, an article with `<script>`, a non-image screenshot (skipped and counted) and a duplicate link (existing copy kept)

## 9. Deployment and docs

- [x] 9.1 Add an optional `chrome` service (`chromedp/headless-shell`, pinned tag) to `compose.yaml`, on an `internal: true` network shared with `boardly`, with `BOARDLY_BROWSER_URL` commented in. Verify with `docker compose config`
- [x] 9.2 Document offline copies, `BOARDLY_BROWSER_URL`, `BOARDLY_OFFLINE_MAX_SIZE_MB` and the advice to keep the sidecar on an isolated network in `README.md`, and add a `CHANGELOG.md` entry. Verify by reading it through

## 10. Checks

- [x] 10.1 Manually run the app with the sidecar: save a link, open details → Offline copy, see the article and the screenshot, Refresh, and see that a page without an `og:image` gets the screenshot as preview. Verify by observation (the `run` skill)
- [x] 10.2 Run `make check` and `make test`. Verify that both pass
- [x] 10.3 Run `openspec validate add-offline-copies --strict`. Verify that it passes
