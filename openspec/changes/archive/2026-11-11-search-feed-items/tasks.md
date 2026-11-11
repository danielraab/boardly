# Tasks

## 1. Backend

- [x] 1.1 Register `boardly_fold` with the SQLite driver in `backend/internal/db` (once, before the first open), and add a test that `SELECT boardly_fold('ÄRGER')` returns `ärger` and `boardly_fold(NULL)` is `NULL`; verify with `go test ./internal/db`
- [x] 1.2 Add `Search string` to `feeds.ItemQuery` and the per-word condition on title and summary in `Store.Items` (at most 20 words, folded in Go); verify with store tests for all words matching, a match in the summary, a non-ASCII match, search combined with `read_later`/`liked` and `SubscriptionIDs`, and paging with a cursor; run `go test ./internal/feeds`
- [x] 1.3 Read `q` in `listFeedItems` (`backend/internal/httpapi/feeds.go`), rejecting more than 200 characters with 400 `invalid_request`; add handler tests for `q`, the 201-character case and a `feedId` of another user giving no items; verify with `go test ./internal/httpapi`
- [x] 1.4 Add the `q` parameter (string, maxLength 200) to `GET /feed-items` in `backend/internal/httpapi/openapi/boardly.yaml`, and describe `feedId` as applying to every view; verify with `go test ./internal/httpapi -run TestOpenAPI`

## 2. Frontend

- [x] 2.1 Extend `feedItemsQueryOptions` in `frontend/src/features/feeds/api.ts` with `q` and `feedId` in the path and the query key; verify with `pnpm exec tsc --noEmit`
- [x] 2.2 Add the search field (debounced, with clear button) and the feed select to `FeedsPage.tsx`, kept across views, with the reset when the chosen feed is removed and the "No matching items" text; add translations in en and de (`feeds.search`, `feeds.clearSearch`, `feeds.feedFilter`, `feeds.allFeeds`, `feeds.noMatches`); verify by trying it in the dev server
- [x] 2.3 Add tests to `Feeds.test.tsx`: typing sends `q` for the current view, the search stays when switching views, choosing a feed sends `feedId`, clear empties the field, and an empty result shows "No matching items"; verify with `pnpm exec vitest run src/features/feeds`

## 3. Finish

- [x] 3.1 Run the full checks of both sides once (`make check` and `make test`) and fix what fails
