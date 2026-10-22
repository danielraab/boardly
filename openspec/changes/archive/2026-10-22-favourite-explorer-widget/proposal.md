# Proposal

## Why

Dashboards should show the user's bookmarks. The favourites tree is the single source of truth for them, so the first content widget shows part of that tree right on a dashboard, like a file explorer.

## What Changes

- New widget kind `favourite_explorer` ("Favourite explorer" / "Favoriten-Explorer"), available in the "Add widget" picker.
- Settings: a root folder from the user's favourites (required; the whole tree's root is allowed), a size (S, M, L; default M) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). Width and height work as for every widget.
- Adding the widget opens its settings first. The widget is only added once a folder is chosen and the dialog is confirmed.
- The widget shows the children of the root folder as a file tree. Folders have a folder icon and can be expanded and collapsed. Links show their favicon, or a globe when there is none, and open in a new tab. Size changes the icon size, the font size and the padding.
- Folders always start collapsed. Expanding and collapsing is remembered per widget in the browser. A small button expands or collapses all folders at once.
- When the root folder has been deleted, the widget shows "This folder no longer exists" to the owner.
- Visitors of a public dashboard see the widget only when "Show on public dashboard" is on. Otherwise the server leaves it out of their response, including the tree.
- The tree is always the current state of the favourites. The widget stores only the folder's ID.

## Capabilities

### New Capabilities

- `favourite-explorer-widget`: the favourite explorer widget's settings, rendering, collapse state and visibility.

### Modified Capabilities

None. The generic widget behaviour comes from `widget-model`, and favicons from `image-store`.

## Impact

- Depends on the changes `widget-model` and `image-store`.
- Backend: a `favourite_explorer` kind registered in `dashboards` (config normalization, resolving the subtree from `favourites` with favicons, visitor filtering) and its tests.
- Frontend: `widgets/favouriteExplorer/` (view, settings dialog with a folder picker, collapse state in `localStorage`), registry entry, en/de locales and tests.
- No new dependencies.
