# Tasks

## 1. Details footer

- [x] 1.1 In `FavouriteDetails.tsx`, make the "Offline copy" button a `secondaryIconButton` with only the `Archive` icon, `aria-label` and `title` set to `favourites.offline.button`, and add `flex-wrap` to the footer row; verify with tests of the button order by accessible name, the missing visible text and the tooltip

## 2. Checks

- [x] 2.1 Confirm that export/import and the OpenAPI documents need no change, since no data or route changes
- [x] 2.2 Run `make check` and the favourites tests and verify they pass
- [x] 2.3 Run `openspec validate compact-details-actions --strict` and verify it passes
