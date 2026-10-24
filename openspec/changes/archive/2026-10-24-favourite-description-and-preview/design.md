# Design

## Context

- Links live in `favourite_nodes` (`internal/favourites`). The Boardly UI, the Karakeep service (`internal/karakeep`) and the Linkwarden service (`internal/linkwarden`) all create and change links through `favourites.Store`, and each calls `Favicons.Queue(url)` after a create or URL change.
- `internal/images` holds the image store (`images` table, files under `<data dir>/images/`), the favicon fetcher with its guarded HTTP client (private-address protection, redirect and time limits), HTML parsing (`ParseIcons`), and the hourly `Sweep`. `images` imports `favourites`, so `favourites` can't call into `images`.
- `images.source` has a CHECK constraint (`favicon`, `upload`). `favicons.image_id` references `images (id) ON DELETE SET NULL`. Migrations run inside one transaction with `foreign_keys` on.
- The server binary is built with `CGO_ENABLED=0` for `distroless/static`. A spike showed that `golang.org/x/image` plus Go's standard library can decode, scale and encode JPEG fast enough on old hardware: about 0.6 s CPU and 100 MB peak for a 3000×2000 photo, about 0.1 s for a typical og:image.

## Goals / Non-Goals

**Goals:**
- Fetching and converting never slows a request down, and at most one image is decoded at a time.
- The precedence rules in the link-previews spec hold under concurrency (fetch vs. screenshot, fetch vs. URL change) without a "source" column.
- No new CGO dependency.

- No background job is lost: favicon fetches, preview fetches and conversions are persisted and continue after a restart.

**Non-Goals:**
- Retrying failed jobs. A job that ran and failed is done; only jobs interrupted by a restart run again.
- A general-purpose job framework with priorities, schedules or several processes. One server process owns the queue.
- Showing previews or descriptions in Boardly's UI or exposing them in `/api/favourites`.
- Reusing the favicon fetch's page load for the preview fetch.

## Decisions

### 1. Schema: two columns on `favourite_nodes`, previews in `images`

Migration `0014_link_previews.sql`:
- `ALTER TABLE favourite_nodes ADD COLUMN description TEXT` and `ADD COLUMN preview_image_id TEXT REFERENCES images (id) ON DELETE SET NULL`. A plain `ADD COLUMN` can't add a table CHECK, so "links only" is enforced in the store.
- `images` gets the source `preview`, and previews have a `user_id` like uploads, so `CHECK ((source = 'upload') = (user_id IS NOT NULL))` becomes `CHECK ((source IN ('upload', 'preview')) = (user_id IS NOT NULL))`. SQLite can't change a CHECK, so the table is rebuilt as in `0013`. **Catch:** with `foreign_keys` on, `DROP TABLE images` runs an implicit delete that sets `favicons.image_id` to NULL, and `PRAGMA foreign_keys` can't be switched off inside the migration's transaction. So the migration first copies `favicons (origin, image_id)` into a temporary table, rebuilds `images`, then restores `image_id` from the copy. A migration test checks that favicons keep their image.

*Alternative:* a separate `link_previews` table, or storing previews as `upload`. Rejected: the first still needs the `images` rebuild, and the second would mix up the upload sweep (24 h, widget references) with previews.

### 2. One persistent job table for all background work

Migration `0015_jobs.sql` adds:

```sql
CREATE TABLE jobs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,  -- FIFO order
	kind       TEXT NOT NULL CHECK (kind IN ('favicon', 'preview_fetch', 'preview_convert')),
	dedup_key  TEXT,                               -- origin / node ID / NULL
	payload    TEXT NOT NULL,                      -- JSON
	attempts   INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	started_at TEXT
);
CREATE UNIQUE INDEX jobs_pending_key ON jobs (kind, dedup_key)
	WHERE started_at IS NULL AND dedup_key IS NOT NULL;
CREATE INDEX jobs_pending ON jobs (kind, id) WHERE started_at IS NULL;
```

