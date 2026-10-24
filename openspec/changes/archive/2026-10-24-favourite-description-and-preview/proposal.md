# Proposal

## Why

Karakeep and Linkwarden show a description and a preview image on every bookmark card, but Boardly has neither (issue #18): the apps show empty cards, Boardly rejects descriptions that the apps send, and the Linkwarden extension's "upload screenshot" fails with an error. Storing both on links makes the connected apps look and work as their users expect.

## What Changes

- Links get a **description** (one field, at most 2048 characters). When a link is created or its URL changes, the server reads the page's `og:description` or `<meta name="description">` and stores it, but only when the link has no description yet, so text the user wrote always wins.
- Links get a **preview image**, owned by the link. When a link is created or its URL changes, the server reads the page's `og:image` or `twitter:image` and stores it, but only when the link has no preview. Changing the URL removes the old preview first. A screenshot uploaded by the Linkwarden extension always replaces the preview. The server never deletes a preview on its own.
- Preview images up to `BOARDLY_PREVIEW_MAX_SIZE_MB` (default 10) and `BOARDLY_PREVIEW_MAX_PIXELS` (default 25 000 000) are accepted, and converted in the background into a JPEG of at most `BOARDLY_PREVIEW_WIDTH` pixels wide (default 800), with transparency flattened onto white.
- **Karakeep API**: bookmarks return `content.description` and `content.imageUrl` (the preview, or else the favicon). `bookmarks.updateBookmark` accepts `description`. `note` stays rejected.
- **Linkwarden API**: links return `description`. `POST /api/v1/links` and `PUT /api/v1/links/{id}` accept `description`. The preview endpoint serves the preview image, or else the favicon. `POST /api/v1/archives/{id}` accepts the extension's screenshot as the link's preview.
- **Favicons are no longer refreshed automatically.** **BREAKING** (behaviour): the 30-day refresh and the startup backfill for links without a favicon are removed. Instead, each link on the Favourites page gets a **"Refresh icon"** action that fetches its site's favicon again right away, ignoring the 24-hour back-off after a failure, rate limited per user.
- **Background jobs are persisted**: favicon fetches, preview fetches and conversions are queued in one database table instead of in memory, so a restart or a busy server no longer drops any of them.
- Boardly's own UI doesn't show or edit descriptions and previews yet, and has no upload for previews. The Boardly API (`/api/favourites`) doesn't expose them either.

## Capabilities

### New Capabilities

- `link-previews`: description and preview image of links — what is fetched from the page and when, the precedence rules, image limits and conversion, and the cleanup of previews.

### Modified Capabilities

- `images`: the image store gets the source `preview`; the shared favicon cache drops the 30-day refresh; a manual favicon refresh is added; background jobs survive restarts; the cleanup also removes unused previews.
- `favourites`: the Favourites page gets a "Refresh icon" action on links.
- `karakeep-api`: bookmarks return the description and preview image; `updateBookmark` accepts `description`.
- `linkwarden-api`: links return the description; create and update accept it; previews serve the preview image with the favicon as fallback; the archive upload stores a screenshot as preview.

## Impact

- **Database**: migrations add `description` and `preview_image_id` to `favourite_nodes`, allow the image source `preview`, and add a `jobs` table.
- **Backend**: new `internal/jobs` (persistent queue), `internal/images` (new preview fetcher and converter, favicon jobs instead of the in-memory queue, favicon refresh, cleanup), `internal/favourites` (new fields, clearing the preview on URL change, enqueuing jobs in the link's transaction), `internal/karakeep`, `internal/linkwarden`, `internal/httpapi` (refresh endpoint, archive upload), `internal/config` (three new variables), `main.go` (workers, backfill removed).
- **Frontend**: a "Refresh icon" action and its translations (en, de) on the Favourites page, and tests.
- **Dependencies**: `golang.org/x/image` (WebP decoding and scaling). No WebP encoder, no CGO.
- **Docs**: README sections "Favourites tree", "Favicons and images" and the configuration table.
- **Network**: one more page load per new link or URL change (the preview fetch), under the same private-address protection as favicons.
