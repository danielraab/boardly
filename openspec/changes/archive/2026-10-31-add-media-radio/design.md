# Design

## Context

See proposal.md for the motivation. These parts of the current code shape the approach:

- The web app is a single-page app with one root layout (`frontend/src/routes/__root.tsx` → `AppShell`). Only something mounted there survives navigation. An `<audio>` element inside a page or widget stops when the route changes.
- Boardly sends no global Content-Security-Policy, so the browser may load audio from any origin. `Content-Security-Policy: sandbox` is set only on served images, blobs and attachments.
- `backend/internal/images` already has a guarded HTTP dialer (`guard.go`: `IsBlocked`, `guardedDialer`) that respects `BOARDLY_FETCH_ALLOW_PRIVATE`, plus image sniffing, size limits and `Store.Save`. Image cleanup (`cleanup.go`) keeps uploads that something refers to.
- Widget kinds register in `backend/internal/widgets/widgets.go` with a `dashboards.Kind{Normalize, Resolve}`. `Resolve` gets a `Viewer` and decides what a visitor sees. The frontend registry is `features/dashboards/widgets/registry.ts`, with `widgetGroups`.
- Icons of notes and note categories are "a symbol or an uploaded image, never both" (`icon_symbol` / `icon_image_id` with a CHECK constraint), edited with `components/IconChooser.tsx`.
- Reordering lists uses `@dnd-kit/sortable`, as on the favourites page.

## Goals / Non-Goals

**Goals:**
- Playback that survives navigation and is independent of which page started it.
- No audio passes through the server, now or later in this design.
- A data model and player API that an `embed` kind can be added to without migrating data.

**Non-Goals:**
- Proxying `http://` streams, ICY "now playing" metadata, recording, alarms or sleep timers.
- Syncing playback state across tabs or devices. Each tab has its own player.
- Searching radio-browser.info from the server, or caching its data.
- Embed sources (Spotify, YouTube, SoundCloud). These come in a follow-up change.

## Decisions

### 1. A separate `media_sources` table, not favourites or widget config
```sql
CREATE TABLE media_sources (
  id            TEXT PRIMARY KEY,
  user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  kind          TEXT NOT NULL,            -- 'stream' (later: 'embed')
  title         TEXT NOT NULL,
  url           TEXT NOT NULL,
  icon_symbol   TEXT,
  icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
  position      INTEGER NOT NULL,
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL,
  CHECK (icon_symbol IS NULL OR icon_image_id IS NULL)
);
CREATE INDEX media_sources_user ON media_sources (user_id, position);
```
`kind` is a free `TEXT` column that the store validates, not a CHECK constraint. Adding `embed` later then needs no table rebuild in SQLite. Positions are dense integers rewritten on reorder, which is simpler than fractional ranks for lists this short.
*Alternatives:* a "stream" flag on favourite links would mix playback into bookmarks and into the Karakeep and Linkwarden APIs. Storing stations only in widget config duplicates them and leaves no page to manage them. Both were rejected while exploring.

### 2. The player is a React context in the root layout, with one long-lived `<audio>`
`MediaPlayerProvider` wraps the outlet in `__root.tsx` and renders a single `<audio>` and the `MiniPlayer` bar. It exposes `useMediaPlayer()`, which returns `{ current, status, play(source), toggle(), stop(), update(source), volume, setVolume, muted, setMuted }`. `update` shows an edited source (title, icon) in the bar without restarting it. `source` is the public shape (`id`, `kind`, `title`, `url`, `icon`), so the media page, owner widgets and visitor widgets all pass the same object. `status` is `idle | loading | playing | paused | error`, driven by the element's `playing`, `waiting`, `pause` and `error` events.
- The audio element's `src` is set only in `play()`, which always runs from a user gesture, so autoplay policies never block it.
- "Pause" on a live stream does `audio.pause()`. "Play" afterwards reloads the `src`, so playback continues at the live position instead of a stale buffer.
- Sign-out calls `stop()`. The provider also stops when the session user changes.
- Volume and muted are stored in `localStorage` (`boardly.player.volume`, `boardly.player.muted`), and every read and write is wrapped in try/catch.
- The bar is `position: fixed; bottom: 0`. While it is shown, it also renders a spacer of the same height at the end of the shell's column, so the bar covers no content.
*Alternative:* a separate popup window would keep audio playing even across reloads. But popups are blocked, clumsy on mobile and break the PWA. Rejected.

### 3. HLS through a lazily loaded `hls.js`
When `url` ends in `.m3u8` (or the URL check stored it as HLS) and `audio.canPlayType('application/vnd.apple.mpegurl')` is empty, the provider `import()`s `hls.js` and attaches it to the same audio element. This keeps the main bundle unchanged for the common MP3 and AAC case. `hls.js` is the only new frontend dependency.

### 4. Media Session from the provider
When `navigator.mediaSession` exists, the provider sets `metadata` (title, app title as artist, the icon URL as artwork) on `play()`. It registers `play`, `pause` and `stop` handlers and clears them on `stop()`. There is no `seek` handler, because these are live streams.

### 5. radio-browser.info is called from the browser
`features/media/radioBrowser.ts`:
- Picks a mirror once per session from `https://all.api.radio-browser.info/json/servers` (randomly, falling back to `de1.api.radio-browser.info` when that list fails) and keeps it in memory.
- Searches with `GET /json/stations/search?name=<q>&limit=30&order=clickcount&reverse=true&hidebroken=true`, debounced by 300 ms and cancelled with `AbortController` when the input changes.
- Maps each result to `{ name, url: url_resolved || url, favicon, country, tags, bitrate, playable }`, where `playable = url starts with "https://"`.
- Sends no click-count calls (`/json/url/<uuid>`), to keep traffic with the directory to the search itself.

