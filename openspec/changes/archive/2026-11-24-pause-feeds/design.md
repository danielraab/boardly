# Design

## Context

Feeds and items are stored once per server, and per-user state lives on `feed_subscriptions` (position, custom title, icon). Every item list (the `/feeds` views, feed widgets, public dashboards) goes through `feeds.Store.Items`, which builds its `FROM` clause per view in `itemFrom`:

- `latest` uses an inner `JOIN feed_subscriptions`, so only subscribed feeds appear.
- `read_later`, `liked` and the listen views use a `LEFT JOIN feed_subscriptions`, so they also show items of feeds the user no longer subscribes to.

The feed widget (`backend/internal/widgets/feed.go`) checks with `HasSubscription` whether any of its chosen feeds still exists before it calls `Items`, and returns `Missing` to the owner otherwise. Refresh and cleanup decide whether a feed is "subscribed" with `EXISTS (SELECT 1 FROM feed_subscriptions WHERE feed_id = …)`.

## Goals / Non-Goals

**Goals:**
- Leave paused feeds out of "Latest" everywhere with a single filter in the store.
- Keep refresh, cleanup and feed merging working without changes to their queries.

**Non-Goals:**
- Pausing fetches. A paused feed is refreshed as before (see the `feeds` delta, "Paused feeds stay up to date").
- Bulk pause, pausing for a set time, or a server-wide pause by an administrator.
- Hiding paused feeds from the media page or the media player.

## Decisions

### Store `paused_at TEXT NULL` on `feed_subscriptions`
Migration `0029_feed_subscription_paused.sql` adds the nullable column. `NULL` means not paused. The API exposes it as a boolean `paused`.
- *Why a timestamp over a boolean:* it costs the same, needs no default for existing rows, and would let a later change show "paused since…" without another migration. It matches `read_later_at` and `liked_at` on flags.
- *Alternative:* a `paused INTEGER NOT NULL DEFAULT 0`. Rejected because the timestamp holds more information at the same cost.

### Filter only the `latest` view, in `itemFrom`
The `latest` join becomes `JOIN feed_subscriptions s ON s.feed_id = i.feed_id AND s.user_id = ? AND s.paused_at IS NULL`. The other views keep their `LEFT JOIN` unchanged, so their paused items still come with `feed.subscriptionId`, title and icon, and a `feedId` filter on them still matches. With this, the `/feeds` page, widgets and public dashboards all behave correctly without extra code.
- *Alternative:* a `WHERE` condition added in `Items` for each caller. Rejected because it spreads the rule across callers, and the widget would need its own copy.

### Widget: "paused" next to "missing"
`HasSubscription` gets a sibling, or a richer variant, that reports for a widget's chosen feeds whether any of them exist and whether any of the existing ones aren't paused. In `Resolve`:
1. none exist → `Missing` (unchanged, and it takes precedence),
2. `Show == latest`, some exist and all of them are paused → owner gets `feedData{Items: [], Paused: true}`, and visitors get no widget,
3. otherwise → `Items` as today.

The frontend widget renders "These feeds are paused" for `paused`, the same way it renders `missing`. "All my feeds" never triggers the message. If every feed is paused, it shows "No items".

### API: `paused` on the subscription and the PATCH body
`Subscription` gets `Paused bool`, which `subscriptionColumns` reads as `s.paused_at IS NOT NULL`. `Patch` gets `Paused *bool`. `true` sets `paused_at` to now unless it is already set, so pausing twice keeps the first time. `false` sets it to `NULL`. `updateFeed` reads it with the existing `field(body, "paused", …, bad)` helper into a `*bool`. A wrong type gives `invalid_request` there, and `null` is rejected with the same code, because only `true` and `false` mean anything. `boardly.yaml` gets `paused` in the `FeedSubscription` schema and in the PATCH request schema.

### Export and import
`export.Feed` gets `Paused bool` (`json:"paused"`), `false` for unsubscribed feeds, and `export.Version` goes to `4`. `dataimport` accepts versions 1–4. The import's subscription insert (`refresh.go`, the import path calling `insertSubscription`) takes the paused state and sets `paused_at` to the import time when it is `true`. A missing field decodes as `false`. Duplicates aren't touched, so they keep their state.

### Feed merge keeps the surviving subscription's state
When two feeds become one and a user had both, the existing merge deletes one subscription and moves the other. The surviving row keeps its own `paused_at`. This is an edge case and not worth special rules.

### Frontend
- `features/feeds/api.ts`: `paused` on the subscription type, and a `setPaused` mutation through the existing PATCH. On success, the subscription list and the item queries are invalidated so "Latest" and widgets update without a reload.
- "Manage feeds": a menu entry that toggles between "Pause feed" and "Resume feed", plus a dimmed row (reduced opacity on icon and text, not on the drag handle or menu) with a "Paused" badge. The drag-and-drop code doesn't change.
- Feed filter on `/feeds`: only subscriptions that aren't paused are offered. When the selected one becomes paused, the filter falls back to "All feeds" through the same path used when a feed is removed.
- Widget settings: the feed search shows a "Paused" badge on paused entries.
- New translation keys in en and de under the existing `feeds.*` and widget namespaces.

## Risks / Trade-offs

- [A paused feed still costs refreshes] → Pausing is meant to be temporary and feeds are shared anyway, so the cost is the same as before for that feed. Stopping fetches was considered and rejected in exploration.
- [Paused items appear in a "Read later" widget whose only chosen feed is paused, which can look inconsistent with "Latest"] → This is intended (flags are deliberate saves) and covered by a scenario in the `feed-widget` delta.
- [Older Boardly versions import a v4 export] → They reject version 4 as unknown, which is the existing behaviour on every version bump. The user has to update first.

## Migration Plan

The migration only adds a nullable column, so existing subscriptions come out as not paused. Rollback means deploying the previous binary. The extra column is ignored by older code, and the migration doesn't need to be reverted.
