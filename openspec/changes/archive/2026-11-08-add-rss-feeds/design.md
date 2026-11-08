# Design

## Context

See `proposal.md` for the motivation and `specs/` for the behaviour. The points below shape the approach.

- Everything in Boardly belongs to one user, with the exception of favicons, which are a shared cache. Feeds are the second shared thing, and the first that users' actions can reach.
- `images.GuardedTransport` already protects outbound fetches. `bluemonday` and `x/net/html` are already dependencies.
- The media player owns one `<audio>` element and treats everything as live: `resume` reloads the source and `onEnded` is an error.
- The import accepts only the current export version, and every string in a widget's config is mapped through the imported IDs, so `feedIds` follow without extra code.
- Each deployment is a single binary on SQLite, so writes are serialized.

## Goals / Non-Goals

**Goals:**

- A feed and its items exist once per server, however many users have them.
- Nothing one user does can change what another user sees, apart from the feed's real content.
- Dashboards never wait on, or cause, an outbound request.

**Non-Goals:**

- A full reader: no read or unread tracking, no article text.
- Detecting that two different URLs carry the same feed, or the same article in two feeds.
- Item thumbnails and podcast artwork.
- Syncing the episode position between devices.
- OPML import and export.

## Decisions

### Four tables, split by who owns what

```
feeds               url UNIQUE, title, site_url, etag, last_modified,
                    last_success_at, last_fetched_at, next_fetch_at, failures
feed_items          feed_id → feeds CASCADE, identity, UNIQUE(feed_id, identity),
                    title, link, summary, item_time, first_seen_at,
                    audio_url, audio_type, audio_seconds,
                    imported_by → users CASCADE (NULL for ordinary items)
feed_subscriptions  user_id → users CASCADE, feed_id → feeds, UNIQUE(user_id, feed_id),
                    custom_title, icon_symbol, icon_image_id, position
feed_item_flags     user_id → users CASCADE, item_id → feed_items CASCADE,
                    PRIMARY KEY(user_id, item_id), read_later_at, liked_at
```

The two unique constraints enforce "never duplicated" in the database, so a race between two users adding one feed can't produce a second row. A flags row with both times `NULL` is deleted. Alternative considered: per-user feeds, as with `media_sources`. It is simpler for export and account deletion but stores and fetches every feed once per user, which is what this change must avoid.

Removing a subscription leaves `feeds` in place, so a flagged item keeps its feed's title without copying it onto the item. This replaces the earlier idea of a nullable feed reference with a stored title.

### A feed's identity is its resolved URL, and nothing inside the document

Only the normalized final URL identifies a feed: lower-case scheme and host, no fragment, no default port. Self-links in the document (`atom:link rel="self"`) are ignored, because any feed could claim another feed's URL and take over its row. Query strings are kept, so private feeds with a token stay distinct.

### Item identity is scoped to the feed

`UNIQUE(feed_id, identity)` means a malicious feed can't overwrite another feed's items by reusing their GUIDs. The same scoping means an article published in two feeds of one site is stored twice. Avoiding that would require trusting GUIDs across feeds.

### Merge on a permanent redirect

When a refresh gets a 301 or 308 to a URL no feed has, the feed's URL is updated. When another feed has that URL, both merge in one transaction: subscriptions and flags move to the surviving feed, colliding rows collapse (the earliest flag time wins, and a user's older subscription position is kept), and the other feed is deleted. Temporary redirects don't change the URL.

### Refresh loop with a `next_fetch_at` column, not the jobs queue

A ticker runs every minute, picks due feeds that have subscribers, and fetches them with at most 4 workers. Each feed's `next_fetch_at` is already persistent, so it survives restarts. Queueing through `internal/jobs` would duplicate that schedule and leave one queued row per feed in a queue meant for one-off work. Each new feed's first refresh offset is spread within the cycle, so a restart doesn't fetch everything at once. Parsing happens outside the write transaction, keeping SQLite write locks short.

