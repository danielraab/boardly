# Design

## Context

- **Player** (`frontend/src/features/media/player.tsx`): one `<audio>` element in `MediaPlayerProvider`, inside `AppShell`. For episodes, `start()` reads a position from `episodePositions.ts` (localStorage, 200 entries) and seeks there on `loadedmetadata`. `rememberPosition()` runs every 10 s of `timeupdate`, on pause, seek and detach. `onEnded` forgets the position. `PlayableSource` has `id`, `kind`, `durationSeconds`. The provider already has `useMe()`.
- **Items** (`backend/internal/feeds/feeds.go`): `itemFrom(view)` returns the FROM clause and sort key of a view, and `Store.Items` adds filters and a keyset cursor on `(sortKey, id)`. `visibleItem` decides visibility for `PATCH`. Flags live in `feed_item_flags`, and `trim`, `Cleanup` and the merge of moved feeds (`refresh.go`) all special-case them.
- **Widget** (`backend/internal/widgets/feed.go`): `Resolve` calls `Store.Items` with the widget's `show` as the view, and maps items to `publicFeedItem` for visitors.
- **Export and import**: `export.Version = 2`; `dataimport/parse.go` accepts `1` and `export.Version`. `RestoreFlagsTx` skips items without flags. `export/coverage_test.go` fails for tables whose user data the export doesn't cover.
- The delta specs assume `improve-feed-rows` and `search-feed-items` are archived first (the row layout with its bottom slot, and `q`/`feedId` in `Items`).

## Goals / Non-Goals

**Goals:**
- One source of truth for a signed-in user's episode positions: the server.
- History views and the widget come from the same `Items` query as the other views, with paging, search and feed filter.
- No extra request to resume an episode.

**Non-Goals:**
- Read state for articles.
- Offline queueing of saves. A save that fails is dropped, and the next one carries the newer position.
- Merging positions from two devices that play at the same time. The last save wins.
- Uploading the old localStorage positions (the user decided against it).
- Playback speed, chapters, or a queue.

## Decisions

### Table `feed_item_listens`
Migration `0028_feed_item_listens.sql`:

```sql
CREATE TABLE feed_item_listens (
	user_id          TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	item_id          TEXT NOT NULL REFERENCES feed_items (id) ON DELETE CASCADE,
	position_seconds INTEGER NOT NULL,
	duration_seconds INTEGER,
	started_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL,
	finished_at      TEXT,
	PRIMARY KEY (user_id, item_id)
);
CREATE INDEX feed_item_listens_item ON feed_item_listens (item_id);
CREATE INDEX feed_item_listens_user ON feed_item_listens (user_id, updated_at DESC);
```

It's a separate table rather than more columns on `feed_item_flags`, because that table's `CHECK` requires a flag and its views are sorted by flag times. A record has a different lifecycle (frequent writes, finished state) and its own sort key. Positions are whole seconds, which is precise enough for resuming and keeps comparisons simple. The duration is stored rounded too.

### Views reuse `itemFrom`
`itemFrom` gets three cases that start from `feed_item_listens ls` joined to the item, with `LEFT JOIN`s on subscription and flags like `read_later`/`liked`, and sort key `ls.updated_at`:

| view | extra condition |
|---|---|
| `listened` | `ls.user_id = ?` |
| `in_progress` | `… AND ls.finished_at IS NULL AND ls.position_seconds > 0` |
| `finished` | `… AND ls.finished_at IS NOT NULL` |

Every view adds `LEFT JOIN feed_item_listens ls ON ls.item_id = i.id AND ls.user_id = ?` (or the inner join above) so that `itemColumns` can return the record for every item. `Item` gets `Listen *Listen`. The cursor on `(updated_at, id)` keeps working, but `updated_at` changes while someone listens, so an item can move between pages. That's accepted, as for flags that are set again. The widget gets `in_progress` by allowing it in `Normalize` and passing it to `Items`.

### Visibility
`visibleItem` also accepts an item with a listen record of the user (`OR ls.item_id IS NOT NULL`), so that history entries of unsubscribed feeds can still be changed. A record can only be created for an item that is already visible. An item from someone else's public dashboard isn't, so its save gets 404 and the player ignores it.

### Endpoint semantics
One `PUT …/listen` covers saving, "Mark as played" (`finished: true`) and "Mark as unplayed" (`positionSeconds: 0`). The player and the menu share it. `DELETE` covers "Remove from history". `PUT` is an upsert: `started_at` is set on insert only, and `updated_at` is set on every write. A left-out `durationSeconds` keeps the stored duration (`COALESCE(excluded.duration_seconds, duration_seconds)`). No rate limit: the client writes about every 15 s, and an upsert by primary key is cheap.

