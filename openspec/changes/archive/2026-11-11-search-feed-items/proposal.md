# Proposal

## Why

With dozens of feeds, `/feeds` is a long timeline that can only be scrolled. There's no way to find an episode or article you remember, or to look at the items of a single feed, although the API already accepts a `feedId`.

## What Changes

- `GET /api/feed-items` gets a `q` parameter. Every word of the query must appear in the item's title or summary, case-insensitive, also for letters such as "Ä". It works together with `view`, `feedId` and the paging.
- `/feeds` gets a search field ("Search items" / "Einträge durchsuchen") that searches within the current view, and a feed filter ("All feeds" / "Alle Feeds" or one of the user's feeds) that applies to every view. A search without results says "No matching items" / "Keine passenden Einträge".
- The search covers only the items the server keeps: the newest `BOARDLY_FEED_LATEST_ITEMS` per feed and the flagged ones.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `feeds`: "Feeds page" gets the search field and the feed filter. "Feed API" gets the `q` parameter and the meaning of `feedId` in the flag views.

## Impact

- Backend: `backend/internal/feeds/feeds.go` (`ItemQuery.Search`, the `WHERE` clause), `backend/internal/db` (a case-folding SQL function registered with the SQLite driver), `backend/internal/httpapi/feeds.go` (parameter), with tests.
- OpenAPI: `boardly.yaml` gets the `q` parameter of `GET /feed-items`.
- Frontend: `frontend/src/features/feeds/FeedsPage.tsx` and `api.ts` (search field, feed select, query keys), translations (en, de), tests in `Feeds.test.tsx`.
- Export and import are unaffected: no stored data changes.
- No change to the feed widget, which keeps its own choice of feeds and has no search.
