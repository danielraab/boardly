# Tasks

## 1. Backend: directory client

- [x] 1.1 Add `backend/internal/media/radiobrowser.go` with the mirror list (cached 1 h, random pick, fallback `de1.api.radio-browser.info`), `Search(ctx, q) ([]Station, error)` with the User-Agent `Boardly (+https://github.com/danielraab/boardly)`, a 10 s timeout and a 1 MB limit, and the `toResult` rules ported from TypeScript. Verify with `radiobrowser_test.go` against an `httptest` server: mapping (`url_resolved` first, up to 3 tags, bitrate 0 → null, playable), fallback when the server list fails, an error on non-200, and no forwarded client headers.
- [x] 1.2 Expose an image fetch from `images.Favicons` that downloads with the favicon rules without saving (refactor `download`, keeping `FetchLogo` on top of it). Verify that the existing favicon/logo tests pass and that a new test covers the returned content type and the rejection of SVG/HTML.

## 2. Backend: endpoints

- [x] 2.1 Add `GET /api/media/search` (requireUser, `q` trimmed to 2–100 chars → otherwise 400 `invalid_query`, per-user limiter at 30/min with burst 10 → 429 `rate_limited` with `Retry-After`, directory error → 502 `directory_unreachable`). Each result gets a signed `logo` link (HMAC-SHA256 over user ID and URL, with a key of 32 random bytes made at start). Verify with httptest tests for 401, 400, 429, 502 and a successful search with logo links.
- [x] 2.2 Add `GET /api/media/search/logo` (requireUser; constant-time signature check → 404 on mismatch, including another user's link or a changed URL; fetch via 1.2; respond with the sniffed type, `nosniff`, `CSP: sandbox`, `Cache-Control: private, max-age=3600`; any failure → 404). Verify with tests for a valid PNG, a wrong user, a tampered URL and a non-image.
- [x] 2.3 Wire the client, limiter and key in `main.go`/router deps, and verify that `go test ./...` passes.

## 3. Frontend

- [x] 3.1 Rewrite `frontend/src/features/media/radioBrowser.ts` to call `GET /api/media/search?q=` through the API helper and drop the mirror logic. Map 502 to "unreachable" and 429 to a "rate limited" error. Update `radioBrowser.test.ts` to mock `/api/media/search` and verify that no request goes to another origin.
- [x] 3.2 In `AddStationDialog.tsx`, show the logo from `station.logo` (keeping the radio-icon fallback), keep posting `logoUrl`, add the "Too many searches" state, and update the notice text. Add `media.search.rateLimited` and the changed notice to `en.json`/`de.json`. Update `MediaPage.test.tsx` and verify that it passes, including the notice scenario.

## 4. Docs and verification

- [x] 4.1 Update the README (adding stations: the search goes through the server, which needs outbound HTTPS to radio-browser.info; streams still play directly from the station) and the CHANGELOG. Verify by reading the rendered sections.
- [ ] 4.2 Run `make check` and `make test` and make sure both pass. In the running app, check in the network tab that a search for "fm4" with logos makes requests only to Boardly's origin.
