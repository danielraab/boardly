# Proposal

## Why

Boardly's start page can show links, notes and radio, but nothing that changes by itself. RSS and Atom feeds bring news and podcasts onto a dashboard, without an account at any outside service and with the same privacy model as the rest of the app: the server fetches, the browser only talks to Boardly.

## What Changes

- New `/feeds` page where a user subscribes to RSS, Atom or JSON feeds by pasting a feed URL or a website URL (the feed is discovered from the page), reorders, renames and removes them, and browses items in the views "Latest", "Read later" and "Liked".
- Feeds and items are stored once per server: one feed per resolved URL and one item per feed and GUID. Users have subscriptions to feeds and their own flags on items, so two users with the same feed share its rows and it is fetched once.
- Each user can flag an item "Read later" and "Liked", independently. An item flagged by anyone survives cleanup, also after the user unsubscribes from its feed.
- The server refreshes subscribed feeds in the background with guarded connections, conditional requests and backoff. Dashboards only read the database.
- Two new environment variables: `BOARDLY_FEED_LATEST_ITEMS` (how many of the newest items are kept per feed, default 100) and `BOARDLY_FEED_REFRESH_MINUTES` (the refresh cycle, default 60).
- New dashboard widget kind `feed`: a glance list of one or more feeds merged newest first, of the latest, read later or liked items. News rows open their link in a new tab. Podcast episode rows play in the app's media player.
- The media player gains on-demand playback for episodes: seeking, pausing and resuming at the same position, a finished state, and a position remembered in the browser.
- Podcasts stay separate from the media library: they don't appear on `/media` or in the media widget.

## Capabilities

### New Capabilities

- `feeds`: subscribing to feeds, shared storage of feeds and items, the background refresh and cleanup, the read later and like flags, the `/feeds` page and its API.
- `feed-widget`: the `feed` dashboard widget, its settings, rendering and behaviour on public dashboards.

### Modified Capabilities

- `media-player`: plays on-demand episodes as well as live streams, with seeking, resuming, a finished state and a remembered position.
- `data-export`: exports the user's subscriptions and their flagged items, and raises the format version.
- `data-import`: imports subscriptions and flagged items without letting a file change what other users see.
- `images`: the private address protection covers feed fetches, and the sweep keeps uploads that a feed subscription uses as its icon.
- `app-shell`: adds "Feeds" to the user menu.
- `dashboard-editing`: adds a "Feeds" group with the `feed` widget to the widget picker.
- `account-deletion`: deleting an account removes its subscriptions and flags and keeps shared feeds.

## Impact

- **Backend**: new `internal/feeds` package (store, fetcher, parser, refresh loop), a new migration with `feeds`, `feed_items`, `feed_subscriptions` and `feed_item_flags`, new routes under `/api/feeds` and `/api/feed-items`, a `feed` widget kind in `internal/widgets`, and two new settings in `internal/config`.
- **Frontend**: new `/feeds` route, `features/feeds`, a `feed` widget, navigation entry, translations (en and de), and on-demand playback in `features/media/player.tsx`.
- **Dependencies**: adds `github.com/mmcdole/gofeed` to the backend.
- **API docs**: `backend/internal/httpapi/openapi/boardly.yaml` documents the new routes and schemas.
- **Documentation**: the two environment variables are documented in `README.md`.
- **Export and import**: `data-export` gains `feeds` with the user's subscriptions and flagged items, and the format version rises to 2, because older imports reject what they don't know. `data-import` reads versions 1 and 2. Unflagged items are not exported, since refreshing the feed brings them back. The `feed` widget's config is exported with its feed references mapped on import, as for the media widget.
