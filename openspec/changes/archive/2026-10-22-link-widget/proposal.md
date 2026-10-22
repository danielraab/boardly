# Proposal

## Why

Besides whole folders, a dashboard needs single prominent links, like a tile for the mail client or the NAS. The link widget shows one link with an icon, either one of the user's favourites or a plain URL.

## What Changes

- New widget kind `link` ("Link"), available in the "Add widget" picker. Adding it opens its settings first.
- The link comes from one of two sources:
  - **Favourite**: pick a link from the favourites. The widget follows the favourite: when its URL or title changes, the widget shows the new values. A "New favourite…" button opens the favourites' own "New link" dialog, with a folder choice, on top of the settings. The saved favourite is chosen right away, and it stays even if the dashboard edit is cancelled.
  - **URL**: a plain URL that isn't stored as a favourite.
- Optional title. Without one, a favourite's own title is used. When there is no title at all, the widget shows only the URL. With a title, it shows the title on the first line and the URL on the second. Both lines are cut off with "…" when they don't fit the width.
- Optional image, pre-filled with the URL's favicon. It can be replaced by an uploaded image or removed. Optional icon from a fixed set of lucide icons. The widget shows the image if set, otherwise the icon, otherwise a globe.
- Target: "New tab" (default) or "Same tab".
- Size S, M or L, with the same scale as the favourite explorer.
- When the favourite behind a widget is deleted, the widget keeps working as a URL-only link with the favourite's last URL and title. It is stored that way on the next save.
- Link widgets are shown to visitors of public dashboards.

## Capabilities

### New Capabilities

- `link-widget`: the link widget's sources, settings, rendering, image and icon fallback, and behaviour when its favourite is deleted.

### Modified Capabilities

None.

## Impact

- Depends on `widget-model` and `image-store`. Uses `POST /api/favicons`, `POST /api/images` and `ImageIcon` from `image-store`, and the S/M/L styles from `favourite-explorer-widget` (`widgets/sizes.ts`, moved into this change if it lands first).
- Backend: a `link` kind registered in `dashboards` (config normalization including the conversion of deleted favourites and the URL/title copy, resolving the favourite's current values) and its tests. The config field for the image is named `imageId`, so the cleanup in `image-store` sees it.
- Frontend: `widgets/link/` (view, settings dialog with source tabs, favourite picker, new-favourite form, image upload, icon picker), registry entry, en/de locales and tests.
- No new dependencies.
