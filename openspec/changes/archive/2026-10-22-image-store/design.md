# Design

## Context

- Only `FAMODASH_DB_PATH` (default `./data/famodash.db`, `/data/famodash.db` in the container) points at storage. The container declares `VOLUME /data`.
- `favourite_nodes` has `url` but nothing about icons. `GET /api/favourites` returns the whole tree.
- The server already makes outgoing HTTP requests for the passkey provider list (`aaguid`), but never to user-supplied URLs.
- The link widget (a later change) will store uploaded image IDs in its widget config. The favourite explorer only needs favourites' favicons.

## Goals / Non-Goals

**Goals:**
- One image store for favicons and uploads, served by ID.
- Safe outgoing fetches to user-supplied URLs (SSRF).
- Favicons that stay correct when a link's URL changes, without a column to keep in sync.

**Non-Goals:**
- Resizing or converting images.
- SVG support (it would need sanitizing).
- Thumbnails or page screenshots.
- A per-user quota (the 1 MB limit and the 24 h cleanup bound the abuse for now).

## Decisions

### Tables
```sql
images   (id TEXT PK, source TEXT CHECK (source IN ('favicon','upload')),
          user_id TEXT NULL REFERENCES users ON DELETE CASCADE,  -- uploads only
          content_type TEXT, size INTEGER, created_at TEXT)
favicons (origin TEXT PK, image_id TEXT NULL REFERENCES images ON DELETE SET NULL,
          fetched_at TEXT NULL, failed_at TEXT NULL)
```
The file name is the image ID, under `<data dir>/images/`. Writes go to a temp file followed by a rename, so a reader never sees half a file. Deleting a user cascades their upload rows, and the cleanup sweep removes files with no row.

### Favicons resolved by origin, not stored on nodes
A link's favicon is `favicons[origin(url)].image_id`, looked up when the tree is read (one `IN (...)` query over the tree's origins). Changing a link's URL changes its icon without any update, and all users share the origin's row.

*Alternative*: an `icon_image_id` column on `favourite_nodes`. That has to be updated on every URL change and duplicates the per-origin cache.

### Refresh keeps the image ID
A refresh writes the new bytes to the same image ID (temp file + rename). References in widget configs (pre-filled favicons in the link widget) stay valid and show the new icon. That's why favicons get `max-age=86400` and uploads get `immutable`.

### Fetcher
- `images.Fetcher` owns one `http.Client` whose `net.Dialer.Control` checks the resolved IP against the blocked ranges (`netip` prefixes: 127/8, 10/8, 172.16/12, 192.168/16, 169.254/16, 100.64/10, 0/8, 224/4, ::1, fc00::/7, fe80::/10, ::/128, IPv4-mapped forms). It checks in `Control`, because that runs after DNS, so DNS rebinding can't bypass it. `CheckRedirect` caps redirects at 5, and every hop dials through the same guard.
- `FetchAllowPrivate` turns the check off.
- The page is parsed with `golang.org/x/net/html` up to 512 KB. Icons are ranked by `rel` (icon > shortcut icon > apple-touch-icon) and by the largest `sizes` up to 256, falling back to `/favicon.ico`.
- The type is sniffed from the bytes (`http.DetectContentType`, plus a check of the ICO magic), never taken from headers.
- Concurrent requests for one origin are collapsed with a `singleflight.Group`, so two users adding the same site cause one fetch.
- Background fetches (on favourite create or URL change) go to a small bounded worker pool (4 workers, queue 256, drop when full). The next read or the next `POST /api/favicons` retries.

### Cleanup by JSON convention
An upload counts as used when any `dashboard_widgets.config` contains its ID under a key named `imageId` at any depth:
```sql
SELECT 1 FROM dashboard_widgets, json_tree(dashboard_widgets.config)
WHERE json_tree.key = 'imageId' AND json_tree.value = ?
```
This keeps `images` independent of the widget kinds. The link widget only has to name its field `imageId`. The sweep runs on start and hourly from `main.go`, deletes rows of unused uploads older than 24 h, and then removes files with no row.

*Alternative*: a reference registry that kinds plug into. That's more explicit, but it couples `images` to `dashboards` for one SQL query.

### Upload as a raw body
`POST /api/images` takes the file as the request body with `Content-Type: application/octet-stream`, not as multipart. The API's CSRF defence accepts only content types that browsers can't send cross-site without a CORS preflight. `multipart/form-data` is a form type that any site can POST. `application/octet-stream` isn't, so `checkCSRF` accepts it next to `application/json`.

### Config
`FAMODASH_DATA_DIR` defaults to `./data`. The Dockerfile sets `FAMODASH_DATA_DIR=/data`. `FAMODASH_FETCH_ALLOW_PRIVATE` is parsed like `FAMODASH_TRUST_PROXY_HEADERS`. The server creates `<data dir>/images` on start (`0o750`).

### Frontend
`FavouriteNode` gets `iconImageId: string | null`. A shared `ImageIcon` component (`components/ImageIcon.tsx`) renders `<img src="/api/images/{id}">` at a given size, with `onError` falling back to lucide `Globe`. The widgets reuse it. The favourites page puts it in front of each link.

## Risks / Trade-offs

- [SSRF through IPv6 or mapped addresses] → The guard unmaps IPv4-mapped addresses before checking, with table tests for each range.
- [Slow or hostile sites] → A 10 s total timeout, read limits on the body, and at most 4 background fetches at once.
- [Disk growth from uploads] → The 1 MB limit, the 24 h cleanup of unused uploads, and uploads that need a session.
- [Favicon from a different site after a redirect] → Accepted: the favicon belongs to where the page ends up, as browsers do it.
- [Homelab links show globes by default] → `FAMODASH_FETCH_ALLOW_PRIVATE` is documented prominently in the README.

## Migration Plan

`0009_images.sql` only adds tables. Existing links get favicons on their first fetch. A one-off backfill on start queues a background fetch for every distinct link origin with no `favicons` row. Rollback: the old binary ignores the new tables and the directory.
