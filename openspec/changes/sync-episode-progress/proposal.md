# Proposal

## Why

The player remembers where an episode stopped only in the browser's local storage: the position is lost on another device or after clearing site data, it is forgotten when an episode ends, and there's no record of which episodes were listened to. Users can't see on a row how far they got, can't find the episodes they started, and can't mark an episode as played without listening to it.

## What Changes

- The server keeps a listen record per user and episode: the position, the duration the player measured, when listening started, when it was last updated and when it finished. Records stay until the user removes them, and an episode with a record is kept by the item cleanup like a flagged item.
- New endpoints `PUT /api/feed-items/{id}/listen` (save the position, mark as played or unplayed) and `DELETE /api/feed-items/{id}/listen` (remove from history). Items in `GET /api/feed-items` and in the feed widget's owner data get a `listen` field.
- `GET /api/feed-items` gets the views `listened`, `in_progress` and `finished`, sorted by when the user last listened.
- The player saves the position to the server for signed-in users, regularly while playing, on pause and seek, and when the page is left. It continues from the server's position, also on another device. An episode that ends is marked as finished.
- **BREAKING**: The player no longer stores anything in the browser. Visitors who aren't signed in start each episode at the beginning after a reload. Positions stored locally so far are not uploaded and are deleted.
- `/feeds` gets the view "Listened" / "Gehört" with the sub-filters "All" / "Alle", "In progress" / "Angefangen" and "Finished" / "Fertig gehört". Every episode row gets a menu with "Mark as played" / "Als gehört markieren", "Mark as unplayed" / "Als ungehört markieren" and "Remove from history" / "Aus dem Verlauf entfernen".
- Rows of episodes with a listen record show a thin progress bar at their bottom edge and "<n> min left" / "Noch <n> Min." or "Played" / "Gehört", on `/feeds` and in the owner's feed widgets.
- The feed widget's "Show" gets "Continue listening" / "Weiterhören" (`in_progress`).
- Export and import include the listen records. The export format version goes to 3.

This change builds on `improve-feed-rows` (row layout with room for the bar) and `search-feed-items` (search and feed filter, which also apply to the new views). Its delta specs assume both are archived first.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `feeds`: the views and the API get listen records, the new views and endpoints; the item cleanup keeps listened items; new requirements for the listen history, its actions and the progress shown on rows.
- `media-player`: "Remembered episode position" moves from the browser to the server and applies only to signed-in users.
- `feed-widget`: "Show" gets `in_progress`; the owner sees progress on rows; visitors get no listen data.
- `data-export`: `feeds` and `feedItems` include listened items with their `listen` record; version 3.
- `data-import`: version 3 is accepted; listen records are restored.

## Impact

- Database: new migration `0028_feed_item_listens.sql` with the table `feed_item_listens` (user, item, position, duration, started, updated, finished), deleted with the user or the item.
- Backend: `backend/internal/feeds` (store methods, views, visibility, cleanup, merge of feeds that moved), `backend/internal/httpapi/feeds.go` and `router.go` (two routes), `backend/internal/widgets/feed.go` (`in_progress`, no listen data for visitors), `backend/internal/export` (reader, document, version 3, table coverage test), `backend/internal/dataimport` (version check, restore).
- OpenAPI: `boardly.yaml` gets the two routes, the `listen` property of `FeedItem` and the new `view` values.
- Frontend: `frontend/src/features/media/player.tsx` (server saves, start position), removal of `episodePositions.ts`, `frontend/src/features/feeds` (views, sub-filters, menu, bar), the feed widget (setting, bar), translations (en, de), tests.
- Export and import: listened items and their records are exported and restored. Older Boardly versions reject a version 3 document. Without the version raise they would silently drop the history, and they would reject a widget with `in_progress`. Version 1 and 2 documents still import, without history.
