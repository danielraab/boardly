# Proposal

## Why

The favourite explorer shows a folder as a tree, which suits browsing but not quick access. A dashboard also needs a plain list of the links in a folder, the way Homepage shows a group of bookmarks: every link visible at once, with no folders to open.

## What Changes

- New widget kind `link_list` ("Link list" / "Linkliste"), available in the "Add widget" picker.
- Settings: a folder from the user's favourites (required; the root "Favourites" is allowed), an optional title, "Show items from subfolders" / "Einträge aus Unterordnern zeigen" (default off), a layout ("List" / "Liste", the default; "Compact" / "Kompakt"; "Tiles" / "Kacheln"), a size (S, M, L; default M) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). Width and height work as for every widget.
- The widget shows the links of the folder as a flat list in favourites order. With "Show items from subfolders" on, the links of all subfolders at every depth are included where their folder is. Folders themselves are never shown.
- Layouts: "List" shows icon, title and URL like a link widget. "Compact" shows icon and title (the URL only when there's no title). "Tiles" shows a wrapping grid of tiles with a large icon above the title. Size scales the icons, text and spacing.
- Links always open in a new tab. In edit mode they don't navigate.
- Empty and deleted folders behave as in the favourite explorer. Visitors only get the widget when "Show on public dashboard" is on, and then only the folder's links.

## Capabilities

### New Capabilities

- `link-list-widget`: the link list widget's settings, layouts, sizes, flattening of subfolders and visibility.

### Modified Capabilities

None. The generic widget behaviour comes from `dashboard-layout`, and the folder picker and favicons are reused as they are.

## Impact

- Backend: a `link_list` kind in `internal/widgets` (config normalization, resolving the folder with `favourites.Store.Subtree` and favicons, flattening, visitor filtering), registered in `widgets.Register`, with tests. No migration, since kinds aren't checked in the database.
- Frontend: `widgets/linkList/` (view with three layouts, settings dialog with `FavouritePicker`), registry entry, en/de locales and tests. The row of the "List" layout reuses `LinkView`'s look.
- README: the widget table.
- No new dependencies.
