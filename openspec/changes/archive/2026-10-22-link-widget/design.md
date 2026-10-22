# Design

## Context

- `widget-model` gives the kind registry (`Normalize(ctx, userID, config)`, `Resolve`), the settings dialog contract and `CommonWidgetFields`.
- `image-store` gives `POST /api/favicons`, `POST /api/images`, `ImageIcon`, `iconImageId` on favourite links, and a cleanup that keeps uploads referenced under a config key named `imageId`.
- `favourite-explorer-widget` introduces `widgets/sizes.ts`. Whichever change lands first creates it.

## Goals / Non-Goals

**Goals:**
- Deleted favourites degrade to URL links without bumping the layout version or touching other dashboards.
- The stored config is always enough to render the widget, even when the favourite is gone.

**Non-Goals:**
- Arbitrary lucide icons (the full set is large; a curated list keeps the bundle and the picker small).
- Showing the URL's host instead of the full URL.

## Decisions

### Config
```json
{
  "source": "favourite" | "url",
  "favouriteId": "<uuid>",          // source = favourite only
  "url": "https://…",               // always: the URL, or the favourite's last known URL
  "favouriteTitle": "Mail" | null,  // source = favourite only: last known title
  "title": "…" | null,              // the widget's own title
  "iconSource": "favicon" | "image" | "symbol" | "none",
  "imageId": "<image id>" | null,   // iconSource = image only
  "icon": "mail" | null,            // iconSource = symbol only: key from the curated list
  "target": "_blank" | "_self",
  "size": "S" | "M" | "L"
}
```
Creating a new favourite is a frontend flow only. Afterwards the widget stores `source: "favourite"`.

### Conversion and copy refresh in `Normalize`
On every save, for `source: "favourite"`, `Normalize` looks the ID up in the owner's tree:
- a link of the owner → refresh `url` and `favouriteTitle` from it
- missing, or not the owner's link → rewrite to `source: "url"`, with `title = title ?? favouriteTitle`, and drop `favouriteId` and `favouriteTitle`

This implements "converted on the next save" without a write when the favourite is deleted. The returned config is what gets stored and returned, so the editor's draft is replaced by the saved tree as usual.

*Alternative*: rewrite widget configs in the favourites delete handler. That bumps `layout_version` on every affected dashboard and gives open editors surprise 409s. It also couples `favourites` to `dashboards`.

### Where it lives
The kind is `widgets.Link(fav)` in `internal/widgets`, next to the favourite explorer, and `widgets.Register` adds it. A new `favourites.Store.Link(ctx, userID, id)` returns one of the user's links, or `ErrNotFound` for a missing node, a folder or another user's node. `Resolve` caches it per lookup with `dashboards.Cached`.

### Height
A link is one or two lines, so a fixed height only adds empty space or cuts the link off. `dashboards.Kind` gets a `FitContent` flag. For such a kind, `normalizeContent` sets the widget's height to "auto" instead of rejecting other heights, so older drafts still save. In the web app, `WidgetSettingsForm` gets a `fitContent` prop: the link settings set it, and the form then leaves out the height choice (`SizeFields` with `height={false}`) and applies "auto".

### Resolve
For `source: "favourite"`, `data = { url, title }` with the favourite's current values, or the copies when it's missing. With `iconSource: "favicon"`, `data.iconImageId` is the favicon of the current URL's origin, when one is known (see "Icon"). Otherwise a URL link has no `data`. The view uses `data` over `config` in view mode. In edit mode it uses the favourites query, as the explorer does, so a draft widget renders right away. Visitors get only these fields: no tags and no favourite ID beyond what is in `config`.

Should `config.favouriteId` reach visitors? It's an opaque UUID that grants nothing, because favourites need the owner's session. It's left in, to keep one config shape.

### Icon
The icon is one explicit choice, `iconSource`, so there is no precedence rule between an image and a symbol. `Normalize` keeps only the field that belongs to the source: `imageId` for `image` (required, ≤ 64 characters), `icon` for `symbol` (required, from the allowed set), neither for `favicon` and `none`. A config without `iconSource`, saved before it existed, is read as `image` when it has an `imageId`, `symbol` when it has an `icon`, and `favicon` otherwise.

