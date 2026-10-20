# Design

## Context

Famodash has users (UUID ids), sessions, the `requireUser` middleware, the CSRF check (JSON content type and Origin), the `{"error": {"code", "message"}}` format and migrations up to `0004_passkeys.sql`. SQLite runs with `foreign_keys(1)`, WAL and `_txlock=immediate`, so a write transaction takes the write lock at `BEGIN`. The frontend uses TanStack Router file routes, TanStack Query, Base UI and the `api()` helper in `frontend/src/lib/api.ts`. Signed-in pages guard themselves in `beforeLoad` with `meQueryOptions`, as `/settings` does.

Requirements are in `favourites` and `app-shell` of this change. Later changes (the Karakeep REST and tRPC APIs, dashboards with navigation-tree widgets, metadata fetching) build on this model. So it has to stay minimal without blocking them.

## Goals / Non-Goals

**Goals:**
- A minimal tree model with database-level guarantees: a single parent, one root per user, and no links in other users' folders.
- Moves, reorders and deletes that are atomic and keep sibling positions contiguous.
- Tags whose case-insensitive uniqueness also works for non-ASCII names (German umlauts).
- Drag and drop that works with mouse, touch and keyboard.

**Non-Goals:**
- Fields beyond the minimum (note, archived, starred, description, preview image, favicon, folder icon). Later changes add them.
- Metadata fetching, a tag management screen, search, import and export.
- Paging or lazy loading the tree: a personal tree is loaded in one request.
- The Karakeep API, its tokens and subtree scoping helpers for it.

## Decisions

### One self-referencing table for folders and links
```
favourite_nodes
  id          TEXT PK (uuid)
  user_id     TEXT NOT NULL → users (id) ON DELETE CASCADE
  parent_id   TEXT NULL
  kind        TEXT NOT NULL CHECK (kind IN ('folder','link'))
  position    INTEGER NOT NULL
  title       TEXT NULL
  url         TEXT NULL
  created_at  TEXT NOT NULL
  updated_at  TEXT NOT NULL
  UNIQUE (id, user_id)
  FOREIGN KEY (parent_id, user_id) → favourite_nodes (id, user_id) ON DELETE CASCADE
  CHECK ((kind = 'folder' AND url IS NULL) OR (kind = 'link' AND url IS NOT NULL AND parent_id IS NOT NULL))
  CHECK (parent_id IS NOT NULL OR title IS NULL)          -- the root has no name

  UNIQUE INDEX favourite_nodes_root   ON (user_id) WHERE parent_id IS NULL
  INDEX        favourite_nodes_parent ON (parent_id, position)
```
- A single table means a single parent column. Subtree and ancestor queries are one recursive CTE, and the later Karakeep mapping ("list = folder, bookmark = link") stays one-to-one. *Alternative:* separate `folders` and `links` tables. A link's parent would still be a folder, but every tree query would need a `UNION`, and moves would need code for each table.
- The composite foreign key `(parent_id, user_id)` makes it impossible for a node to sit in another user's folder, even through a bug in the store. Deleting a folder cascades to its whole subtree.
- The parent must be a folder. A CHECK can't express that across rows, so the store enforces it inside the write transaction.
- Folder names are stored in `title`. The root has `parent_id IS NULL` and no title. The UI labels it "Favourites".

### The root is created lazily
`EnsureRoot(userID)` runs `INSERT … ON CONFLICT DO NOTHING` against the partial unique index and then selects the root. Every tree read and write calls it first. Existing users therefore need no backfill, and account creation doesn't have to know about favourites. *Alternative:* create the root in `users.EnsureByEmail` plus a backfill in the migration. That couples the two packages and adds a one-off data migration for no gain.

### Integer positions, renumbered in a transaction
Siblings have positions `0..n-1`. Create appends at `n`. A move closes the gap in the old parent and opens one in the new parent, all within one transaction, which takes the write lock at `BEGIN` because of `_txlock=immediate`. The index on `(parent_id, position)` isn't unique, because SQLite checks uniqueness per row during the shifting `UPDATE`s. *Alternative:* fractional or lexicographic keys (LexoRank) avoid renumbering but need rebalancing and are harder to inspect in sqlite-web. Personal folders are small, so renumbering is cheap.

