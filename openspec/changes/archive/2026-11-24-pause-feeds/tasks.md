# Tasks

## 1. Data model and store

- [x] 1.1 Add migration `backend/internal/db/migrations/0029_feed_subscription_paused.sql` with the nullable column `paused_at TEXT` on `feed_subscriptions`, and verify with `go test ./internal/db` that it applies and existing rows read as not paused
- [x] 1.2 Add `Paused bool` to `feeds.Subscription` (read as `s.paused_at IS NOT NULL` in `subscriptionColumns`) and `Paused *bool` to `feeds.Patch`. `Update` sets `paused_at` to now when pausing (keeping an existing time) and to `NULL` when resuming. Verify with store tests in `backend/internal/feeds` for pause, pausing twice, resume, a new subscription being unpaused, and another user's subscription giving `ErrNotFound`
- [x] 1.3 Leave paused subscriptions out of the `latest` view in `itemFrom` (`AND s.paused_at IS NULL` on its join). Verify with `backend/internal/feeds` tests: Latest without and with `SubscriptionIDs` naming a paused feed gives none of its items; `read_later`, `liked`, `in_progress` and `listened` still return items of a paused feed with their `feed.subscriptionId`, also with `SubscriptionIDs`; flagging and listening to an item of a paused feed still work; another user's Latest is unaffected
- [x] 1.4 Verify with `backend/internal/feeds` tests that a feed whose only subscriber paused it is still picked by the refresh query and isn't deleted by cleanup, and that merging two feeds keeps the surviving subscription's `paused_at`

## 2. HTTP API and OpenAPI

- [x] 2.1 Read `paused` in `updateFeed` (`backend/internal/httpapi/feeds.go`) with `field` into a `*bool`, rejecting a wrong type and `null` with 400 `invalid_request`. Verify with httpapi tests: PATCH `{ "paused": true }` returns `paused: true` and `GET /api/feeds` shows it; `"yes"` and `null` give 400 `invalid_request` and leave the subscription unchanged; `POST /api/feeds` returns `paused: false`
- [x] 2.2 Add `paused` to the subscription schema and the PATCH request schema in `backend/internal/httpapi/openapi/boardly.yaml`, and verify that `TestOpenAPIDocumentsMatchRoutes` still passes and the schema matches the handler's response

## 3. Feed widget backend

- [x] 3.1 Extend the chosen-feeds check in `backend/internal/widgets/feed.go`, with a store helper next to `HasSubscription`, to report whether any chosen feed exists and whether any existing one isn't paused. Return `Paused: true` with no items to the owner and no widget to visitors when `show` is `latest` and every existing chosen feed is paused. `Missing` keeps precedence. Verify with widget tests for: one of two chosen feeds paused; all chosen paused (owner gets `paused`, visitor gets no widget); one removed and one paused gives `paused`; all removed still gives `missing`; "All my feeds" with every feed paused gives empty items, not `paused`; a "Liked" widget with a paused chosen feed still returns its liked items to the owner and, when public, to visitors

## 4. Export and import

- [x] 4.1 Add `Paused bool` (`json:"paused"`) to `export.Feed`, filled from `paused_at` (`false` without a subscription), and raise `export.Version` to `4`. Verify with export tests (a paused subscription exports `paused: true`, an unsubscribed feed `false`, the document has `version: 4`) and with `TestEveryUserTableIsCovered`
- [x] 4.2 Accept versions 1–4 in `backend/internal/dataimport/parse.go`, read `paused` on feeds, and pass it to the import's subscription insert so an added subscription gets `paused_at` set to the import time when it is `true`. Verify with import tests: a paused feed is restored paused and absent from Latest; a version 3 file without `paused` imports unpaused; a duplicate keeps its current paused state; version 5 gives `invalid_import_file`

## 5. Frontend: feeds page

- [x] 5.1 Add `paused` to the subscription type and a pause/resume mutation through `PATCH /api/feeds/{id}` in `frontend/src/features/feeds/api.ts`, invalidating the subscription list and the item queries (page and widgets) on success. Verify with a Vitest test that pausing refetches "Latest"
- [x] 5.2 In "Manage feeds" (`FeedsPage.tsx`), add the "Pause feed" / "Resume feed" menu entry and show paused rows dimmed with a "Paused" badge, keeping them draggable. Add the en and de keys. Verify with tests in `Feeds.test.tsx`: pausing changes the menu label, shows the badge and removes the feed's items from "Latest"; resuming brings them back; a paused feed can still be reordered with the keyboard
- [x] 5.3 Leave paused feeds out of the feed filter, and fall back to "All feeds" when the selected feed is paused. Verify with tests in `Feeds.test.tsx` for both scenarios of the `feeds` delta ("Paused feed not in the filter", "Filtered feed paused")

## 6. Frontend: feed widget

- [x] 6.1 Render "These feeds are paused" / "Diese Feeds sind pausiert" for `paused` in `frontend/src/features/dashboards/widgets/feed/index.tsx`, like `missing`, with settings still openable. Verify with a test in `Feed.test.tsx`
- [x] 6.2 Show a "Paused" / "Pausiert" badge on paused feeds in the widget settings' feed search, keeping them selectable. Verify with a test in `Feed.test.tsx`

## 7. Changelog and integration checks

- [x] 7.1 Add an "Added" entry for pausing feeds, and a note about export format version 4, under `[Unreleased]` in `CHANGELOG.md`
- [x] 7.2 Run `make check` and `make test` once at the end (both sides changed) and verify both pass

## Workflow follow-up

- Archive the change with `/opsx:archive` once it is implemented and reviewed, so the delta specs are synced into `openspec/specs/`.