The UI's "Added" mark compares a result's URL with the library's URLs after normalising them (lowercase host, no trailing slash).
*Why the frontend:* the user chose it while exploring. It needs no backend endpoint, and the notice makes the data flow visible. The cost is that a browser can't set the User-Agent that radio-browser asks for. That's acceptable at this volume.

### 6. Server-side URL check in a new `internal/media` package
`media.Checker.Check(ctx, rawURL) (finalURL string, err error)` uses an `http.Client` built on the guarded dialer from `internal/images` (exported as `images.GuardedTransport(allowPrivate)`, which the favicon fetcher now uses too), with a `CheckRedirect` that rejects `http` targets and more than 5 hops. The steps:
1. Parse and validate the scheme. For `http`, rewrite to `https` and remember that the original was `http`, so a failure maps to `stream_not_https`.
2. `GET` with an `Icy-MetaData: 0` header, read the headers, and close the body at once. Some Icecast servers ignore `HEAD`, so `GET` is used. The URL before redirects is the one saved, since redirect targets of stream servers often carry short-lived tokens.
3. If the content type or extension says playlist (`audio/x-scpls`, `audio/x-mpegurl`, `audio/mpegurl`, `application/vnd.apple.mpegurl`, `.pls`, `.m3u`, `.m3u8`), read up to 64 KB. A `#EXT-X-TARGETDURATION` or `#EXTINF` with segment lines means an HLS media playlist, and `#EXT-X-STREAM-INF` means an HLS master playlist. Both are accepted as they are. Otherwise parse the PLS `FileN=` or M3U lines, take the first entry, and repeat from step 1 once (no nested playlists).
4. Accept `audio/*`, `application/ogg` and the HLS types. Everything else is `stream_unreachable`.
The whole check has a 10-second deadline.

### 7. The logo is fetched into image storage
`media.Store.Create` calls `Store.FetchLogo`, which `main.go` sets to `images.Favicons.FetchLogo` (reusing the favicon `download` rules: guarded client, `MaxSize`, `Sniff`, accepted types) and saves the result with `Store.Save(ctx, SourceUpload, userID, data)`. A failure is logged at debug level and ignored. The cleanup query in `images/cleanup.go` gets `AND NOT EXISTS (SELECT 1 FROM media_sources WHERE icon_image_id = images.id)`. Proxying radio-browser's favicon URLs instead would break on `http` logos (mixed content) and leak the viewer's IP to arbitrary hosts.

### 8. REST endpoints
These live in `httpapi/media.go`, registered in `router.go` behind the session middleware: `GET /api/media`, `POST /api/media`, `PATCH /api/media/{id}`, `PUT /api/media/order`, `DELETE /api/media/{id}`. The responses follow the existing JSON and error conventions. A deleted station leaves its widget references dangling, and `Resolve` filters them out (as with notes), so deleting needs no layout rewrite.

### 9. The `media` widget kind
Config: `{ "sourceIds": string[], "title"?: string, "layout": "list"|"tiles", "size": "S"|"M"|"L", "showIfPublic": boolean }` (the same flag key as the note widget). `Normalize` validates shape and limits only, so sources needn't exist. `Resolve` loads the existing sources of the dashboard owner in config order with one `WHERE user_id = ? AND id IN (…)` query. For a non-owner it returns `ok=false` when `showIfPublic` is false or no source exists. Otherwise it returns `{ title, layout, size, sources: [{ id, kind, title, url, icon }] }`. On the frontend, `mediaKind` goes in a new `"media"` group appended to `widgetGroups`. Its view calls `useMediaPlayer().play(source)` and compares `current?.id` to mark the playing source.

### 10. Route and navigation
The route is `frontend/src/routes/media.tsx`, with an auth guard like `/attachments`, the `wide` page width and a user menu entry between Notes and Attachments. TanStack Query keys: `["media"]` for the list. Create, update, delete and reorder invalidate it. Reorder updates optimistically, as on favourites.

## Risks / Trade-offs

- [Many stations only offer `http`] → They are shown as not playable in search, and an `http` URL gets one automatic `https` retry. A proxy can be a later, opt-in change. Nothing in this design blocks it.
- [Stream URLs change or die] → The player shows an error with "Try again". The user can edit the URL. There is no automatic re-resolve in this change.
- [The URL check passes at save time but the stream later fails in the browser, for example because of a codec or CORS on HLS segments] → This is the same error path. HLS through `hls.js` needs CORS on the stream server, while native `<audio>` playback of MP3 and AAC doesn't.
- [radio-browser.info is down or slow] → A timeout and a clear message; the "Enter URL" tab is unaffected.
- [Visitors of public dashboards make the owner's stream URLs public] → Only when the owner turns on "Show on public dashboard". The dialog explains it.
- [iOS PWA suspends audio when the app is evicted] → A platform limitation. Media Session improves the experience, but it can't prevent eviction.
- [Two tabs both play] → Accepted. Each tab has its own player, as Non-Goals state.
- [Playlist entries with tokens] → Some `.pls`/`.m3u` files list stream URLs with session tokens (seen with streamabc.net), and the resolved URL is what gets saved. If such a token expires, the station fails with the player's error and the user re-adds it from the playlist URL. Storing the playlist URL and resolving it on every play would need a server round trip per play; deferred until it proves to be a problem.

## Migration Plan

One new migration, `0023_media_sources.sql`, that only adds a table and an index. Rolling back to an older binary leaves the table unused and harmless. No data migration, and no new configuration.
