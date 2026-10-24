# Tasks

## 1. Schema and configuration

- [x] 1.1 Add migration `0014_link_previews.sql`: `description` and `preview_image_id` (references `images`, `ON DELETE SET NULL`) on `favourite_nodes`, and rebuild `images` with the source `preview` and the CHECK `(source IN ('upload', 'preview')) = (user_id IS NOT NULL)`, keeping `favicons.image_id` through a temporary copy; verify with a db test that migrates a database with favicons, uploads and links and checks that every favicon still points to its image and a `preview` row with a user can be inserted
- [x] 1.2 Add migration `0015_jobs.sql` with the `jobs` table and its two partial indexes from design decision 2; verify with a db test that two pending jobs with the same kind and key are rejected, while a running one and a pending one with the same key are allowed
- [x] 1.3 Read `BOARDLY_PREVIEW_MAX_SIZE_MB` (10), `BOARDLY_PREVIEW_MAX_PIXELS` (25000000) and `BOARDLY_PREVIEW_WIDTH` (800) into `Config.Previews`; verify with config tests for the defaults, custom values, and an error naming the variable for `0`, negative and non-numeric values

## 2. Persistent job queue (`internal/jobs`)

- [x] 2.1 Add `Enqueue(ctx, q db.Querier, kind, key, payload)` with the dedup rules (`favicon`: keep the pending job; `preview_fetch`: replace its payload; `preview_convert`: no key) and a non-blocking wake-up signal per kind; verify with tests for both dedup rules, enqueue inside a transaction that rolls back (no job), and the signal
- [x] 2.2 Add the worker pool: `Run(ctx, kind, workers, handler)` that claims jobs in `id` order with one `UPDATE … RETURNING`, runs the handler under `recover`, deletes the job afterwards whether it succeeded or failed, wakes on the signal and polls every 30 seconds; verify with tests that jobs run in FIFO order with one worker, that two workers never get the same job, that a panicking handler doesn't stop the pool and its job is deleted, and that a job enqueued in a transaction committed later still runs
- [x] 2.3 Add `Recover(ctx)` for start: put running jobs back in the queue, and delete and log jobs with 3 or more attempts; verify with tests for an interrupted job that runs again and a job with 3 attempts that is dropped with a warning

## 3. Favourites store

- [x] 3.1 Add `NormalizeDescription` (trim, at most 2048 characters, empty → none) and `Description` on `NewNode` and `Patch` (links only), stored by `Create` and `Update`; verify with favourites tests for create, change, clear, too long, and a description on a folder rejected
- [x] 3.2 Clear `preview_image_id` in `Update` when the URL actually changes, in the same transaction, and keep the description; verify with a favourites test that changing the URL clears the preview, while changing only the title or setting the same URL keeps it
- [x] 3.3 Enqueue a `favicon` job (key: origin) and a `preview_fetch` job (key: node ID, payload: user, node, URL) in `Create` and in `Update` when the URL changes, in the same transaction; verify with favourites tests that a link create queues both, a title-only change queues none, a URL change replaces the pending preview fetch's URL, and a failed create queues nothing
- [x] 3.4 Add `Description` and `PreviewImageID` to `ScopedNode` and select them in `SubtreeNodes`, and leave `Node` (`/api/favourites`) unchanged; verify with a favourites test and an httpapi test that the favourites JSON has no description or preview fields

## 4. Image pipeline (`internal/images`)

- [x] 4.1 Make the size limit of `Sniff` and `ReadLimited` a parameter (uploads and favicons keep 1 MB), and serve `preview` images with the immutable cache header; verify that the existing image tests pass and a new test checks the cache header of a preview
- [x] 4.2 Add `ParseMeta` (first `og:description`, else `name=description`, whitespace collapsed and cut to 2048; first `og:image`, else `twitter:image`, resolved against the page URL, http(s) only); verify with table tests for each tag, order of precedence, relative URLs, `data:` and `javascript:` URLs, and pages without metadata
- [x] 4.3 Add the EXIF orientation reader for JPEG (APP1 tag 0x0112, both byte orders, anything unreadable = 1) and the transform for orientations 2–8; verify with tests on small JPEG fixtures with orientations 1, 3, 6 and 8 and a JPEG without EXIF
- [x] 4.4 Add the converter: `DecodeConfig` with the pixel cap, decode PNG/JPEG/WebP/GIF (first frame), apply orientation, scale to at most the configured width with BiLinear (never enlarge), draw over white, encode JPEG q80; verify with tests for a 3000×2000 JPEG → 800×533, a 600×315 PNG kept at its size, a transparent PNG whose transparent pixels become white, an animated GIF, an image above the pixel cap rejected without decoding, and an SVG and an ICO rejected
- [x] 4.5 Add the `preview_convert` handler (one worker): read the raw file from `<images dir>/raw/`, convert, save as `preview` with the user, apply it with the conditional updates from design decision 4 for `fill` and `replace`, delete the new image when nothing was updated, and remove the raw file; verify with tests for fill into an empty link, fill skipped when a preview exists, replace over an existing preview, any job skipped after the URL changed or the link was deleted, a missing raw file, and the raw file removed in every case
- [x] 4.6 Add the `preview_fetch` handler (two workers): load the page with the guarded client and favicon limits, fill an empty description with the conditional update, download the image within the preview size limit into a raw file and enqueue a `preview_convert` job with mode `fill`; verify with httptest-server tests for the Open Graph scenario, a description that already exists, an image that is too large, a page that redirects to a blocked address, and an `og:image` on a blocked address
- [x] 4.7 Extend `Sweep` to delete `preview` images older than one hour that no link references, and raw files older than one hour that no job names; verify with cleanup tests for an unreferenced old preview (deleted), an unreferenced new preview (kept), a referenced old preview (kept), the preview of a link removed by a folder delete with contents (deleted), and an old raw file with and without a job