A small package `internal/jobs` owns the table. `images` and `favourites` both import it, and it imports neither.

- **Enqueue** is an `INSERT` that takes a `db.Querier`, so it can run inside a caller's transaction. **`favourites.Store.Create` and `Update` enqueue `favicon` and `preview_fetch` jobs in the same transaction as the link change.** A link can then never exist without its pending jobs, and the `Favicons.Queue` calls in `httpapi`, `karakeep` and `linkwarden` go away instead of being duplicated for previews.
- **Deduplication** uses the partial unique index on pending jobs:
  - `favicon` is keyed by origin with `ON CONFLICT DO NOTHING`, so a bulk save of 50 GitHub links fetches once.
  - `preview_fetch` is keyed by node ID with `ON CONFLICT DO UPDATE SET payload = excluded.payload`, so only the newest URL is fetched.
  - `preview_convert` has no key.
- **Claiming:** `UPDATE jobs SET started_at = ?, attempts = attempts + 1 WHERE id = (SELECT id FROM jobs WHERE kind = ? AND started_at IS NULL ORDER BY id LIMIT 1) RETURNING id, payload, attempts`. That's one statement, so two workers can't claim the same job.
- **Finishing:** the row is deleted after the handler returns, whether it succeeded or failed. Failures follow each kind's own rules (the favicon back-off; previews aren't retried).
- **Restart:** on start, `UPDATE jobs SET started_at = NULL` puts interrupted jobs back in the queue. A job claimed 3 times is deleted with a warning instead of being run, so an image that crashes the process can't cause a crash loop. Handlers also run under `recover`.
- **Waking workers:** enqueue sends a non-blocking signal to the worker pool of that kind, and workers also poll every 30 seconds, which covers jobs inserted inside a transaction that committed later.
- **Workers per kind:** `favicon` 4, `preview_fetch` 2, `preview_convert` 1, as in the in-memory version. Nothing is dropped any more, so there's no queue size limit.
- **Raw images of `preview_convert` jobs** are files in `<images dir>/raw/`, named in the payload. `removeOrphans` doesn't touch that directory. The sweep deletes raw files older than an hour that no pending or running job names. The file is removed when its job finishes.

*Alternatives:*
- Keep the Go channels and persist only the convert queue. Rejected: favicon and preview fetches would still be lost, and the user asked for all queues.
- Store raw images as BLOBs in `jobs`. Rejected: up to 10 MB per row bloats the database and its WAL, while files are already the image store's format.

### 3. Background stages in `internal/images`

```
 favourites.Store.Create / Update (same transaction)
        ──▶ jobs: favicon {url}, preview_fetch {userID, nodeID, url}
                              │  preview_fetch worker (2)
                              ▼
                   load page (guarded client, 512 KB, 10 s)
                   ParseMeta → description, image URL
                   fill description:  UPDATE … SET description = ?
                                      WHERE id = ? AND url = ? AND description IS NULL
                   download image (≤ max size) ──▶ raw file
                              │
 Linkwarden screenshot ───────┤  jobs: preview_convert {nodeID, userID, url, rawFile, mode}
   (mode = replace)           │  mode = fill for fetched images
                              ▼  preview_convert worker (1)
                   DecodeConfig → pixel cap → decode → scale to ≤ width (BiLinear)
                   over white → EXIF orientation → JPEG q80
                   → Save(source = preview, userID) → apply (see 4) → remove raw file
```

- **ParseMeta** sits next to `ParseIcons`. It uses the same tokenizer and reads `og:description`, `name=description`, `og:image` and `twitter:image`, the first of each kind in document order.
- **Raw files, not memory:** raw images wait as files (see 2). That caps memory with large uploads and lets conversions survive a restart.
- **One convert worker:** decoding is where the memory goes (about 100 MB for 25 MP), so there is a single worker. The FIFO order of `jobs.id` also orders fill and replace jobs in a predictable way.
- **Screenshot upload:** the Linkwarden handler writes the raw file first, then inserts the `preview_convert` job. If the insert fails, it removes the file.

