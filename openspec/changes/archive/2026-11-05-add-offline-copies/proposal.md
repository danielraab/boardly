# Proposal

## Why

A favourite is only a pointer: when the page changes, moves behind a paywall or disappears, the user loses what they saved. Karakeep keeps a readable copy and a screenshot of every bookmark. Boardly already fetches every new link's page in the background for its preview, so it can keep an offline copy as well and show it right where the user looks at a link: its details dialog.

## What Changes

- New: every link gets an **offline copy**, made of
  - a **readable article**, the page's main content extracted with a readability algorithm and sanitized, with its images left as links to the original;
  - a **viewport screenshot** of the page (1280×800), taken by an optional headless browser sidecar.
- The copy is captured in the background when a link is created (in the UI, the API or by a connected app) and when its URL changes. Bookmark imports and data imports **don't** capture copies.
- New config:
  - `BOARDLY_BROWSER_URL`, the Chrome DevTools endpoint of the sidecar. When it isn't set, no screenshots are taken and copies have only the article.
  - `BOARDLY_OFFLINE_MAX_SIZE_MB`, default `5`, the size limit for the fetched page.
- Every request the browser makes is checked against the private address protection. Requests that fail the check are blocked.
- Copies are visible **only to the owner**:
  - new session-only routes under `/api/favourites/{id}/offline` to get the copy's status, its article and its screenshot, and to request a new capture;
  - the article is served with a strict sandbox CSP and no referrer.
- The favourite details dialog gets an **"Offline copy" / "Offline-Kopie"** button. It opens a modal with:
  - Article and Screenshot tabs;
  - the capture time;
  - a **Refresh** / **Aktualisieren** button that queues a new capture;
  - "Saving…" and "Failed" states.
- A link whose page declares **no** `og:image` or `twitter:image` gets its screenshot as its **preview**, as long as it has no preview when the capture finishes. Refresh never sets the preview, so a removed preview stays removed.
- One copy per link: a new capture replaces the old copy, a URL change removes it, and deleting the link deletes it.
- The export contains the copies and the import restores them. The export format version stays `1`, because the new field is optional and older importers ignore unknown fields.

## Capabilities

### New Capabilities

- `offline-copies`: capturing, storing, replacing and deleting a link's offline copy; the readable article and the screenshot; the browser sidecar and its configuration; owner-only API routes and safe serving.

### Modified Capabilities

- `favourite-details`: "Offline copy" button in the details dialog, and the offline copy modal.
- `link-previews`: "Fetched values only fill empty fields" adds the screenshot as preview for pages without a preview image.
- `images`:
  - "Private address protection" also covers page captures and every request of the browser sidecar;
  - "Background jobs survive restarts" also covers offline captures, combined per link.
- `bookmark-import`: "Fetches for imported links" says imported links get no offline copy.
- `data-export`: links in the export document have an `offlineCopy`, and the archive contains its files.
- `data-import`: offline copies are restored from a ZIP export, and imported links don't capture a copy.

## Impact

- **Backend:**
  - new package `backend/internal/offline`, with capture job, readability and sanitizing, the CDP screenshot client with request filtering, the store and cleanup;
  - a new job kind `offline_capture`;
  - migration `0025_offline_copies.sql`;
  - `favourites.queueFetches` gets a flag so imports skip the capture;
  - preview fallback through the existing `preview_convert` job;
  - account deletion removes the user's pending capture jobs;
  - new config in `backend/internal/config`.
- **New Go dependencies:**
  - `github.com/go-shiori/go-readability`;
  - `github.com/microcosm-cc/bluemonday`;
  - `github.com/chromedp/chromedp`.
- **API:** four new routes under `/api/favourites/{id}/offline`, documented in `backend/internal/httpapi/openapi/boardly.yaml`. Connected-app APIs are unchanged.
- **Frontend:**
  - `FavouriteDetails.tsx` gets the button;
  - new `OfflineCopyDialog.tsx` with a sandboxed iframe and the screenshot;
  - en/de i18n keys.
- **Deployment:** `compose.yaml` gets an optional `chrome` service (`chromedp/headless-shell`) on its own network, and the README documents `BOARDLY_BROWSER_URL` and `BOARDLY_OFFLINE_MAX_SIZE_MB`. The runtime image stays distroless.
- **Export and import (adapted):**
  - `backend/internal/export` adds `offlineCopy` (`capturedAt`, `title`, `article`, `screenshot` file paths) to link records, and `offline/<linkId>/article.html` and `offline/<linkId>/screenshot.jpg` to ZIP exports;
  - `backend/internal/dataimport` restores the copies from a ZIP;
  - the format version stays `1`.
- **Storage:** an article is up to a few hundred KB and a screenshot about 100–300 KB, so there is noticeable extra disk use per link.
