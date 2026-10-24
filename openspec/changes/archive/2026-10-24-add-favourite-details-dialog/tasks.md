# Tasks

## 1. Shared dialog and button

- [x] 1.1 Add en/de translations under `favourites.details` (`button` with `{{title}}`, `open`, `edit`, `noDescription`, `close`, `tags`); verify the i18n key parity test passes
- [x] 1.2 Add `DetailsButton` in `features/favourites/` (lucide `Info`, `iconButton`, accessible name from `favourites.details.button` with the title or else the URL, optional smaller icon class); verify with a test that it renders the accessible name and calls `onClick`
- [x] 1.3 Add `FavouriteDetailsDialog({ favouriteId, onClose })` reading the node via `findNode` from `favouritesQueryOptions`: title or URL as heading, preview only when present, favicon and URL as a new-tab link, description or muted "No description", tags only when present, close button, "Open" (new-tab `<a>`) and "Edit"; verify with tests for full details, no description, no preview and no tags, URL and Open targets, and Escape closing
- [x] 1.4 Make "Edit" swap to `NodeDialog` with `target={{ node }}`, show the details again after cancel or save, and invalidate `dashboardViewQueryKey` in `onSaved`; verify with tests that saving a new description shows it in the details and cancelling keeps the old values

## 2. Link widget

- [x] 2.1 Add optional `onDetails` to `LinkView`, rendering `DetailsButton` as a sibling of the `<a>` (row becomes flex, `<a>` `min-w-0 flex-1`); verify that existing `Link.test.tsx` tests still pass without `onDetails`
- [x] 2.2 In the link widget `View`, enable the favourites query for `isOwner`, pass `onDetails` only when owner, not editing, source favourite, and the node exists as a link; render the dialog; verify tests: owner sees the button and opens details without navigating, no button for visitor, edit mode, URL source and deleted favourite

## 3. Favourite explorer

- [x] 3.1 Add optional `onDetails(id)` to `ExplorerTree`, rendering a smaller, muted, always-visible `DetailsButton` at the right end of each link row as a sibling of the `<a>`, none on folders; pass it from `View` for the owner outside edit mode and render the dialog; verify tests in `FavouriteExplorer.test.tsx`: button on link rows only, opens details, absent for visitor and in edit mode

## 4. Link list

- [x] 4.1 Add optional `onDetails(id)` to `LinkListView`: beside the row in "List" (via `LinkView`) and "Compact", top-right corner overlay in "Tiles" (`relative` `<li>`, absolute sibling button); verify with tests per layout that the button exists and opens details without following the link
- [x] 4.2 In the link list `View`, enable the favourites query for `isOwner`, pass `onDetails` for the owner outside edit mode once the tree has loaded, and render the dialog; verify tests: absent for visitor and in edit mode

## 5. Verification

- [x] 5.1 Run `pnpm check`, typecheck and `pnpm test` in `frontend/` and verify they pass (`make check` and `make test`)
- [x] 5.2 Run the app and check the info buttons and the dialog in all three widgets and all link-list layouts, at sizes S and L, in light and dark themes, signed in as owner and as a visitor of a public dashboard