The website icon isn't stored. `Resolve` looks up the favicon of the origin of the link's current URL (the favourite's live URL, or the plain URL) with `images.Favicons.Lookup`, cached per origin within one lookup with `dashboards.Cached`. When none is known, it queues a background fetch with `Favicons.Queue`, so a plain URL link gets its favicon on a later visit, as favourites do. This replaces the earlier pre-fill of `imageId` with the favicon, and with it the 500 ms pre-fill logic and the "auto-filled" tracking in the dialog. It also removes the old trade-off that a favourite moved to another site kept the old site's favicon.

In the web app, a `useFavicon(url, knownId)` hook gives the favicon of a URL for the dialog's preview and for draft widgets in edit mode: a favourite's `iconImageId` when it is known, otherwise `POST /api/favicons` through React Query, keyed by origin, without retries, and treating 404 and `blocked_address` as "none". The dialog passes the URL debounced by 500 ms. In view mode, the widget uses `data.iconImageId`.

### Icon chooser
The settings dialog shows one "Icon" field: a button with the current icon and the name of its kind (for a symbol, the symbol's name), with a chevron. It opens a Base UI `Popover` (inside the settings dialog) with Base UI `Tabs` for the four kinds, starting on the current kind, and a "Done" button:
- **Website icon**: the favicon found for the URL at a larger size, or a globe with "No website icon found for <host>", "Enter a URL first" or "Looking for the website icon…". A "Use the website icon" button.
- **Own image**: the current own image, if any, and an "Upload image" button (a file input with `accept` for the allowed types, posting to `/api/images` through `api()` with its `file` option, which sends `Content-Type: application/octet-stream`). A successful upload makes it the icon. `unsupported_image` and `image_too_large` are shown translated, and the icon stays as it was.
- **Symbol**: a filter field over the translated names and a grid of tiles. The tiles are native radio buttons (one group), so arrow keys move between them and screen readers read their names. Choosing one makes it the icon.
- **None**: a globe and a "Show a globe" button.

Choices take effect right away, but the popover stays open until "Done", Escape or a click outside, because arrow keys in the symbol grid change the choice as they move. The trigger's accessible name is "Icon: <kind or symbol name>".

The curated list has 44 lucide icons (Mail, Calendar, Cloud, Server, Database, Film, Music, Home, Settings, …) with translated names. lucide 1.x has no brand icons, so there is no GitHub or YouTube icon. It lives in `widgets/link/icons.ts` and is mirrored as an allowed set in `internal/widgets/link.go`.

*Alternatives*: separate image and icon fields with a precedence rule (as first built), which was confusing; the four kinds as radio buttons inline in the dialog, which made the dialog long.

### Rendering
The whole widget is one `<a>` (`rel="noopener noreferrer"` for `_blank`) with `min-w-0`, so `truncate` works. The icon box is fixed by size, followed by a column of `truncate` lines (title `font-medium`, URL `text-muted` one step smaller) and `title=` for the tooltip. In edit mode the anchor gets `onClick` → `preventDefault` and `aria-disabled`, so drag and the settings button stay usable.

### New favourite in its own dialog
The two sources are a radio group (like the height choice in the shared size fields), not tabs. The "Favourite" choice uses `FavouritePicker` in link mode (see the favourite explorer's design). Below it, a "New favourite…" button opens the favourites page's `NodeDialog` for a new link, nested on top of the settings dialog. That dialog gets two optional props:
- `chooseFolder`: it shows a `FavouritePicker` in folder mode, starting at the given parent (the root), instead of the fixed "In: …" folder line.
- `onSaved(node)`: it is called with the created node.

Validation, tags, errors and the favourites query refresh stay the dialog's own. The link settings choose the returned node. Until the refetched tree contains it, they keep the node in local state, so the dialog can be confirmed right away. Creating the favourite isn't tied to confirming the widget, so it stays when the widget or the edit is cancelled.

*Alternative*: fields for a new favourite inside the settings dialog, as first built. That made the settings dialog long, duplicated the favourites form without tags, and mixed two kinds of saving in one dialog.

## Risks / Trade-offs

- [The stored URL copy goes stale between saves] → It's only used after deletion, and `Resolve` serves live values until then.
- [Orphaned favourites when the edit is cancelled] → Intended (agreed). They are ordinary favourites the user can delete.
- [Upload during a draft that's never saved] → Covered by the 24 h cleanup in `image-store`.
- [A favicon lookup per link widget on every dashboard view] → One indexed query per origin, cached within the lookup; links to the same site share it.
