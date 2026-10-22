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
config: { "rootFolderId": "<uuid>", "title": "…" | null, "size": "S" | "M" | "L", "showIfPublic": false }
data:   { "missing": true }
     |  { "children": [ { id, kind: "folder", title, children: [...] }
                      | { id, kind: "link", title, url, iconImageId } ] }
```
`Normalize` checks the size and that `rootFolderId` is a non-empty string of at most 64 characters, trims the title (at most 200 characters, empty or missing → `null`, shared with the link widget), and it fills in a missing `showIfPublic` as `false`. It doesn't require the folder to exist, so a layout with a deleted folder can still be saved. A folder ID of another user is never resolved, because `Resolve` only looks in the owner's tree, so it behaves like a deleted folder.

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

### Title
With a title, the widget has a header line: the title as a heading (`truncate font-medium`) and the expand-all button at its right; the tree takes the full width below. Without a title, the button sits in a narrow column beside the tree. The heading level comes from a new `headingLevel` prop of every widget view: `SectionGrid` passes `min(level + 1, 6)`, the level a section's title has in the same list.

### Collapse state
The key is `famodash.explorer.<widgetId>`, holding a JSON array of expanded folder IDs. It is read once on mount and written on change, inside try/catch, so a blocked storage falls back to everything collapsed. Unknown IDs are ignored on read. The expand-all/collapse-all toggle is a small icon button (`ChevronsUpDown` / `ChevronsDownUp`) at the top right of the widget, in a narrow column beside the tree, so it is level with the first row and never takes a row of its own. (Floating it doesn't work: the full-width rows can't wrap around a float and move below it.) It is shown only when the tree has at least one folder.

### Sizes
| Size | Icon | Text | Row padding | Indent |
|---|---|---|---|---|
| S | 14 px | `text-xs` | `py-0.5` | 12 px |
| M | 16 px | `text-sm` | `py-1` | 16 px |
| L | 20 px | `text-base` | `py-1.5` | 20 px |

These are kept in one `sizeStyles` map in `widgets/sizes.ts`, so the link widget uses the same scale.

### Tree semantics
It's rendered as nested lists, like the tree on `/favourites`: a folder is a button with `aria-expanded` (chevron, folder icon, name), and a link is a link with its favicon and name. `/favourites` has no ARIA `tree` or arrow-key navigation to reuse, and links inside `treeitem`s are poorly supported by screen readers, so the explorer keeps to buttons and links in the Tab order. In edit mode links are plain text, so clicking them doesn't leave the editor. `label()` from `favourites/tree.ts` names the rows.

### Favourite picker
Both widgets choose from the favourites with one shared component, `FavouritePicker` in `features/favourites/`. It loads the favourites itself (the cached `favouritesQueryOptions`) and has two modes:
- `folder`: only folders, starting with the root "Favourites"; every folder can be chosen.
- `link`: links can be chosen, and folders only open and close. Folders without any link inside are left out.

It shows the favourites as a tree with folder icons and favicons, in a box that scrolls on its own (`max-h-60`). Folders with subfolders or links can be expanded and collapsed. The folders above the current choice start expanded. A search field above the tree filters by name and URL, ignoring case, and then shows a flat list of the matches, each with its folder's path. Below the box, "Selected: Favourites / Work / Tools" names the choice, even when it is scrolled away or collapsed.

The options are native radio buttons in a `radiogroup` labelled by the field's label, so arrow keys and screen readers work without extra code. The chosen row is highlighted, and the focused row gets a focus ring. A value that isn't in the tree (deleted) selects nothing, and the caller shows its own message.

*Alternative*: a native `<select>` with paths, as first built. It works, but a long flat list of paths is hard to scan, it can't show icons, and it can't be searched.

## Risks / Trade-offs

- [Large subtree in the lookup payload] → Accepted for now, since a personal bookmark tree is small. A depth or node cap can follow if needed.
- [localStorage grows with removed widgets] → Each entry is a short list. Acceptable.
- [Edit-mode preview differs from the visitor view] → Only when the flag is off, which is the intended difference. The owner sees the same tree in both modes.
