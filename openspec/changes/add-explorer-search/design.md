# Design

## Context

`ExplorerTree` (`frontend/src/features/dashboards/widgets/favouriteExplorer/ExplorerTree.tsx`) draws the tree from `data.children` and reads the expanded folders from `useExpanded(widgetId)`, which saves them in localStorage. Its header has the optional "Add favourite" button (`onAdd`) and the expand-all toggle. With a title, they share the title's line; without one, they sit beside the tree.

The favourites page's search is in `frontend/src/features/favourites/tree.ts`: `searchTerms`, `filterTree` (with a `TreeFilter` of terms, tags and domain) and `highlight`. `FavouritesPage.tsx` renders the highlighted parts with a private `Highlighted` component (`<mark className="rounded-sm bg-accent/25 text-inherit">`).

The widget data the server sends, also to visitors, contains complete `favourites.Node`s (title, URL, tags, children), so filtering in the browser has everything it needs. Requirements: see `specs/favourite-explorer-widget/spec.md`.

## Goals / Non-Goals

**Goals:**
- Use the favourites page's matching for the widget, so it doesn't drift from the page.
- Keep the stored expanded state unchanged while searching.

**Non-Goals:**
- No tag or domain filter chips in the widget. It searches text only.
- No search across several widgets, and no keyboard shortcut to open the search.
- No server-side search.

## Decisions

### Filter with `filterTree` on a synthetic root
`filterTree` takes a root node. The explorer passes `{ id: "", kind: "folder", children }` with `{ terms, tags: [] }` and renders the returned children, or the no-match message when it returns `null`. Folder paths then start below the explorer's root folder, which is what the spec asks for. The search only uses text the explorer already shows. Alternative: a separate widget filter. Rejected because the two searches would drift apart.

### Expanding without touching storage
During a search, rendering ignores `expanded` and treats every folder as open. A filtered folder only contains matches, so "all open" shows exactly the matches. Toggling a folder row during a search is turned off (the chevron shows open, `aria-expanded="true"`, and the click does nothing), so `setExpanded` is never called. When the search is cleared, rendering goes back to `expanded`. Alternatives:
- Keep a temporary expanded set for the search, so folders can be collapsed while searching. It's more state for little gain, and it can be added later.
- Write the matches into the stored set. That breaks "clearing the search brings back the tree", so it's rejected.

### Search state lives in `ExplorerTree`
`open: boolean` and `query: string` are `useState` in `ExplorerTree`, not in storage or the URL. The spec says the search text isn't stored. Keeping the state here leaves `index.tsx` (View, Settings, config) untouched, and the button shows in edit mode and for visitors without any new props.

### Share `Highlighted`
Move `Highlighted` from `FavouritesPage.tsx` into a small shared component, for example `frontend/src/features/favourites/Highlighted.tsx`, and use it in both places. Matches then look the same on the page and in the widget. The folder name keeps `title={name}` for its tooltip.

### Search bar
Below the title's line, or at the top of the tree's column beside the buttons when there is no title, and above the tree, the bar is a `type="search"` input with a `Search` icon and an `X` close button, sized to the widget's `size` text. It's a solid `bg-paper` strip with a `border-line` divider below it, so it stands apart from the tree by colour and line rather than by spacing. When the bar opens, focus moves to the input. Escape inside the input closes the bar, calling `preventDefault` so the dashboard or a dialog doesn't react to it as well. When the bar closes because of Escape or ×, focus goes back to the search button. The search button gets `aria-expanded` and `aria-controls` pointing to the bar.

### Button order and visibility
Header buttons, left to right: Add (`onAdd`), Search (when the folder has children), toggle-all (when there are folders and no active terms). "Active terms" means `searchTerms(query).length > 0`, so a query of only spaces counts as no search, as the spec says.

## Risks / Trade-offs

- [Large folders re-filter on every keystroke] → `filterTree` is linear and the trees are small, so wrap it in `useMemo` keyed on the children and terms. No debounce is needed.
- [Very narrow widgets (size S, one column) and a header with three buttons] → The bar sits in the tree's column, never in the button row, and its input has `min-w-0`, so the buttons never wrap into it.
- [The native clear button of `type="search"` would duplicate ×] → It's hidden with `[&::-webkit-search-cancel-button]:hidden`.
- [Folders can't be collapsed during a search] → It's intentional. The viewer can narrow the search or close it instead.
