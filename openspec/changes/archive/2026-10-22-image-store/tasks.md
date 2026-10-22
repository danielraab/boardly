# Tasks

## 1. Configuration and storage

- [x] 1.1 Add `DataDir` (`FAMODASH_DATA_DIR`, default `./data`) and `FetchAllowPrivate` (`FAMODASH_FETCH_ALLOW_PRIVATE`, default `false`) to `config.go`, create `<data dir>/images` on start in `main.go`, and set `FAMODASH_DATA_DIR=/data` in the Dockerfile. Verify with `config_test.go` cases for defaults and invalid booleans
- [x] 1.2 Add migration `0009_images.sql` with `images` and `favicons`, and verify with a migration test
- [x] 1.3 Create `backend/internal/images` with the store (save via temp file + rename, open, delete, type sniffing for PNG/JPEG/WebP/GIF/ICO, the 1 MB limit). Verify with unit tests per type, SVG rejection and the size limit

## 2. Serving and uploading

- [x] 2.1 Add `GET /api/images/{id}` with content type, `nosniff`, `CSP: sandbox` and per-source `Cache-Control`. Verify with httptest for 200, 404 and the headers
- [x] 2.2 Add `POST /api/images` (raw `application/octet-stream` body, session required) with 201, 400 `unsupported_image`, 413 `image_too_large` and 401. Verify with httptest

## 3. Favicon fetcher

- [x] 3.1 Implement the address guard (dialer `Control` with the blocked `netip` prefixes, unmapping IPv4-mapped addresses, an allow-private switch). Verify with table tests for every range and an httptest server on 127.0.0.1 being refused
- [x] 3.2 Implement page parsing and icon ranking with `golang.org/x/net/html` (512 KB limit, `/favicon.ico` fallback, at most 5 redirects, 10 s timeout). Verify with httptest sites: declared icon, fallback, SVG only, oversize, redirect loop
- [x] 3.3 Implement the per-origin cache (30-day refresh keeping the image ID, 24 h failure back-off) and `singleflight`. Verify with tests using a fake clock and a request counter
- [x] 3.4 Add `POST /api/favicons` (200, 400 `invalid_url`, 400 `blocked_address`, 404 `favicon_not_found`, 401). Verify with httptest
- [x] 3.5 Add the background worker pool, queue fetches on favourite link create and URL change, and backfill all link origins without a `favicons` row on start. Verify with a favourites test that a created link gets `iconImageId` after the worker runs

## 4. Favourites

- [x] 4.1 Return `iconImageId` for link nodes in `GET /api/favourites` (and the create/update responses), resolved by origin with one batched query. Verify with `favourites_test.go`
- [x] 4.2 Add `iconImageId` to `FavouriteNode`, add `components/ImageIcon.tsx` with the globe fallback, and show it in front of links on the favourites page. Verify with `Favourites.test.tsx` cases for an icon and the globe

## 5. Cleanup

- [x] 5.1 Implement the sweep (unused uploads older than 24 h via `json_tree` on `imageId`, then orphan files) and run it on start and hourly. Verify with a test: an old unused upload is deleted, an old referenced one and a fresh one are kept, and favicons are untouched

## 6. Docs and verification

- [x] 6.1 Document `FAMODASH_DATA_DIR` and `FAMODASH_FETCH_ALLOW_PRIVATE` in the README and `.env.example`, and verify that both list them with their defaults
- [x] 6.2 Run `make check` and `make test` and verify that both pass
