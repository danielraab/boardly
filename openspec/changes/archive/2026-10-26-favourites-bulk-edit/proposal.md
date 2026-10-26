# Proposal

## Why

Cleaning up a favourites tree one row at a time is slow: tagging twenty links, moving a batch into a new folder or deleting a set of dead links each takes dozens of dialogs (issue #40). Together with search and the tag filter from `favourites-search-filter`, bulk actions let users find a group of links and change them in one step.

This is the second of three changes for issue #40. It builds on `favourites-search-filter`.

## What Changes

- `/favourites` gets a "Select" mode. Only in this mode, every row shows a checkbox. Checking or unchecking a folder checks or unchecks everything below it. Shift-click checks a range, and "Select all" checks all shown rows.
- A bar shows how many links and folders are selected and offers:
  - **Add tags**, **Remove tags** and **Set tags**, which act on the selected links.
  - **Move to…**, which moves the selection into a chosen folder. Fully selected folders move as a whole.
  - **Delete**, which deletes the selected links. Folders are never deleted in bulk.
- Bulk delete has no confirmation dialog. Instead an "Undo" toast is shown for a few seconds, and the delete is sent only when the toast is gone.
- A new endpoint `POST /api/favourites/bulk` applies one action to many nodes in one transaction.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: the favourites page gets selection mode and bulk tag, move and delete actions with undo, and the API gets a bulk endpoint.

## Impact

- Backend: `backend/internal/favourites` (bulk store method in one transaction, reusing the move, tag and cleanup logic), `backend/internal/httpapi` (new route and handler), Go tests.
- Frontend: `features/favourites` (`FavouritesPage.tsx` selection state and bar, new bulk dialogs, `api.ts`, `tree.ts` selection helpers), `FavouritePicker` (folders that can't be chosen), a small toast component, locales (en, de), and their tests.
- No migration.