## 5. Favicons

- [x] 5.1 Replace the favicon channel and `Favicons.Queue` with a `favicon` job handler (four workers) that calls `Get`, and remove `Queue` from the `Favicons` interfaces in `karakeep` and `linkwarden` and its calls in `httpapi`; verify that the existing favicon, Karakeep and Linkwarden tests pass with the store enqueuing instead
- [x] 5.2 Remove the 30-day refresh from `Favicons.Get` (a stored favicon is always used), and remove `Backfill` and its call in `main.go`; verify with favicon tests that a 60-day-old favicon isn't fetched again and that the 24 h back-off still applies
- [x] 5.3 Add `Favicons.Refresh(ctx, url)`, which ignores freshness and the back-off, shares the `singleflight` key and keeps the old image on failure; verify with tests for a refresh after a recent failure, a failing refresh that keeps the stored image under the same ID, and a blocked address
- [x] 5.4 Add `POST /api/favicons/refresh` (session required, 30 per hour per user, burst 10, the error codes from the spec); verify with httptest tests for 200 with the ID, `favicon_not_found`, `invalid_url`, `blocked_address`, 401 without a session, and 429 `rate_limited` after the limit

## 6. Server wiring

- [x] 6.1 In `main.go`, call `jobs.Recover` after the migrations and before serving, then start the worker pools for `favicon` (4), `preview_fetch` (2) and `preview_convert` (1); verify with a test that starts the server on a database with a pending `favicon` and an interrupted `preview_fetch` job and sees both run

## 7. Karakeep API

- [x] 7.1 Return `content.description` and `content.imageUrl` (absolute preview URL, else the favicon URL, else `null`) on every bookmark; verify with Karakeep tests for a link with description and preview, a link with only a favicon, and a link with neither
- [x] 7.2 Accept `description` in `bookmarks.updateBookmark` (raw JSON; `null` or empty clears it, too long → `BAD_REQUEST`) and keep `note` rejected; verify with Karakeep tests for set, clear, too long, and a note that still fails

## 8. Linkwarden API

- [x] 8.1 Return `description` on links, accept it in `POST /api/v1/links` and `PUT /api/v1/links/{id}` (empty or `null` clears it on update), and set `preview` from the preview image or else the favicon; verify with Linkwarden tests for create with a description, update and clear, too long rejected, and `preview` for a link with a preview but no favicon
- [x] 8.2 Serve the preview image from `GET /api/v1/archives/{id}?preview=true`, falling back to the favicon; verify with httptest tests for a link with a preview (JPEG), a link with only a PNG favicon, and a link with neither (404)
- [x] 8.3 Accept `POST /api/v1/archives/{id}?format=0|1` with a multipart `file`, limited to the preview size, sniffed, written as a raw file and enqueued as a `preview_convert` job with mode `replace`, responding 200 with the link; other formats, too large files and unsupported types respond 400 with a message; verify with httptest tests shaped like the extension's request (PNG screenshot), a 2 MB file with a 1 MB limit, a text file, `format=2`, and a link outside the subtree

## 9. Frontend

- [x] 9.1 Add the "Refresh icon" / "Icon neu laden" action to links on the Favourites page: disabled while running, refreshes the tree on success, and shows translated messages for `favicon_not_found`, `blocked_address` and `rate_limited`; verify with Vitest tests using `mockApi` for success (the request body has the link's URL and the tree is loaded again), each error message, and the disabled state, and that the i18n key check passes for en and de

## 10. Docs and checks

- [x] 10.1 Update the README ("Favourites tree": descriptions and previews for connected apps; "Favicons and images": previews, conversion, persisted background jobs, no automatic favicon refresh, the refresh action, no backfill for existing links; configuration table: the three new variables; Linkwarden section: screenshot upload now works) and add a CHANGELOG entry; verify by reading them against the specs
- [ ] 10.2 Run `make check` and `make test`, then on a local server:
  - save a page with Open Graph tags from the Karakeep extension and see its description and image in the Karakeep app
  - save one from the Linkwarden extension with "upload screenshot" on and see the screenshot as preview in the Linkwarden app
  - change a link's URL and see the new preview
  - stop the server right after saving a link, start it again, and see the favicon and preview arrive
  - refresh a favicon on the Favourites page

  Verify that both commands pass and the apps show the expected cards.