### Player
- `PlayableSource` gets `startAt?: number`. `episodeSource(item)` sets it from `item.listen` (0 when finished or missing). `start()` uses it instead of `readPosition`.
- A small module `episodeProgress.ts` replaces `episodePositions.ts`. `saveProgress(id, position, duration, { finished, keepalive })` calls the API, catches every error, and then patches the cached items of that ID through `queryClient.setQueriesData` on `feedItemsQueryKey` and `dashboardViewQueryKey`, so bars elsewhere update without a refetch. Finishing also invalidates both keys, because the item leaves `in_progress` lists and "Continue listening" widgets.
- Saves are made only when `useMe()` has a user. The save points are the existing ones (start, interval, pause, seek, detach), with the interval raised from 10 s to 15 s. A `pagehide` listener saves with `fetch(..., { keepalive: true })`, so closing the tab doesn't lose the last seconds. `api()` gets a `keepalive` option, passed through to `fetch`.
- `onEnded` saves `finished: true` instead of forgetting the position.
- On mount, the provider removes the localStorage key `boardly.player.positions` once, wrapped in try/catch. No upload.

### Live bar
The bar is in `FeedItemRow`, in the bottom slot from `improve-feed-rows`. For the item that `player.current` has loaded, it takes `usePlayerProgress()` (position and duration from the audio element). Otherwise it uses `item.listen`. Only the current row subscribes to the progress context, through a child component that is rendered only when the row is current, so the list doesn't re-render on every `timeupdate`. The "<n> min left" / "Played" text goes into the meta line, so screen readers get it as text, and the bar itself is `aria-hidden`.

### Episode menu
A Base UI `Menu` with the `EllipsisVertical` trigger, like the feed list's menu, placed after the play button on `/feeds` rows only (`showFlags` and not in a widget, passed as a new `showEpisodeMenu` prop). Before an action runs on `player.current?.id === item.id`, it calls `player.stop()`. `stop()` saves first through `detach`, then the action's request follows, so the action's state wins.

### Widget privacy
`publicFeedItem` has no `listen` field, so nothing leaks to visitors through the existing mapping. The owner's `feedData` passes `feeds.Item` with `Listen`. Visitor rows render no bar because `item.listen` is undefined.

### Cleanup and merge
- `trim` and `Cleanup` add `NOT EXISTS (SELECT 1 FROM feed_item_listens WHERE item_id = …)` next to the flag checks, and the feed deletion checks listened items as well.
- The merge of moved feeds copies the records of the dropped item to the kept one with `INSERT … SELECT … ON CONFLICT (user_id, item_id) DO UPDATE` and keeps the row with the larger `updated_at`, before the dropped item is deleted (which would cascade).

### Export version 3
- `export.FeedItem` gets `Listen *FeedItemListen`. The feeds and items queries use `UNION` of the flag and the listen sources, so an item with both appears once.
- `Version = 3`. `parseDocument` accepts 1, 2 and 3. A version 2 server rejects version 3 files. Without the raise it would silently drop the history (its `RestoreFlagsTx` skips items without flags) and reject dashboards with `in_progress` widgets.
- Import: `RestoreFlagsTx` becomes `RestoreItemTx` and takes the item with its optional listen record. The item is restored (or created as an imported item, as today) when it has flags or a valid record. The record is inserted with `ON CONFLICT DO NOTHING`, so an existing record wins, and invalid values or an item without audio drop the record only.
- `coverage_test.go` gets `feed_item_listens` through the exported items.

## Risks / Trade-offs

- [Write load: one request per 15 s per listener] → For self-hosted use that's trivial. SQLite handles it within the existing connection setup.
- [Two devices playing the same episode overwrite each other] → Last write wins. That's accepted and noted in Non-Goals.
- [Saves lost when the tab is killed] → `keepalive` on `pagehide` covers normal closes. A crash loses at most 15 s.
- [The history keeps old episodes forever and stops their cleanup] → That's the user's decision. Removing an entry from history releases the item to the cleanup again.
- [Items move between pages of the history views while a position is saved] → The list can show an item twice or skip one until reloaded. The query key changes on view switches, and finishing invalidates.
- [Visitors lose their local resume] → Accepted (**BREAKING**, stated in the proposal). Pause and resume within a page still work through the audio element.
- [The version raise blocks importing new exports into old servers] → That's intended. Silently losing the history would be worse.

## Migration Plan

1. Deploy: migration 0028 creates the table, and no data migration is needed.
2. On first load, clients delete `boardly.player.positions`. Episodes start from the beginning once, as decided.
3. Rollback: an older binary ignores the table, and positions are lost for that period. Version 3 exports made in the meantime can't be imported until the new version is back.
