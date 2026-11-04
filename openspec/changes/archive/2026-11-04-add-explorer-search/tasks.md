# Tasks

## 1. Shared pieces

- [x] 1.1 Move `Highlighted` out of `FavouritesPage.tsx` into `frontend/src/features/favourites/Highlighted.tsx` and import it from there; verify the favourites page's existing search highlight tests still pass
- [x] 1.2 Add the i18n keys `search`, `searchLabel`, `closeSearch` and `noMatches` under `dashboards.widgets.favouriteExplorer` in `en.json` ("Search", "Search this folder", "Close search", "Nothing matches your search.") and `de.json` ("Suchen", "Diesen Ordner durchsuchen", "Suche schließen", "Nichts passt zu deiner Suche."); verify the i18n key parity test passes

## 2. Explorer search

- [x] 2.1 In `ExplorerTree.tsx`, add `open` and `query` state and a search button (Search icon, `iconButton`, `aria-expanded`, `aria-controls`) placed after Add and before toggle-all, shown only when the folder has children; verify with a test of the button order Add → Search → Expand all and its absence for empty or missing folders
- [x] 2.2 Render the search bar (input with `searchLabel`, close button) directly above the tree, with and without a title; focus the input when it opens, and close and clear it on Escape (with `preventDefault`), on × and when the button is activated again, returning focus to the button; verify with tests for opening, focus, Escape, ×, the toggle, and that the field is closed after a remount
- [x] 2.3 Filter with `searchTerms` + `filterTree` on a synthetic root (memoised); while terms are active, render every shown folder as expanded without calling `setExpanded`, highlight matching names with `Highlighted`, show `noMatches` when nothing matches, and hide the toggle-all button; verify with tests for title, multi-term, folder-name, URL and tag matches, no matches, whitespace-only queries, the hidden toggle-all, and that the stored expanded set (localStorage) is the same after searching and clearing
- [x] 2.4 Verify the search works in edit mode and for visitors with tests that render the explorer with `editing` and without owner rights

## 3. Specs, docs and checks

- [x] 3.1 Confirm that export/import (`backend/internal/export`, `data-import`) and the OpenAPI documents need no change, since no data, config or route changes; note this when archiving
- [x] 3.2 Run `make check` and `make test` and verify both pass
- [x] 3.3 Run `openspec validate add-explorer-search --strict` and verify it passes
