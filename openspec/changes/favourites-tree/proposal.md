# Proposal

## Why

The favourites tree is the core of Famodash: dashboards (through navigation-tree widgets) and the Karakeep-compatible API both read from and write to it, as the README describes. Neither exists yet, and both need the tree first. This change adds the tree and a page to manage it. The model holds only the fields needed now. It's shaped so that the later Karakeep work (folders as lists, links as bookmarks, stored tags) can build on it without a migration of existing data.

## What Changes

- Every user gets exactly one favourites tree with a root folder that can't be renamed, moved or deleted.
- The tree holds **folder** nodes (a name, nested to any depth) and **link** nodes (a URL and an optional title). Siblings keep a user-defined order.
- Links can carry **tags**. Tags belong to the user and are created on the fly when typed on a link. While typing, existing tags are suggested. A separate tag management screen is out of scope.
- New page `/favourites` (signed-in only) to manage the tree: add, edit and delete folders and links, and move or reorder nodes with **drag and drop**.
- Deleting a folder that isn't empty asks whether its contents should be deleted with it or moved to the folder's parent.
- The user menu gets a "Favourites" / "Favoriten" link to `/favourites`, above "Settings".
- New REST endpoints under `/api/favourites` and `/api/tags`.

Not part of this change: the Karakeep API and its tokens, metadata fetching (titles, favicons, preview images), further link fields (note, archived, starred, description, images, icons), and dashboards.

## Capabilities

### New Capabilities

- `favourites`: the per-user favourites tree (root, folders, links, ordering), its management page with drag and drop, deleting folders with either mode, and tags on links with on-the-fly creation and suggestions.

### Modified Capabilities

- `app-shell`: the signed-in user menu gets a "Favourites" link to `/favourites` between the email address and "Settings".

## Impact

- **Backend**: migration `0005` (`favourite_nodes`, `tags`, `favourite_node_tags`). A new package `internal/favourites` with the store, and new handlers under `/api/favourites` and `/api/tags`.
- **Frontend**: new route `/favourites` and a feature folder `features/favourites`. New dependency for drag and drop (see design). The user menu gets a new item. New translation keys (en and de).
- **Later changes**: the Karakeep API and its tokens (with tRPC support for the official extension and mobile apps), dashboards with navigation-tree widgets, and metadata fetching all build on this model.
