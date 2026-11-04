# Proposal

## Why

A favourite explorer with many nested folders makes a link slow to find: you have to expand folders one by one. The favourites page already has a live search, but the widget has none, so finding a link on a dashboard means leaving it.

## What Changes

- A new search button ("Search" / "Suchen") in the favourite explorer, between "Add favourite" and "Expand all" / "Collapse all". Everyone who sees the explorer gets it: the owner, visitors on public dashboards, and the owner in edit mode.
- Activating the button opens a small search bar above the tree and focuses it. Activating it again, pressing Escape or using the bar's clear/close button closes the bar and clears the search.
- Typing filters the tree live in the browser, with the same rules as the favourites page: a link matches when every term occurs in its title, URL, a tag or a folder name on its path, and a folder whose name matches is shown with everything inside it. Matching text is highlighted.
- While a search is active, every shown folder is expanded. The folders the viewer has expanded, which the browser remembers, are not changed, and clearing the search brings that tree back.
- While a search is active, the "Expand all" / "Collapse all" button is hidden.
- When nothing matches, the explorer says "Nothing matches your search." / "Nichts passt zu deiner Suche."
- The search text is not stored anywhere and is gone after a reload.
- "Add favourite" now sits left of the search button rather than directly left of the expand-all button.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourite-explorer-widget`: new requirement "Explorer search". "Collapse state" changes because the expand-all button is hidden while searching. "Add a favourite from the explorer" changes because the button order is now Add, Search, Expand all.

## Impact

- Frontend only: `frontend/src/features/dashboards/widgets/favouriteExplorer/ExplorerTree.tsx` (search button, bar, filtered rendering) and its tests. It reuses `searchTerms`, `filterTree` and `highlight` from `frontend/src/features/favourites/tree.ts`, plus new en/de i18n keys under `dashboards.widgets.favouriteExplorer`.
- Backend, API and OpenAPI documents are unaffected. The widget data already contains titles, URLs and tags, also for visitors.
- Export and import are unaffected: no user data, widget config or stored state changes.
