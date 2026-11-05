# Design

## Context

Links already get a background page fetch. The pipeline is:

- `favourites.queueFetches` queues `favicon` and `preview_fetch` jobs in the link's transaction;
- `images.Previews` loads the page through a client whose dialer refuses non-public addresses (`images/guard.go`), parses `og:*` (`images/meta.go`), and hands images to `preview_convert`;
- `preview_convert` with `Replace=false` stores a preview only when the link has none.

Files live in `blobs.Dir` directories with orphan cleanup. `/api/images/{id}` is public and isn't suitable for owner-only content.

The runtime image is `distroless/static`, so the server can't run a browser. See proposal.md for the why and specs/ for the behaviour.

## Goals / Non-Goals

**Goals:**
- Reuse the existing job queue, guarded HTTP client, meta parser, preview conversion and blob storage.
- Keep the browser fully optional and outside the app container.
- Make it impossible for captured content to run scripts on Boardly's origin or to reach private networks, including through the browser.

**Non-Goals:**
- Full-page archives (monolith or SingleFile), PDF archives, or a copy history.
- Downloading images inside the article.
- Accepting archives that connected apps upload (Karakeep's `precrawledArchiveId`, Linkwarden archives). This is a possible follow-up.
- Exposing copies through the Karakeep or Linkwarden APIs.

## Decisions

### 1. One job kind `offline_capture`, keyed by link

Payload: `{userId, nodeId, url, setPreview}`. The job is combined per link with `replaceOnConflict`, like `preview_fetch`. `queueFetches` gets an `offline` argument:
- `true` from `Create` and `Update` with a URL change; the Karakeep and Linkwarden handlers go through these;
- `false` from `favourites/import.go`, which serves both the bookmark import and the data import.

`setPreview` is `true` for these queued captures and `false` for `POST …/offline`.

Workers: two for the job kind, with a semaphore of one around the browser, so articles don't wait behind slow screenshots and the sidecar sees one tab at a time.

*Alternative:* do the capture inside `preview_fetch`. Rejected, because a refresh would also re-fetch metadata, the failure modes would get mixed up, and imports would need a second flag anyway.

### 2. State in `offline_copies`, derived "dropped" state

Migration `0025_offline_copies.sql`:

```
offline_copies(
  node_id TEXT PRIMARY KEY REFERENCES favourite_nodes(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  url TEXT NOT NULL,          -- URL the current/pending capture is for
  status TEXT NOT NULL,       -- pending | ready | failed
  error TEXT,
  title TEXT,
  article_id TEXT,            -- blob ID in <data>/offline
  screenshot_id TEXT,
  captured_at TEXT,
  updated_at TEXT NOT NULL)
```

Enqueueing upserts the row to `pending` in the same transaction. A URL change also clears `article_id`, `screenshot_id`, `title` and `captured_at`.

The job's result is written with `WHERE node_id = ? AND url = ?`. That condition, plus the cascade, gives "discard if the URL changed or the link was deleted" without any locking.

A job that `jobs.Recover` drops after three interrupts leaves a `pending` row without a job. The status read reports a `pending` row with no queued or running `offline_capture` job for that key as `failed`/`fetch_failed`. So the queue needs no drop hook.

### 3. Article: guarded fetch → go-readability → bluemonday

- Use the `Previews`-style guarded client with its own `MaxBytes` (`BOARDLY_OFFLINE_MAX_SIZE_MB`).
- Check `Content-Type` for `text/html` or `application/xhtml+xml`.
- Run `go-readability` with the page URL, so relative URLs resolve.
- Sanitize the output with a bluemonday `UGCPolicy` that is extended for tables, `figure` and `figcaption`, and code blocks. It allows only `http` and `https` URLs, adds `target="_blank" rel="noopener noreferrer"` to links and `referrerpolicy="no-referrer"` and `loading="lazy"` to images.
- Measure the text length on the sanitized output, with 200 characters as the minimum.
- Wrap the result in a fixed HTML template with `<meta charset>`, `<meta name="referrer">`, the title as `<h1>`, the source URL, and a small inline reading stylesheet that also respects `prefers-color-scheme`.

The stored file is the final document, so serving is a plain file response. `images/meta.go` parses the same bytes, which gives `hasImage` (an og or twitter image is present) for decision 5.

*Alternatives:*
- Mozilla Readability in the sidecar's browser: needs the browser and doesn't work without one.
- `go-shiori/go-readability`: no longer maintained (last commit 2025-12). **Chosen:** `codeberg.org/readeck/go-readability/v2` (v2.1.3), the maintained fork.

chromedp is pinned to v0.21.0, a release at least two weeks old at implementation time.

The `jobs.kind` CHECK constraint gains `offline_capture`, so migration 0025 rebuilds the `jobs` table. No other table refers to it.

### 4. Screenshot: chromedp remote allocator, every request fulfilled by Boardly

Use `chromedp.NewRemoteAllocator(BOARDLY_BROWSER_URL)`, and a fresh incognito browser context per capture, disposed afterwards.

**Request filtering:** enable `Fetch.enable` with pattern `*`. On `Fetch.requestPaused`, **Boardly performs the request itself** through the guarded client and answers with `Fetch.fulfillRequest`. It answers with `Fetch.failRequest(BlockedByClient)` for a blocked address or scheme, or an error.

- Because the browser never connects anywhere on its own, DNS rebinding between a check and the browser's own lookup isn't possible.
- The sidecar can run on an `internal: true` Docker network without internet access. That also catches what CDP interception can't see, such as WebSockets and WebRTC, by failing it.
- Per-request limits: 10 s timeout, 10 MB per response, at most 300 requests per capture. Responses are streamed back with their headers, without `Set-Cookie`.

Steps: set the viewport to 1280×800 with scale 1, navigate, wait for `load` plus 1 s for late rendering (capped by the overall 30 s), then `Page.captureScreenshot` as JPEG with quality 85 and only the viewport. The result goes through `images.Converter` with width 1280, which keeps the size and orientation rules the same as for previews.

*Alternatives:*
- Check only (`Fetch.continueRequest` after resolving): open to rebinding, and the browser still needs egress.
- A Boardly forward proxy configured with `--proxy-server`: needs control over the sidecar's flags.
- Chromium in the main image: rejected by the user.

**Endpoint address:** Chrome answers DevTools requests, including the WebSocket handshake, only when the Host header is an IP address or `localhost`. A Compose service name such as `ws://chrome:9222` would be refused. Boardly therefore resolves the endpoint's host name to an address before every screenshot, since container addresses can change, and connects to the IP.

### 5. Screenshot as preview through `preview_convert`

When the job has `setPreview` and the page parsed with no og or twitter image (or couldn't be parsed), and a screenshot exists, the job writes the screenshot's bytes to the images' raw directory. It then queues `preview_convert{Replace:false, url}`. The existing conversion produces the `BOARDLY_PREVIEW_WIDTH` JPEG, stores it only when the link still has no preview at the same URL, and deletes it otherwise.

The preview is an ordinary `preview` image, public like every preview, and separate from the owner-only screenshot.

### 6. Storage and cleanup

New `blobs.Dir` at `<data>/offline`. The hourly cleanup calls `RemoveOrphans` with "ID referenced by `offline_copies.article_id` or `screenshot_id`". Replaced, URL-changed and cascaded copies therefore lose their files within OrphanAge plus one interval, which is at most about two hours, the same as previews.

Account deletion adds `offline_capture` to the per-user job deletion in `accounts.go`. The rows cascade with the nodes.

### 7. HTTP routes

The routes go in `httpapi` next to the favourites routes and need a session (no app-token middleware). The ownership check is `SELECT … FROM favourite_nodes WHERE id = ? AND user_id = ? AND kind = 'link'`, and a failed check returns 404.

- The article and the screenshot are served from blobs with the headers from the spec. They don't go through `blobs.SetHeaders`, because that sets a plain `sandbox`.
- `POST` uses the existing `ratelimit.Window`, 30 per hour per user, just as the favicon refresh does.

All four routes are added to `openapi/boardly.yaml` with an `OfflineCopy` schema.

### 8. Frontend

`FavouriteDetails.tsx` gets the button. The new `OfflineCopyDialog.tsx` is a Base UI Dialog that replaces the details while it is open and returns to them on close, the same way the edit dialog does.

- **Status:** a query on `GET …/offline` with `refetchInterval` 3000 while the copy is `pending` and the dialog is open.
- **Article:** an `<iframe src=".../offline/article" sandbox="allow-popups allow-popups-to-escape-sandbox" referrerpolicy="no-referrer">`. There is no `allow-same-origin` and no `allow-scripts`, so the sandbox holds even if the headers are lost.
- **Screenshot:** an `<img>`.
- **Cache-busting:** a `?v=<capturedAt>` query parameter, since the responses are `no-cache` but share a URL across captures.
- **Tabs:** Base UI Tabs.

i18n goes under `favourites.offline.*`.

### 9. Export and import

- `export.Link` gets `OfflineCopy *OfflineCopy` (`capturedAt`, `title`, `article`, `screenshot`). The ZIP writer adds `offline/<linkId>/article.html` and `screenshot.jpg`.
- The version stays `1`. The importer (`dataimport/parse.go`) doesn't reject unknown fields, so older servers read new files and ignore the copies. New servers read old files, which have no `offlineCopy`, as having no copies.
- The import re-sanitizes the article with the same policy and template (the file can't be trusted) and re-converts the screenshot. It then inserts a `ready` row with the file's `capturedAt` and doesn't queue a capture.

## Risks / Trade-offs

- [The sidecar is misconfigured with internet access and somebody bypasses the interception, e.g. through WebSockets] → Requests are fulfilled by Boardly, and the README and compose file put the sidecar on an `internal` network. WebSockets carry no data back into the screenshot path that the attacker couldn't render anyway.
- [Proxying every subresource through Boardly makes heavy pages slow] → The 30 s cap, the 300-request cap and a single browser slot. A screenshot that times out falls back to "no screenshot", and the article is unaffected.
- [Readability extracts nothing on SPAs] → The screenshot still captures them. Without the sidecar the copy is `failed`/`no_content`, which is honest.
- [Article images hotlinked from the origin break when the site goes away, and the origin sees the viewer's IP when the copy is opened] → Accepted by the user. No referrer is sent. The screenshot preserves the visual.
- [Disk use grows with every link] → Copies are captured only for new or changed links, never for imports. They're size-capped and deleted with the link. A quota can come later.
- [Sites block headless browsers or show cookie banners] → The screenshot shows what the site shows. That's accepted.
- [A sanitizer policy gap could allow active content] → The CSP sandbox without scripts, the iframe sandbox and `nosniff` each block scripts on their own. Tests cover the known XSS vectors (script, event handlers, `javascript:`, SVG, `srcdoc`, meta refresh).

## Migration Plan

- The new migration only creates a table, and existing links get no copies. Users can request one with Refresh.
- The sidecar is opt-in: without `BOARDLY_BROWSER_URL` nothing changes except article capture for new links.
- Rollback: an older binary ignores the table and the `offline` directory. Pending `offline_capture` jobs of an unknown kind stay in `jobs` without a worker. The migration down/cleanup note in the README tells admins to delete them, or the next version cleans them up.

## Open Questions

- The exact sidecar image tag (`chromedp/headless-shell:<version>`) for `compose.yaml`.
