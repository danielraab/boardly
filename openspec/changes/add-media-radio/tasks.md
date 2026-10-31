# Tasks

## 1. Data model and store

- [x] 1.1 Add migration `0023_media_sources.sql` (table, index, icon CHECK as in design §1) and verify `make test` runs the migration on a fresh database
- [x] 1.2 Create `backend/internal/media` with a `Store` (list, get, create at end, update, reorder with `invalid_order`, delete, all scoped by user) and kind/title validation (`invalid_kind`, `invalid_title`); verify with Go tests covering another user's source (not found), reorder with a missing ID, and cascade on user deletion

## 2. URL check and logo

- [x] 2.1 Export a guarded HTTP client constructor from `internal/images` and use it in the favicon code without behaviour change; verify the existing images tests still pass
- [x] 2.2 Implement `media.Checker` (scheme check, http→https retry, playlist parsing for PLS/M3U, HLS detection, audio content-type check, redirect rules, 10 s deadline, 64 KB playlist limit) and verify with httptest cases for each "Stream URL check" scenario, including private-address blocking
- [x] 2.3 Implement the logo fetcher in `internal/images` (favicon download rules, saved as `upload` for the user) and verify a PNG logo is stored and a 404 logo leaves the station without icon
- [x] 2.4 Extend the upload cleanup query to keep images used by `media_sources.icon_image_id` and verify with a cleanup test for a 25-hour-old station icon

## 3. REST API

- [x] 3.1 Add `httpapi/media.go` with `GET/POST /api/media`, `PATCH/DELETE /api/media/{id}`, `PUT /api/media/order`, wire the checker and logo fetcher, register routes behind the session middleware; verify with httptest tests for create (201, resolved URL), check errors (`invalid_url`, `stream_not_https`, `stream_unreachable`), reorder, delete (204), 401 without session and 404 for foreign IDs

## 4. Media widget (backend)

- [x] 4.1 Add the `media` kind in `internal/widgets` (Normalize limits: 1–50 sources, title, layout, size, `showIfPublic`; Resolve for owner and visitors, skipping deleted sources) and register it; verify with tests for `invalid_layout`, flag off for visitors, only shown sources in the visitor response, and all-sources-deleted behaviour

## 5. Media player (frontend)

- [x] 5.1 Implement `MediaPlayerProvider` and `useMediaPlayer` (single `<audio>`, play/toggle/stop, status from media events, live resume on play, volume/muted in guarded `localStorage`) and mount it in `__root.tsx`; verify with Vitest tests for switching stations, pause/resume, volume persistence and storage failure
- [x] 5.2 Build the `MiniPlayer` bar (icon, title, play/pause, loading state, volume + mute hidden below 480 px, close, error message with "Try again", `--player-height` content padding) and verify with Testing Library tests for each control's accessible name and the error state
- [x] 5.3 Add lazy `hls.js` loading for `.m3u8` URLs without native support (add the dependency with pnpm) and verify with a test that the module is imported only for HLS sources
- [x] 5.4 Wire Media Session metadata and play/pause/stop handlers, and stop playback on sign-out or user change; verify with tests using a mocked `navigator.mediaSession` and a sign-out test
- [x] 5.5 Verify playback survives navigation: a test that renders the router, starts a station, navigates to another route and asserts the same audio element is still playing

## 6. Media page (frontend)

- [x] 6.1 Add `features/media/api.ts` (TanStack Query hooks for list/create/update/delete/reorder with optimistic reorder) and verify with tests against mocked fetch
- [x] 6.2 Add the `/media` route (auth guard, `wide` width, document title, empty state) and the station list with play/pause button, host, edit/delete menu and drag-and-drop/keyboard reordering; verify with tests for the "Media page" and "Manage stations" scenarios, including deleting the playing station
- [x] 6.3 Add `features/media/radioBrowser.ts` (mirror selection with fallback, debounced/abortable search, result mapping with `playable`) and verify with unit tests using mocked fetch, including that nothing is requested before typing
- [x] 6.4 Build the "Add station" dialog: Search tab (notice with external link, results with not-playable and "Added" states, loading and directory-down messages) and Enter URL tab (name, URL, HTTPS hint, translated check errors); verify with tests for each "Station search", "External directory notice" and "Add a station by URL" scenario
- [x] 6.5 Add the edit dialog with title, URL and the shared `IconChooser` (image/symbol/none) and verify that renaming and changing the icon are saved

## 7. Media widget (frontend)

- [x] 7.1 Add the `"media"` widget group and `mediaKind` with settings (source search, ordered chosen list with drag/keyboard reorder and remove, title, layout, size, public flag with hint, empty-library message with link) and verify with tests for the settings scenarios and the picker group order in German
- [x] 7.2 Implement the widget view (list and tiles, S/M/L, playing/paused mark from the player, no playback in edit mode, "These stations no longer exist" for the owner) and verify with tests for the rendering and deleted-source scenarios, including a visitor playing a station

## 8. Shell, translations and docs

- [x] 8.1 Add the "Media" user menu link between Notes and Attachments and verify the updated app-shell menu tests
- [x] 8.2 Add all en/de translation keys (`media.*`, `player.*`, widget strings) and verify the "en and de have exactly the same keys" test in `i18n.test.ts` passes
- [x] 8.3 Document the Media page, player, radio-browser.info notice and the HTTPS-only rule in README, add a CHANGELOG entry, and verify `make check` and `make test` pass
- [ ] 8.4 Manually run the app, add a station by search and by URL, play it, navigate between pages, and play it from a public dashboard as a signed-out visitor