*Alternative:* do the conversion in the fetch worker. Rejected: fetch workers run in parallel and would decode several images at once, and the screenshot upload would need its own path anyway.

### 4. Precedence with conditional updates instead of a source column

Every write-back is a single conditional `UPDATE`, and `RowsAffected == 0` means the result is discarded (for images, the new image is deleted right away):

| Writer | Condition |
|---|---|
| fetched description | `url = <job url> AND description IS NULL` |
| fetched preview (`fill`) | `url = <job url> AND preview_image_id IS NULL` |
| screenshot (`replace`) | `url = <url at upload> AND kind = 'link'` |
| URL change (`Store.Update`) | sets `preview_image_id = NULL` in the same transaction when the URL actually changes |

This covers every scenario in the spec. A job that runs again after a restart is harmless for the same reason: its conditions are checked when it writes back. Because the queue is FIFO, a screenshot converted before the fetched image wins (the fill finds the slot taken), and one converted after it replaces it. Changing the URL clears the slot and makes the old jobs' `url = ?` condition fail. Fetch write-backs don't change `updated_at`, because they aren't user edits.

### 5. Deleting old previews: the hourly sweep only

A replaced, cleared or orphaned preview is simply no longer referenced. `Sweep` additionally deletes `source = 'preview'` images that no `favourite_nodes.preview_image_id` references and that are older than one hour. The age check protects a just-saved preview between `Save` and the conditional `UPDATE`. The old preview's row is usually older than an hour, so it goes on the next sweep. That gives one deletion path for every case: replace, URL change, link delete and folder delete via `ON DELETE CASCADE`.

*Alternative:* delete right away in each code path. Rejected: folder deletes cascade in SQL, so a sweep is needed anyway.

### 6. Image limits and helpers

- `images.MaxSize` (1 MB) stays for uploads and favicons. Previews use `Config.Previews.MaxBytes`.
- `Sniff` and `ReadLimited` currently apply `MaxSize` internally. They get a limit parameter, or a variant that takes one.
- Previews accept PNG, JPEG, WebP and GIF. ICO is out, since it isn't a picture format.
- **EXIF orientation:** a small parser reads the orientation tag from the JPEG APP1 segment (values 1–8). Anything it can't parse counts as orientation 1. No dependency. The image is scaled first, with the width limit applied to the upright sides, and only the small result is turned or flipped. Turning a 25 MP image pixel by pixel would cost seconds of CPU on slow machines.
- **Output:** `image/jpeg` at quality 80.
- `getImage` serves `preview` with the same immutable cache header as `upload`.

### 7. Configuration

`internal/config` reads three variables into `Config.Previews`, validated like the existing ones:

| Variable | Default | Stored as |
|---|---|---|
| `BOARDLY_PREVIEW_MAX_SIZE_MB` | `10` | bytes |
| `BOARDLY_PREVIEW_MAX_PIXELS` | `25000000` | pixels |
| `BOARDLY_PREVIEW_WIDTH` | `800` | pixels |

### 8. Favicons: no automatic refresh, manual refresh endpoint

- `Favicons.Get`: a stored image is always fresh (`RefreshAfter` is removed), and the 24 h back-off stays.
- `Favicons.Queue` and the channel are replaced by `favicon` jobs (see 2). The `favicon` handler calls `Get`.
- `Backfill` and its call in `main.go` are removed.
- The new `Favicons.Refresh(ctx, url)` skips the freshness and back-off checks and uses the same `singleflight` key, so it shares the fetch with a concurrent `Get`. A failed refresh keeps the old image, as `refresh` already does.
- `POST /api/favicons/refresh` uses a `ratelimit.Limiter` keyed by user ID, with 30 per hour and a burst of 30, so a user can refresh 30 icons right away and then one every two minutes. This matches the spec's "at most 30 per hour".

