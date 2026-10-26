# Proposal

## Why

With a growing favourites tree, `/favourites` is hard to work with: there is no way to find a link by name, URL or tag, every folder starts expanded on each visit, and there is no quick way to collapse the whole tree. A favourite explorer on a dashboard also offers no way to get to that folder on `/favourites` to manage it (issue #40).

This is the first of three changes for issue #40. Bulk edit (`favourites-bulk-edit`) and tag management (`favourites-tag-management`) follow separately.

## What Changes

- `/favourites` gets a search field that filters the tree by title, URL and tag names, and a tag filter that narrows it to links with all chosen tags. Matching links are shown with their folders, which are expanded. The matched text is highlighted.
- The search text and chosen tags are kept in the URL (`?q=`, `?tags=`), so they survive a reload and the back button.
- Clicking a tag on a row adds it to the tag filter. `/` focuses the search field.
- While a filter is active, nodes can't be moved by drag and drop or keyboard. A hint says to clear the filters to reorder.
- `/favourites` gets an "Expand all" / "Collapse all" button, which works like the one in the explorer widget.
- The folders the user collapses on `/favourites` are remembered in the browser. Folders still start expanded on a first visit.
- Folder rows show how many links they contain at all levels.
- `/favourites?folder=<id>` expands that folder and its ancestors, scrolls it into view and highlights it.
- For the owner, outside edit mode, the title of a favourite explorer links to `/favourites?folder=<root folder>`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: the favourites page gains search, a tag filter, expand/collapse all, remembered collapse state, folder link counts and a folder deep link. Drag and drop is turned off while filtering.
- `favourite-explorer-widget`: the explorer title links to its folder on `/favourites` for the owner.

## Impact

- Frontend only: `features/favourites` (`FavouritesPage.tsx`, `tree.ts` with new filter and count helpers), `routes/favourites.tsx` (search params), `features/dashboards/widgets/favouriteExplorer/ExplorerTree.tsx` and `index.tsx` (title link), locales (en, de), and their tests.
- No backend, API or migration changes: the page already loads the whole tree, with tags, in one request.
