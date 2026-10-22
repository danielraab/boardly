# Design

## Context

- After `widget-model`, a kind is registered in Go (`Normalize`, `Resolve`) and in the web app (`icon`, labels, `defaults`, `View`, `SettingsDialog`). Resolved `data` comes with the lookup and is stripped on save.
- After `image-store`, favourite link nodes have `iconImageId` (resolved by origin), and `ImageIcon` renders an image with a globe fallback.
- The owner can always read `/api/favourites`. Visitors can't.

## Goals / Non-Goals

**Goals:**
- Visitors receive exactly the subtree they're allowed to see, and nothing when the flag is off.
- The draft preview in edit mode works for a widget that hasn't been saved yet.

**Non-Goals:**
- Editing favourites from inside the widget (drag, rename, add).
- Tag filters or search.
- Syncing the collapse state across devices.

## Decisions

### Config and data
```json
config: { "rootFolderId": "<uuid>", "size": "S" | "M" | "L", "showIfPublic": false }
data:   { "missing": true }
     |  { "children": [ { id, kind: "folder", title, children: [...] }
                      | { id, kind: "link", title, url, iconImageId } ] }
```
`Normalize` checks the size and that `rootFolderId` is a non-empty string of at most 64 characters, and it fills in a missing `showIfPublic` as `false`. It doesn't require the folder to exist, so a layout with a deleted folder can still be saved. A folder ID of another user is never resolved, because `Resolve` only looks in the owner's tree, so it behaves like a deleted folder.

### Resolve
`Resolve(owner, viewerIsOwner, config)`:
- visitor and `!showIfPublic` → `ok=false` (left out)
- folder missing, or not a folder of the owner → owner: `{missing:true}`, visitor: `ok=false`
- otherwise, the subtree from `favourites.Store`. There's a new `Subtree(ctx, userID, folderID)` with one recursive CTE query, plus the batched favicon lookup from `image-store`.

Tags and timestamps are left out of `data`, because visitors shouldn't see them.

Several explorers on one dashboard each run one query, which is acceptable at the 500-widget cap. A per-lookup cache by folder ID avoids repeats.

The kind lives in a new package `internal/widgets` (`FavouriteExplorer(fav, icons) dashboards.Kind`), and `widgets.Register` adds it to the dashboard store's `Kinds` in `main.go`, so `dashboards` doesn't import `favourites` or `images`. The per-lookup cache is `dashboards.Cached(ctx, key, load)`, backed by a map that `Lookup` and `SaveLayout` put into the context for one resolve. Setting favicons on a tree moved from `httpapi` to `images.Favicons.SetIcons`, so the favourites API and the widget share it.

### Draft preview from the favourites query
A widget added in the draft has no `data` until it is saved, and changing the root folder makes the old `data` stale. In edit mode the view therefore builds its tree from `favouritesQueryOptions` (the owner's full tree, already cached by the folder picker) and ignores `data`. In view mode it uses `data`. Both paths feed the same `ExplorerTree` component.

### Collapse state
The key is `famodash.explorer.<widgetId>`, holding a JSON array of expanded folder IDs. It is read once on mount and written on change, inside try/catch, so a blocked storage falls back to everything collapsed. Unknown IDs are ignored on read. The expand-all/collapse-all toggle is a small icon button (`ChevronsUpDown` / `ChevronsDownUp`) floated to the top right of the widget, so only the first rows make room for it. It is shown only when the tree has at least one folder.

### Sizes
| Size | Icon | Text | Row padding | Indent |
|---|---|---|---|---|
| S | 14 px | `text-xs` | `py-0.5` | 12 px |
| M | 16 px | `text-sm` | `py-1` | 16 px |
| L | 20 px | `text-base` | `py-1.5` | 20 px |

These are kept in one `sizeStyles` map in `widgets/sizes.ts`, so the link widget uses the same scale.

### Tree semantics
It's rendered as nested lists, like the tree on `/favourites`: a folder is a button with `aria-expanded` (chevron, folder icon, name), and a link is a link with its favicon and name. `/favourites` has no ARIA `tree` or arrow-key navigation to reuse, and links inside `treeitem`s are poorly supported by screen readers, so the explorer keeps to buttons and links in the Tab order. In edit mode links are plain text, so clicking them doesn't leave the editor. `label()` from `favourites/tree.ts` names the rows.

### Folder picker
The settings dialog uses a native `<select>`, like every other select in the app, over a flat list of all folders in tree order. Each option is labelled with its path ("Favourites / Work / Tools"), which also shows its depth. A deleted folder isn't an option, so the select shows "Choose a folder" and the dialog can't be applied until another one is picked.

## Risks / Trade-offs

- [Large subtree in the lookup payload] → Accepted for now, since a personal bookmark tree is small. A depth or node cap can follow if needed.
- [localStorage grows with removed widgets] → Each entry is a short list. Acceptable.
- [Edit-mode preview differs from the visitor view] → Only when the flag is off, which is the intended difference. The owner sees the same tree in both modes.
