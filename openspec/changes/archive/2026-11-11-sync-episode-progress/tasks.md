# Tasks

## 1. Database and store

- [x] 1.1 Add migration `backend/internal/db/migrations/0028_feed_item_listens.sql` with the table and indexes from design.md; verify with `go test ./internal/db`
- [x] 1.2 Add `Listen` to `feeds.Item`, the `LEFT JOIN` on `feed_item_listens` in every view, and the views `listened`, `in_progress` and `finished` in `itemFrom` with `ls.updated_at` as sort key; verify with store tests for each view's content and order, paging with a cursor, and search and `SubscriptionIDs` in the new views; run `go test ./internal/feeds`
- [x] 1.3 Extend `visibleItem` to items with a listen record of the user; verify with a test that a listened item of an unsubscribed feed can be patched and that an unrelated item still gives `ErrNotFound`
- [x] 1.4 Add `Store.SaveListen(userID, itemID, position, duration *int, finished *bool)` (upsert, finished resets the position, unfinished clears `finished_at`, a left-out duration is kept, items without audio give `ErrInvalidQuery`) and `Store.DeleteListen`; verify with store tests for save, mark played, play again, mark unplayed, delete, and no audio
- [x] 1.5 Keep listened items in `trim` and `Cleanup`, and move listen records in the merge of moved feeds (the later `updated_at` wins); verify with tests for "Listened item kept", "Removed from history" and a merge where both items have a record of the same user

## 2. API and OpenAPI

- [x] 2.1 Add `PUT` and `DELETE /api/feed-items/{id}/listen` in `backend/internal/httpapi/feeds.go` and `router.go` with validation (0–172800, rounding down, duration `null` or greater than 0) and accept the new views in `listFeedItems`; verify with handler tests for 200, 204, 400 `invalid_request`, 401 and 404
- [x] 2.2 Update `backend/internal/httpapi/openapi/boardly.yaml`: the two routes, the `view` enum, and `listen` (required, nullable object) on `FeedItem`; verify with `go test ./internal/httpapi -run TestOpenAPI`

## 3. Feed widget backend

- [x] 3.1 Accept `in_progress` for `show` in `backend/internal/widgets/feed.go` and make sure `publicFeedItem` carries no listen data; verify with widget tests for an `in_progress` widget's items and order, and a visitor response without `listen`; run `go test ./internal/widgets`

## 4. Export and import

- [x] 4.1 Add `Listen` to `export.FeedItem`, include listened items and their feeds in `read.go`, and raise `export.Version` to 3; verify with export tests for "Listened episode", "Flagged item without a record" and an item with both a flag and a record listed once, and with the table coverage test in `backend/internal/export`; run `go test ./internal/export`
- [x] 4.2 Accept versions 1, 2 and 3 in `dataimport/parse.go`, and restore listen records (`RestoreItemTx`; an existing record wins, invalid records are dropped but the item's flags are still restored, items without flags and record are skipped) in `import.go`; verify with import tests for "Restore the listen history", "Existing record stays", a version 2 file, a version 4 file, and a round trip of export and import; run `go test ./internal/dataimport`
- [x] 4.3 Update the import test fixtures and the frontend import report text if it names "flagged items"; verify with `go test ./internal/dataimport` and `pnpm exec vitest run src/features/settings`

## 5. Player

- [x] 5.1 Add a `keepalive` option to `api()` in `frontend/src/lib/api.ts`, `listen` to `FeedItem` and the new views to `FeedView` in `frontend/src/features/feeds/api.ts`, and `saveListen`/`deleteListen`; verify with `pnpm exec tsc --noEmit`
- [x] 5.2 Replace `episodePositions.ts` with `episodeProgress.ts` (save through the API, ignore errors, patch cached items, invalidate on finish), add `startAt` to `PlayableSource` and set it in `episodeSource`, save only with a signed-in user, raise the interval to 15 s, save with `keepalive` on `pagehide`, mark finished in `onEnded`, and delete the old localStorage key once; verify with player tests: start at `startAt`, saves at start, interval, pause, seek and `pagehide`, finished on end, no request and no storage for a visitor, and a failed save that doesn't stop playback; run `pnpm exec vitest run src/features/media`
- [x] 5.3 Delete `episodePositions.ts` and its test, and update the tests that relied on it; verify with `pnpm exec vitest run src/features/media src/features/feeds`

## 6. Feeds page and rows

- [x] 6.1 Add the "Listened" view with the "All" / "In progress" / "Finished" toggles (mapped to `listened`, `in_progress`, `finished`), with search and feed filter applying; add translations in en and de; verify with tests in `Feeds.test.tsx` for the requests per sub-filter and the empty text
- [x] 6.2 Add the progress bar (`aria-hidden`, live for the loaded episode) and the "<n> min left" / "Played" text to `FeedItemRow`, only when `item.listen` is present; verify with tests for 45 of 60 minutes (three quarters, "15 min left"), finished ("Played", full), unknown duration (no bar), and no bar for visitor items
- [x] 6.3 Add the episode menu on `/feeds` rows with audio, with the actions shown per state, stopping the player for the loaded episode, and updating views and widgets; verify with tests for each action's request and visibility, and that the player stops for the loaded episode

## 7. Feed widget frontend

- [x] 7.1 Add "Continue listening" to the widget's "Show" choice and to the edit preview, and show the progress bar for the owner; add translations in en and de; verify with tests in `widgets/feed/Feed.test.tsx` for the setting, the owner's bar, a visitor without bar, and a finished episode leaving the widget after invalidation

## 8. Finish

- [x] 8.1 Check by hand in the dev server: resume on a second browser profile, the live bar, closing the tab mid-episode and resuming, and a visitor's public dashboard storing nothing
- [x] 8.2 Run the full checks of both sides once (`make check` and `make test`) and fix what fails