### 9. API mappings

- **Store:**
  - `NewNode.Description` and `Patch.Description` (`""` clears it) are validated by `NormalizeDescription` (trim, ≤ 2048, empty → nil).
  - `ScopedNode` gets `Description` and `PreviewImageID`.
  - `Node` doesn't get either field, so `/api/favourites` stays unchanged.
- **Karakeep:**
  - `LinkContent` gets `description` and `imageUrl` (preview, else favicon).
  - `UpdateBookmarkInput.Description` moves out of `checkUnsupported`. Like `Title`, it's raw JSON, to tell a missing value from `null`.
- **Linkwarden:**
  - `description` is mapped both ways, and `"descriptions"` is dropped from `checkUnsupported`.
  - `Preview` returns the preview image when there is one, else the favicon, and `mapLinks` sets `preview` the same way. `previewTypes` keeps applying to the favicon fallback only.
  - `POST /archives/{id}` reads the multipart `file` field with `MaxBytesReader` (max size plus 64 KB for the form), sniffs it, writes the raw file and enqueues a `preview_convert` job with mode `replace`.
- **Services:** `Favicons` interfaces in `karakeep` and `linkwarden` lose `Queue`, because the store enqueues.

### 10. Frontend

The Favourites page gets a "Refresh icon" item in each link's action menu. It calls `POST /api/favicons/refresh` with the link's URL, then invalidates the favourites query. Errors (`favicon_not_found`, `blocked_address`, `rate_limited`) show as translated toasts or messages, following the page's existing error pattern. New keys go under `favourites.*` in en and de.

## Risks / Trade-offs

- **[Queue growth]** Nothing is dropped any more, so a large import of links can queue thousands of jobs. → Favicon jobs are deduplicated per origin and preview fetches per link. The rows are small, and workers work through them in order.
- **[Poison job]** A job whose image crashes the process would run again after every restart. → Each claim counts as an attempt; after 3 the job is deleted with a warning. Handlers recover from panics.
- **[Write load on SQLite]** Each job costs an insert, a claim and a delete. → That's negligible next to the page fetch it stands for. Enqueues share the link's transaction.
- **[Late wake-up]** A signal sent before the enqueuing transaction commits can find no job yet. → Workers also poll every 30 seconds, so a job waits at most that long.
- **[Existing links]** Links that exist before the upgrade get no previews or descriptions, because there's no backfill. → Documented in README and CHANGELOG.
- **[Memory]** A 25 MP image decodes to about 100 MB. → One convert worker; the pixel cap is configurable for small machines.
- **[Migration rebuild of `images`]** If restoring `favicons.image_id` is done wrong, every favicon reference is lost. → Tested with a migration test on a database that has favicons. The favicons would be fetched again only on a manual refresh, so a mistake here would be visible.
- **[Public preview URLs]** Previews are served without a session, like uploads, so anyone with the unguessable ID can load a private link's preview. → Same model as uploads and favicons; IDs are random.
- **[Third-party content]** Descriptions and images come from arbitrary pages. → Boardly doesn't render them. The apps show the description as text. Images are re-encoded, so no original bytes are served, and they're served with `nosniff` and a sandbox CSP.
- **[More outbound traffic]** Each create or URL change now loads the page once more. → Bounded by the same guarded client, limits and queue size as favicons.
- **[Favicon staleness]** With no automatic refresh, a site's new logo shows only after a manual refresh. → This is the intended behaviour.

## Migration Plan

1. Migrations `0014` and `0015` run automatically on start. There are no down migrations: rollback means restoring the database backup, as for every release.
2. After the upgrade, favicons are no longer refreshed or backfilled. Existing links keep the favicons they have.
3. Update the README sections "Favourites tree", "Favicons and images" and the configuration table, and add a CHANGELOG entry that notes the changed favicon behaviour.
