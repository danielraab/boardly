# Design

## Context

- Content widget kinds are registered in Go in `internal/widgets` (`Normalize`, `Resolve`) through `widgets.Register`, and in the web app in `widgets/registry.ts` (`icon`, labels, `defaultConfig`, `View`, `Settings`). Resolved `data` comes with the lookup and is stripped on save. Width and height are handled by `WidgetSettingsForm` for every kind.
- The favourite explorer already does most of what's needed: it picks a folder with `FavouritePicker` in folder mode, loads it with `favourites.Store.Subtree` plus `images.Favicons.SetIcons` through the per-lookup cache `dashboards.Cached`, leaves itself out for visitors without `showIfPublic`, and in edit mode builds its data from `favouritesQueryOptions`.
- `LinkView` renders one link with icon, title and URL, scaled by `sizeStyles`. `ImageIcon` renders a favicon with a globe fallback.

## Goals / Non-Goals

**Goals:**
- Visitors receive only the links the widget shows, never folder names or hidden subfolder contents.
- Owner view, visitor view and edit-mode preview show the same links for the same favourites.

**Non-Goals:**
- Grouping links under subfolder headings, sorting other than favourites order, tag filters.
- A link target setting, or per-item icons other than the favicon.
- Sharing a single component with the explorer for the view.

## Decisions

### A separate kind, not an explorer mode
`link_list` is its own kind with its own config. Subfolder flattening and the layout choice mean nothing for a tree, and collapse state means nothing for a list, so a mode flag would leave half the settings unused in each mode. The shared parts (folder picker, title, size, subtree lookup) are already helpers.

### Config and data
```json
config: { "folderId": "<uuid>", "title": "…" | null, "includeSubfolders": false,
          "layout": "list" | "compact" | "tiles", "size": "S" | "M" | "L", "showIfPublic": false }
data:   { "missing": true }
     |  { "links": [ { id, title, url, iconImageId } ] }
```
`Normalize` decodes strictly, checks the size and layout, checks that `folderId` is a non-empty string of at most 64 characters, and trims the title with the shared `normalizeTitle`. Missing booleans decode as `false`. Like the explorer, it doesn't require the folder to exist.

### Flatten on the server
`Resolve` loads the subtree the same way the explorer does, then flattens it: a depth-first walk that emits each link and descends into folders only when `includeSubfolders` is set. The result contains only link fields (`id`, `title`, `url`, `iconImageId`), so a visitor's payload holds no folder names and nothing from skipped subfolders.

The subtree loading (lookup by owner, not-found as missing, `SetIcons`, `Cached` by `subtreeKey`) moves from the explorer into a small shared helper in `internal/widgets`, so an explorer and a link list on the same folder share one query per lookup.

*Alternative*: send the subtree and flatten in the browser. It's less Go code, but it would send hidden subfolders to visitors.

### Edit-mode preview
As in the explorer, the view in edit mode ignores `data` and builds the links from `favouritesQueryOptions`: `findNode` for the folder, then a TypeScript `flattenLinks(folder, includeSubfolders)` with the same order as the Go walk. A non-folder or missing node gives `{ missing: true }`. View mode uses `data`.

### Layouts
All three render a `<ul>` of `<a target="_blank" rel="noopener noreferrer">` items with `title` set to "title\nURL" for the tooltip. In edit mode, a click is prevented and `aria-disabled` is set, as in `LinkView`.
- **List**: the row look of `LinkView` (icon, title, smaller URL line). `LinkView` is reused if it fits as is; otherwise its markup is extracted into a shared row component.
- **Compact**: the same row with only the title, or the URL when there's no title.
- **Tiles**: a CSS grid with `grid-template-columns: repeat(auto-fill, minmax(<min>, 1fr))`, each tile a column of a large icon and a one-line truncated label. Being grid-based, it wraps to the widget's width without measuring.

### Sizes
Rows use the existing `sizeStyles` (icon, text, row padding) and the URL line scale of `LinkView`. Tiles get their own small map in the link list module:

| Size | Tile icon | Min tile width | Label |
|---|---|---|---|
| S | 24 px | 4.5rem | `text-xs` |
| M | 32 px | 6rem | `text-xs` |
| L | 40 px | 7.5rem | `text-sm` |

### Title
Same as the explorer: a `truncate font-medium` heading of `headingLevel` above the items when there's a title, and nothing otherwise.

### Messages
"No links in this folder" (rather than the explorer's "This folder is empty") because a folder with only subfolders isn't empty, but shows nothing when the subfolder flag is off. "This folder no longer exists" reuses the explorer's wording under its own key.

## Risks / Trade-offs

- [Root folder with subfolders on lists every bookmark] → It's what the user asked for. The widget's fixed height scrolls, and the payload is the same size as an explorer on the root.
- [Go and TypeScript flatten could diverge] → Both are a few lines of depth-first walk, and each has a test for the same nested example from the spec.
- [Explorer refactor for the shared subtree helper] → Behaviour stays the same and is covered by the explorer's existing lookup tests.