### Moves are checked against the subtree
`Move(id, newParentID, position)` rejects the move when the node is the root, the new parent isn't a folder of the same user, or the new parent is the node itself or one of its descendants. The descendant check is a recursive CTE that walks the target's ancestors. The position is clamped to `0..len(children)`.

### Delete has two modes
- `children=delete`: delete the folder row. `ON DELETE CASCADE` removes the subtree.
- `children=move`: in one transaction, shift the parent's later siblings by `k−1`, where `k` is the number of children. Then give the children `parent_id = folder.parent_id` and `position = folder.position + their old position`, and delete the now empty folder.

Links and empty folders need no mode. A non-empty folder without `children` is rejected with `400 folder_not_empty`, so the server never guesses.

### Tags with a normalized key
```
tags                                       favourite_node_tags
  id         TEXT PK (uuid)                  node_id → favourite_nodes (id) ON DELETE CASCADE
  user_id    TEXT → users ON DELETE CASCADE  tag_id  → tags (id) ON DELETE CASCADE
  name       TEXT NOT NULL                   PRIMARY KEY (node_id, tag_id)
  name_key   TEXT NOT NULL                   INDEX (tag_id)
  created_at TEXT NOT NULL
  UNIQUE (user_id, name_key)
```
- `name_key` is the trimmed name passed through Go's `strings.ToLower`. SQLite's `NOCASE` only folds ASCII, so "Ärger" and "ärger" would become two tags. `name` keeps the spelling first used.
- A link's tags are set as a full list of names on create and edit. The store resolves each name by `name_key`, creating missing tags, and replaces the link's rows in `favourite_node_tags`.
- Removing unused tags: after every write that can detach tags (edit, delete), the same transaction runs `DELETE FROM tags WHERE user_id = ? AND NOT EXISTS (…)`. Without a management screen, unused tags would otherwise stay in the suggestions forever. The Karakeep change will revisit this, since its `POST /tags` creates tags without links.
- Suggestions: `WHERE user_id = ? AND name_key LIKE ? ESCAPE '\'`, using the lower-cased prefix with `%`, `_` and `\` escaped, `ORDER BY name_key LIMIT 10`. Tags the link already has are filtered in the client, which knows them.
- The store checks that only links get tags. The tables can't express that.

### Validation
- Folder name: trimmed, 1–100 characters (the same limit as Karakeep list names).
- Link title: trimmed, at most 1000 characters (the Karakeep limit). An empty title becomes `NULL`.
- URL: trimmed, at most 2048 characters, parsed with `net/url`, and the scheme must be `http` or `https` with a non-empty host. Rejecting `javascript:` and similar schemes matters because dashboards will render these URLs as links.
- Tag name: trimmed, 1–100 characters.
- The frontend runs the same checks for immediate feedback, using `new URL()` and a scheme check. The server stays authoritative.

### REST API
All endpoints need a signed-in user. JSON uses camelCase.

| Method & path | Body / query | Response |
|---|---|---|
| `GET /api/favourites` | – | `200` the whole tree from the root, nested |
| `POST /api/favourites` | `{parentId, kind, title?, url?, tags?}` | `201` the node, appended to the parent |
| `PATCH /api/favourites/{id}` | `{title?, url?, tags?}` | `200` the node |
| `POST /api/favourites/{id}/move` | `{parentId, position}` | `204` |
| `DELETE /api/favourites/{id}` | `?children=delete\|move` | `204` |
| `GET /api/tags` | `?q=<prefix>` | `200 [{id, name}]`, at most 10 |

A node in the response looks like this:

```json
{ "id": "…", "kind": "folder", "title": "Work", "children": [
  { "id": "…", "kind": "link", "title": "Docs", "url": "https://example.org/docs", "tags": ["reading"] }
] }
```

- Error codes: `not_found` (404), and `invalid_title`, `invalid_url`, `invalid_tag`, `invalid_kind`, `invalid_field` (URL or tags on a folder), `invalid_parent`, `invalid_move`, `folder_not_empty` and `root_immutable` (renaming, moving or deleting the root), all 400.
- `position` in a move is the index the node has among the target folder's children after the move. It is clamped to the valid range.
- A link's `tags` are returned sorted by name, in the tree and in the responses of `POST` and `PATCH`.
- `kind` can't be changed by `PATCH`, and `url` and `tags` are rejected on folders.
- The move is its own endpoint and isn't part of `PATCH`: it has different rules (subtree check, renumbering) and is the only call drag and drop makes.
- Reads return the whole tree because the page needs all of it, and the size of a personal tree makes paging unnecessary. The tree is built in Go from one `SELECT … ORDER BY parent_id, position` plus one query for the tags.

### Backend layout
A new package `internal/favourites` (`Store` with `Now`, same pattern as `users.Store`) owns the tree and the tags, because tags only exist on links. Handlers go in `internal/httpapi/favourites.go` and are registered in `router.go` with `a.requireUser`. Store errors map to API errors in the handlers, as in `passkeys.go`.

### Frontend
- **Route** `frontend/src/routes/favourites.tsx`, with the same `beforeLoad` guard as `/settings`. The feature lives in `frontend/src/features/favourites/`.
- **Data**: TanStack Query key `['favourites']` for the tree. Mutations update the cache optimistically for moves, so the dropped item doesn't jump back while waiting, and invalidate the key afterwards. If a request fails, the cache rolls back and an error is shown.
- **Drag and drop**: **dnd-kit** (`@dnd-kit/core` and `@dnd-kit/sortable`, the stable line, versions pinned when they're added). It has pointer, touch and keyboard sensors and screen-reader announcements. The tree is rendered as a flat list of visible nodes with a depth indent, following dnd-kit's sortable-tree pattern. The drop target comes from the pointer's position over a row: the upper quarter means "before", the lower quarter means "after", and the middle of a folder row means "into", with append at the end. With the keyboard, the arrow keys move the dragged node one slot at a time through the same flat list, and the slot gives the parent and the position. Rows of the dragged folder's own subtree are not valid targets. *Alternatives:* `react-arborist` is a complete tree component, but its own styling and keyboard model are hard to fit into Base UI and Tailwind. `@atlaskit/pragmatic-drag-and-drop` has good tree hit-boxes but no keyboard dragging. Native HTML5 drag and drop has no touch or keyboard support.
- **Collapse state**: kept in component state, with all folders expanded on load. Dropping onto a collapsed folder appends to it.
- **Dialogs**: Base UI `Dialog` for adding and editing (folder: name; link: URL, title, tags) and for deleting. For a non-empty folder, the delete dialog offers the two modes as a radio group. Neither is preselected, so the user has to choose.
- **Tag input**: Base UI `Combobox` in multiple-selection mode (chips). Its options come from `GET /api/tags?q=`, debounced by about 200 ms. Typed text that matches no suggestion can be added as a new chip with Enter. Tags are only sent when the link form is saved.
- **User menu**: a new item in `components/UserMenu.tsx` between the email address and "Settings", with a lucide icon (`Star`).
- **i18n**: keys under `favourites.*` (page, dialogs, validation, empty state) and `shell.favourites` (next to `shell.settings`), in English and German.

## Risks / Trade-offs

- [Drag and drop in a nested tree is the most complex UI in the app so far. Ambiguous drop zones and keyboard dragging are easy to get wrong.] → Keep the three-zone rule simple and unit-test the "flat list and slot → parent and position" projection as a pure function. Add component tests for reorder, move-into and the forbidden move into the folder's own subtree.
- [Optimistic moves can briefly show an order the server rejects.] → The server is authoritative. The cache rolls back, the tree is refetched and an error toast is shown.
- [Removing unused tags automatically will conflict with Karakeep's `POST /tags`, which creates tags without links.] → This is documented here, and the Karakeep change decides whether to keep the cleanup or add a tag management screen.
- [The whole tree is loaded on every visit.] → That's fine for thousands of nodes. Paging can be added behind the same API if it ever matters.
- [Concurrent edits from two tabs can interleave moves.] → Each move is one write transaction with contiguous renumbering, so the tree stays consistent. The last write wins.

## Migration Plan

Migration `0005_favourites.sql` creates the three tables and indexes. It contains no data migration, because roots are created lazily. Rolling back to the previous binary leaves the new tables unused, and migrations only go forward.

## Open Questions

- Should the collapse state be remembered per browser (for example in `localStorage`)? It isn't for now, and all folders start expanded.
