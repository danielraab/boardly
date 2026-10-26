# Proposal

## Why

With many nested folders, the tree on `/favourites` breaks: the indentation grows with every level while the move handle, toggle, icon, link count and action buttons keep their width, so the names shrink to nothing and, from about six levels on a phone, the rows overflow the tree and the whole page scrolls sideways with buttons overlapping each other. The favourite explorer widget has the same structure and breaks the same way in a narrow widget.

## What Changes

- Names in the favourites tree keep a minimum width and are cut off with "…" instead of disappearing, and have a maximum width of 24rem: longer names end with "…" there, and the full name shows as a tooltip.
- When the rows don't fit, the tree gets its own horizontal scrollbar; the page itself never scrolls sideways.
- The action buttons of a row come after its name and link count and never cover them. Rows that fit keep their buttons at the right edge of the tree; deeper rows are wider than the tree, and their buttons are reached by scrolling sideways.
- The indentation per level is halved on narrow screens (below the `sm` breakpoint), so more levels fit before scrolling is needed. The drop indicator of drag and drop uses the same indentation.
- The favourite explorer widget gets the same minimum and maximum name width and horizontal scrolling.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: the favourites page keeps names readable at any depth with minimum widths, a horizontally scrolling tree, buttons that never overlap names, and a smaller indentation on narrow screens.
- `favourite-explorer-widget`: the explorer tree keeps a minimum name width and scrolls horizontally instead of hiding names.

## Impact

- Frontend only: `features/favourites/FavouritesPage.tsx` (tree container, row layout, indentation variable, drop indicator), `features/dashboards/widgets/favouriteExplorer/ExplorerTree.tsx`, and their tests.
- No API, backend or translation changes.
