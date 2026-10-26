# Design

## Context

`favourites.Store` has `Move`, `Delete`, `Update` (with `setTags`) and `removeUnusedTags`, each opening its own transaction. `DetachTag` already removes one tag from many links. The page renders rows from `flatten(tree, collapsed)`. After `favourites-search-filter`, the shown tree may be a pruned copy, and `TreeRow` has a `movable` flag. `FavouritePicker` chooses a folder or link, but can't exclude folders. The frontend has no toast component yet.

## Goals / Non-Goals

**Goals:**
- One request and one transaction per bulk action.
- Selection logic as pure helpers in `tree.ts`, tested on their own.

**Non-Goals:**
- Bulk delete of folders, and undo for actions other than bulk delete.
- Server-side trash or restore.
- Selection in the explorer widget.

## Decisions

### Selection is a set of node IDs
The page stores `selected: Set<string>` over all nodes, folders included. Checking a folder adds it and all its shown descendants, and unchecking removes them. The shown checkbox state of a folder is computed from its descendants (`checked`, `mixed`, `unchecked`), so it stays right after a child is toggled. Before sending, two helpers turn the set into what the action needs:
- `selectedLinks(tree, selected)` returns the selected links, used for tag actions and delete.
- `moveRoots(tree, selected)` returns the topmost nodes to move, in tree order. A folder is included as a whole when it and all its descendants are selected. Otherwise its selected descendants are considered on their own.

IDs that are no longer in the tree (deleted elsewhere) are dropped when the tree changes.
*Alternative:* store only links and derive folder state. Then empty folders couldn't be selected or moved, and "fully selected folder moves as a whole" would be ambiguous.

### Selection mode replaces the drag handle
In selection mode `TreeRow` renders a checkbox (`aria-checked="mixed"` for partly checked) where the handle was, so the layout doesn't shift. Drag and drop is off, like while filtering. The last clicked row's index in the shown rows is kept for shift-click ranges.

### Bulk endpoint, one store method
`POST /api/favourites/bulk` decodes `{ ids, action, tags?, parentId? }` and calls `Store.Bulk(ctx, userID, BulkOp)`. `Bulk` validates the request shape, opens one transaction, loads every ID with the existing `load` (scoped to the user, so a missing or foreign ID gives `ErrNotFound`; 1000 indexed lookups in SQLite are cheap), validates kinds, and then:
- `delete`: deletes the links, closes the position gaps per parent, runs `removeUnusedTags` once. No token revocation, because only folders can be token roots.
- `addTags` / `removeTags` / `setTags`: normalizes names with `NormalizeTags`, then per link computes the new names and calls the existing `setTags` with the transaction. `removeUnusedTags` runs once at the end.
- `move`: rejects overlapping IDs and a target inside a given node (one recursive CTE over the given IDs), then appends each node at the end of the target in the given order. The client sends tree order.

To reuse the single-node code, `Move`'s body and the gap-closing part of `Delete` move into helpers that take a `db.Querier`, and the existing methods call them inside their own transaction.
*Alternative:* N single requests from the client. Not atomic, N round trips, and add/remove tags would need a read-modify-write per link. Rejected (see the exploration in issue #40).

### Undo by delaying the delete
Bulk delete doesn't call the server right away. The page keeps a `pendingDelete: { ids, timer }`. The shown tree leaves those IDs out, also after a refetch. The toast's "Undo" clears it, and nothing has been sent. The request is sent when the 8-second timer fires, when a new bulk delete starts (the previous one is sent first), and on `pagehide` or route change. The `pagehide` case uses `fetch(..., { keepalive: true })`, so closing the tab still deletes. If the request fails, the links come back and the error is shown.
*Alternative:* soft delete with a restore endpoint. It needs a `deleted_at` column, a purge job, and every read query to filter deleted nodes, including the Karakeep and Linkwarden APIs. Too much for one toast.

### Toast component
A small `Toast` in `components/` with an `aria-live="polite"` region, one message at a time, an action button and a close button. It pauses its timer while hovered or focused, so keyboard users can reach "Undo".

### Tag dialogs reuse `TagInput`
"Add tags" and "Set tags" use the existing `TagInput` (suggestions included). "Remove tags" shows checkboxes for the tags on the selected links with counts, computed from the tree. All three dialogs show "Applies to N links".

### Folder picker exclusions
`FavouritePicker` gets an optional `disabledIds` (with descendants), rendered as not choosable. The move dialog passes the selected folders.

## Risks / Trade-offs

- [The delete is lost if the browser is killed before `pagehide` or the keepalive request fails] → the links simply stay, which is the safe failure. The toast says "deleted", and that becomes true within 8 seconds in all normal cases.
- [Another tab changes the tree while a selection exists] → stale IDs are pruned on refetch. The server rejects unknown IDs with 404, the page reloads the tree and shows an error.
- [1000-ID limit] → "Select all" on a larger tree shows a message that at most 1000 nodes can be changed at once, and the actions are disabled. This is well above typical trees.
- [Refactoring `Move` and `Delete` into shared helpers can break single-node behaviour] → the existing store and HTTP tests cover them and must keep passing unchanged.
