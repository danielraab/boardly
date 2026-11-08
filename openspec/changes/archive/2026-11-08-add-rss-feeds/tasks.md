# Tasks

## 1. Data and configuration

- [x] 1.1 Add migration `0027_feeds.sql` with `feeds`, `feed_items`, `feed_subscriptions` and `feed_item_flags`, with the unique constraints from the design; verify with a migration test that a second row with the same URL or the same feed and identity is rejected
- [x] 1.2 Add `BOARDLY_FEED_LATEST_ITEMS` (1–1000, default 100) and `BOARDLY_FEED_REFRESH_MINUTES` (5–10080, default 60) to `internal/config`; verify defaults, custom values and out-of-range errors in `config_test.go`
- [x] 1.3 Add `github.com/mmcdole/gofeed` to the backend and verify `go build ./...` succeeds

## 2. Fetching and parsing

- [x] 2.1 Create `internal/feeds` with a fetcher using `images.GuardedTransport`, the 32 MB and 20 second limits, 5 redirects, `https` first, and conditional headers; verify with httptest cases for 304, oversize, timeout and a blocked private address
- [x] 2.2 Add URL normalization and feed discovery from `<link rel="alternate">`; verify with table tests, including that `atom:link rel="self"` is ignored
- [x] 2.3 Add parsing to the item model: identity fallback, plain-text title and summary, `http`/`https` links only, time clamping, audio detection and `https` upgrade, newest N only; verify with RSS, Atom, JSON and podcast fixtures, including `javascript:` links and script tags

## 3. Store

- [x] 3.1 Implement subscribe, list, patch, reorder and unsubscribe, reusing a successfully fetched feed by URL; verify that two users share one feed row, and that concurrent adds create one row
- [x] 3.2 Implement item upsert that never touches flags, and item listing by view with cursor pagination and the `imported_by` filter; verify a re-seen GUID keeps its ID and flags, and that one user's flags are invisible to another
- [x] 3.3 Implement flag changes with the access rule, deleting rows without flags; verify 404 for items of feeds the user doesn't have, and independent flags
- [x] 3.4 Implement the cleanup rule and orphan feed removal; verify every scenario in "Item cleanup", including a lowered limit and a flag held only by another user
- [x] 3.5 Implement the feed merge on a permanent redirect; verify collapsed items, moved flags and a user subscribed to both feeds ending with one subscription

## 4. Background refresh

- [x] 4.1 Add the refresh loop with `next_fetch_at`, at most 4 workers, backoff to 24 hours, `Retry-After`, and the `failing` state after 3 failures, and start it with the server; verify with a fake clock that feeds without subscribers aren't fetched and that the interval is honoured
- [x] 4.2 Run the cleanup after a refresh and hourly, and verify a test asserts both paths remove old unflagged items

## 5. API

- [x] 5.1 Add the `/api/feeds` and `/api/feed-items` routes with per-user rate limiting on adding; verify with httptest for 401, 404 isolation, every error code and `already_subscribed`
- [x] 5.2 Add the routes, parameters and schemas to `backend/internal/httpapi/openapi/boardly.yaml`; verify `TestOpenAPIDocumentsMatchRoutes` passes and review the schemas against the handlers by hand

## 6. Export, import and other sweeps

- [x] 6.1 Raise the export `Version` to 2, add `feeds` and `feedItems` to the document and reader, and update `export/coverage_test.go` for the new tables; verify only flagged items are exported and no other user's data appears
- [x] 6.2 Make the import accept versions 1 and 2, and add the `feeds` part with its report; verify with `parse_test.go` and `import_test.go` that version 3 is rejected and version 1 still imports
- [x] 6.3 Implement the import rules for shared data: only URL from the file, flags only on existing items, `imported_by` for unknown items, confirmation by a refresh, no fetching; verify every scenario in "Imported feeds", including the crafted-file cases
- [x] 6.4 Add feed subscriptions to the unused-image sweep in `images/cleanup.go`; verify a used feed icon survives and an unused one is deleted
- [x] 6.5 Include feeds in account deletion; verify subscriptions and flags are removed, another user's feed stays, and `imported_by` items of the deleted user are gone

## 7. Widget backend

- [x] 7.1 Add the `feed` kind in `internal/widgets` with config validation and registration; verify every `invalid_layout` case in `feed` widget tests
- [x] 7.2 Implement `Resolve` with owner and visitor shapes; verify a visitor's response has no feed URL, subscription ID or flags, and that the flag-off case leaves no trace

## 8. Media player

- [x] 8.1 Add the `episode` kind to `PlayableSource` and branch `start`, `resume` and `onEnded` on it, adding the `finished` status; verify in `player.test.tsx` that streams behave as before and episodes resume in place and finish without an error
- [x] 8.2 Add the remembered position with its 200-entry cap and storage failure handling; verify continuing after a reload and forgetting on finish
- [x] 8.3 Add the position slider and time labels to `MiniPlayer`, the episode error text and the media session seek and artist details; verify the slider appears only for episodes and works with the keyboard

## 9. Feeds page

- [x] 9.1 Add the feeds API client and the `/feeds` route with its views and "Load more"; verify with tests for each view, the empty states and the signed-out redirect
- [x] 9.2 Add the "Add feed" dialog with translated errors; verify the dialog stays open on each error code
- [x] 9.3 Add "Manage feeds" with edit, remove confirmation, reorder and the failing state; verify keyboard reordering and that the confirmation mentions kept items
- [x] 9.4 Add item rows with flag buttons, link opening and episode playback; verify flags update optimistically, links use `noopener`, and episodes don't appear on `/media`
- [x] 9.5 Add "Feeds" to the user menu; verify the order in `AppShell.test.tsx`

## 10. Dashboard widget

- [x] 10.1 Add the `feed` widget to the registry in a new "Feeds" picker group, with its settings form; verify defaults, range validation and the empty state
- [x] 10.2 Add the widget view with rows, flag buttons, removed-feed states and the edit-mode guard; verify that flag changes refresh other widgets, and that visitors get no flag buttons

## 11. Import and export UI, docs and final checks

- [x] 11.1 Add the "Feeds" part to the import form, the report text and the export section text, in `en` and `de`; verify in `ImportExport.test.tsx`
- [x] 11.2 Document both environment variables and the feeds feature in `README.md`, and add an entry to `CHANGELOG.md`; verify the variables appear in the configuration table
- [x] 11.3 Run `make check` and `make test` and verify both pass
- [x] 11.4 Try the feature in the running app with a real news feed and a real podcast: add, refresh, flag, play, seek, reload, and view on a public dashboard