### The server fetches, then `gofeed` parses

Fetching stays in Boardly so it uses the guarded transport, size limit, timeout and conditional headers. `gofeed` only parses bytes, which gives RSS, Atom, JSON Feed and iTunes tags through one model. Alternative considered: writing a parser, which would mean handling years of malformed real-world feeds.

Text passes through `bluemonday`'s strict policy and is unescaped before storing, so the frontend renders plain text only.

### Cleanup is a single rule

```sql
DELETE FROM feed_items
WHERE feed_id = ?
  AND id NOT IN (newest N by item_time, id)
  AND id NOT IN (SELECT item_id FROM feed_item_flags)
```

Running it after each refresh keeps feeds bounded, and the hourly sweep applies a lowered limit and removes orphaned feeds. An entry beyond the newest N of a document is never inserted, so a long feed doesn't churn.

### Imported items are private until the feed confirms them

An export contains item content, and a crafted file could otherwise inject items that other subscribers see. An import therefore only ever sets flags on an existing item. A missing item is inserted with `imported_by` set, and every read filters on `imported_by IS NULL OR imported_by = :user`. A refresh that returns the same identity overwrites the content and clears `imported_by`. A feed created by an import takes nothing from the file but its URL, and it isn't treated as known until a fetch has succeeded, so adding it goes through the normal check.

If another user's private row holds the identity, the item is skipped and reported. That is rare and loses only a restored flag, whereas allowing it would show one user's unverified content to another.

### Feed icons are shared favicons

The image store only has per-user uploads and shared favicons. A feed's default icon is the existing shared favicon of its site, looked up through `Favicons`, so it isn't stored per user. A user's own icon lives on the subscription. Channel artwork would need a new shared image source, so it is left out.

### Episodes are a second source kind in the existing player

`PlayableSource.kind` gains `"episode"`, plus an optional subtitle for the feed's title. `start`, `resume`, `onEnded` and the media session branch on that kind, and `status` gains `finished`. Streams keep their behaviour. Positions live under one `localStorage` key as a list ordered by last use, capped at 200, so stale entries can't accumulate.

### One widget query

The `feed` kind resolves from the database in one joined query per widget, filtered by owner, view and subscription IDs. An empty `feedIds` means all feeds. Items are mapped to a separate visitor shape, so flags and URLs can't be added to the visitor response by accident. The dashboard lookup also sends every widget's stored config, so the kind strips `feedIds` from it for visitors through a new optional `VisitorConfig` hook on `dashboards.Kind`.

### Export format version 2, import reads 1 and 2

Version 1 readers reject unknown versions, so the new sections need a new version. `dataimport/parse.go` accepts both, and a version 1 document has no `feeds`.

## Risks / Trade-offs

- [An `http`-only enclosure fails on an `https` page] → The audio URL is upgraded, and failures show the episode error with "Try again". Some old podcasts won't be playable.
- [Audio hosts see the listener's IP address, and may run tracking prefixes] → The same as for radio streams. Visitors only reach the host after activating play.
- [A private feed's episode URLs reach visitors of a public widget] → The feed URL is never sent, and the settings dialog says what visitors can see.
- [Two users with the same private URL share a row] → They already share the secret. Identical URLs fetch identical content.
- [Many feeds on a small server] → At most 4 concurrent fetches and short write transactions. A 32 MB limit bounds memory per fetch. It is that high because long-running podcasts really do publish feeds of over 20 MB, and only the newest items are stored.
- [Merging feeds is complex for a rare event] → It is one transaction with a dedicated test, and it is required for the no-duplicates promise to hold over time.
- [Feeds with unstable GUIDs produce repeats] → Items are bounded by the cleanup. Not solved further.

## Migration Plan

Migration `0027` only adds tables. Nothing existing changes, and the refresh loop idles until someone subscribes. Rollback means running the previous binary, which ignores the new tables. Exports made with version 2 can't be imported by older binaries, as intended.
