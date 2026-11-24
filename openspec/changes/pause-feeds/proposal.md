# Proposal

## Why

Some feeds are worth keeping but are too noisy to follow all the time, such as a podcast between seasons or a news site during a busy week. Today the only way to get their items out of "Latest" is to remove the feed, which loses its position, custom title and icon. Users want to pause a feed and resume it later without losing anything.

## What Changes

- A user can pause and resume each of their feed subscriptions. The pause is per user, because feeds and items are shared and another subscriber's feed stays as it is.
- Items of paused feeds are left out of "Latest" on `/feeds` and out of feed widgets that show "Latest", for the owner and for visitors of public dashboards.
- Items the user flagged ("Read later", "Liked") or has a listen record of stay in those views and in widgets that show them, also when their feed is paused. This matches how removed feeds work today.
- On `/feeds`, the feed filter doesn't offer paused feeds. In "Manage feeds", a paused feed is dimmed and labelled "Paused" / "Pausiert", and its menu has "Pause feed" / "Feed pausieren" or "Resume feed" / "Feed fortsetzen".
- With "All my feeds", a feed widget leaves paused feeds out. When a "Latest" widget has only paused feeds left among its chosen ones, the owner sees "These feeds are paused" / "Diese Feeds sind pausiert" and visitors don't see the widget. The widget's feed picker marks paused feeds.
- Pausing only hides a feed. A paused feed is still refreshed and its items are still kept and cleaned up as before, so a resumed feed shows current items right away.
- `GET /api/feeds` returns `paused` for each subscription, and `PATCH /api/feeds/{id}` accepts it.
- The Boardly export includes `paused` for each feed, the export format version goes from 3 to 4, and imports restore it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `feeds`: pausing and resuming subscriptions (API and "Manage feeds"); "Latest" and the feed filter leave out paused feeds; refresh and cleanup state that paused subscriptions still count as subscribers.
- `feed-widget`: widgets leave out paused feeds in "Latest", show the paused message when nothing is left, and mark paused feeds in the settings.
- `data-export`: feeds carry `paused`, and the document version is `4`.
- `data-import`: version `4` is accepted, and imported subscriptions keep their `paused` state.

## Impact

- **Database:** a new migration adds the nullable column `paused_at` to `feed_subscriptions`.
- **Backend:** `backend/internal/feeds` (subscription read and update, the `latest` view query), `backend/internal/widgets/feed.go` (paused message), `backend/internal/httpapi/feeds.go` (PATCH body).
- **API:** `GET`, `POST` and `PATCH /api/feeds` responses get `paused`, and the `PATCH` body accepts it. No route is added or removed. `backend/internal/httpapi/openapi/boardly.yaml` gets the schema changes.
- **Frontend:** `frontend/src/features/feeds` (Manage feeds menu and row state, feed filter), the feed widget's rendering and settings dialog, and en and de translations.
- **Export and import:** affected. `backend/internal/export` writes `paused` and version 4. `backend/internal/dataimport` accepts versions 1–4 and restores `paused` on added subscriptions. Files without the field import as not paused. A duplicate subscription keeps its current state.
- Refresh, cleanup and the connected-app APIs (Karakeep, Linkwarden, Memos) are unaffected.
