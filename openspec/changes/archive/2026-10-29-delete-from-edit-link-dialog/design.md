# Design

## Context

`NodeDialog` (`frontend/src/features/favourites/NodeDialog.tsx`) creates and edits favourites. It is opened from `FavouritesPage` (edit from a tree row) and from `FavouriteDetailsDialog` (`FavouriteDetails.tsx`). The dashboard widgets link, link list, recent favourites and favourite explorer all use that details dialog and pass `onSaved` to refresh `dashboardViewQueryKey`.

`DeleteDialog` (`DeleteDialog.tsx`) is the confirmation used from the tree. It is an `AlertDialog` that deletes the node on confirm, invalidates the favourites, tags and app-token queries, shows a generic error on failure, and calls `onClose`. For a link it ignores `parentLabel`.

`FavouriteDetailsDialog` is open while `link !== undefined && !editing`. If the edit dialog just closed after a delete, the details dialog would show up again with the stale link until the favourites query refetches.

## Goals / Non-Goals

**Goals:**
- Reuse `DeleteDialog` unchanged for the confirmation, so the text, error handling and query invalidation stay the same as in the tree.
- Callers of `NodeDialog` and `FavouriteDetailsDialog` keep their current props.

**Non-Goals:**
- Deleting folders from the edit-folder dialog.
- Undo, as bulk delete has.

## Decisions

- **Delete button in `NodeForm`, shown only when `editing?.kind === "link"`.** It sits on the left of the button row (`justify-between`, with Cancel and Save grouped on the right), styled as a secondary button with a `Trash2` icon so it isn't confused with Save. The label is the new key `favourites.dialog.delete` ("Delete" / "Löschen").
- **`NodeForm` renders `DeleteDialog`** with `target={{ node: editing, parentLabel: "" }}` while a local `confirmDelete` state is set. Base UI nests the alert dialog over the open dialog, so the edit form stays mounted and keeps its unsaved state when the confirmation is cancelled.
  - *Alternative:* lift the delete state to every caller, as `FavouritesPage` does for tree rows. Rejected because it touches every caller for no gain.
- **Telling the caller that the delete succeeded:** `DeleteDialog` gets an optional `onDeleted` callback, called from its mutation's `onSuccess` before `onClose`. `NodeDialog` gets an optional `onDeleted` too. `NodeForm` passes a handler that calls `onDeleted?.()` and then `onClose()`, which closes the edit dialog. `FavouritesPage` needs no prop, because the favourites invalidation already refreshes the tree.
- **`FavouriteDetailsDialog` handles `onDeleted`** by calling `onSaved?.()` (each widget passes the dashboard refresh there) and `onClose()`. `onClose` clears the widget's `detailsId`, so the details dialog can't briefly come back with the stale link. Reusing `onSaved` means "favourite changed, refresh", so the four widgets need no change.
  - *Alternative:* a separate `onDeleted` prop on `FavouriteDetailsDialog`. Rejected because every caller would pass the same refresh again.

## Risks / Trade-offs

- [Nested dialog focus: after cancelling the confirmation, focus has to return into the edit dialog] → Base UI returns focus to the trigger (the Delete button). A test checks that the edit dialog is still open with its values.
- [`onSaved` now also fires after a delete] → Every current caller only uses it to refresh, so no caller misbehaves. The prop's doc comment will say so.
