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
  "imageId": "<image id>" | null,
  "icon": "mail" | null,            // key from the curated list
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
For `source: "favourite"`, `data = { url, title }` with the favourite's current values, or the copies when it's missing. For `source: "url"`, `data` is empty. The view uses `data` over `config` in view mode. In edit mode it uses the favourites query, as the explorer does, so a draft widget renders right away. Visitors get only these fields: no tags and no favourite ID beyond what is in `config`.

Should `config.favouriteId` reach visitors? It's an opaque UUID that grants nothing, because favourites need the owner's session. It's left in, to keep one config shape.

### Favicon pre-fill
When the dialog's URL changes (debounced 500 ms, only for a valid URL) and the image is empty, or still holds the favicon that was filled automatically for the previous URL, the dialog calls `POST /api/favicons`. For a favourite with `iconImageId`, that ID is used directly. A 404 or `blocked_address` leaves the image empty with no error. The dialog tracks "auto-filled" in local state only, and stores just `imageId`. When a new URL has no favicon, an auto-filled image from the previous URL is cleared. Opening the settings of an existing widget doesn't fill anything until its URL changes, so an image the owner removed stays removed.

A favourite's URL can later change to another origin while the widget keeps the old favicon. That's accepted: the owner set or accepted that image. Refreshes keep the image ID, so the icon stays current for the same origin.

### Image field UI
The field shows a preview (`ImageIcon`), an "Upload" button (a file input with `accept` for the allowed types, posting to `/api/images`, with errors `unsupported_image`/`image_too_large` shown translated) and a "Remove" button. Next to it, an icon select over the curated list (44 common lucide icons: Mail, Calendar, Cloud, Server, Database, Film, Music, Home, Settings, …), with "None" and translated names. lucide 1.x has no brand icons, so there is no GitHub or YouTube icon. The list lives in `widgets/link/icons.ts` and is mirrored as an allowed set in `internal/widgets/link.go`. The upload sends the file through `api()` with a new `file` option, which sets `Content-Type: application/octet-stream`.

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
