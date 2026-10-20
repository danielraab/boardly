# Tasks

## 1. Data model

- [x] 1.1 Add migration `0005_favourites.sql` (`favourite_nodes` with the composite parent foreign key, CHECKs, partial unique root index and `(parent_id, position)` index; `tags` with `UNIQUE (user_id, name_key)`; `favourite_node_tags`). Verify migration tests that a link without a parent, a folder with a URL, a second root per user and a node under another user's folder are all rejected by the database, and that deleting a user removes their nodes and tags

## 2. Favourites store (`internal/favourites`)

- [x] 2.1 Implement validation (folder name, link title, `http`/`https` URL, tag name) with the limits from the design. Verify table tests, including `javascript:`, a URL without a host, whitespace-only names and limits ±1
- [x] 2.2 Implement `EnsureRoot` and `Tree` (whole tree in sibling order with tag names on links). Verify tests that the root is created once even when called concurrently and that the tree comes back nested and ordered
- [x] 2.3 Implement `Create` (parent must be the user's folder, appended at the end, tags resolved by `name_key` and created when missing) and `Update` (title, URL, full tag list; no URL or tags on folders; root is immutable). Verify tests for case-insensitive tag reuse including umlauts ("Ärger"/"ärger"), a foreign parent → not found, and tags on a folder → rejected
- [x] 2.4 Implement `Move` (root immutable, target must be a folder, no move into own subtree, position clamped, contiguous renumbering in old and new parent). Verify tests for reorder within a folder, move to another folder, move into a descendant → rejected, and positions `0..n-1` in both folders afterwards
- [x] 2.5 Implement `Delete` with `children=delete|move` (`folder_not_empty` without a mode; moved children take the folder's place in order) and remove unused tags after edits and deletes. Verify tests for both modes, the "A, Tools, Docs, B" order from the spec, and that a tag disappears only when its last link loses it
- [x] 2.6 Implement `SuggestTags` (prefix on `name_key`, `%`/`_`/`\` escaped, sorted, at most 10). Verify tests for prefix matching, case-insensitivity and a literal `%` in the query

## 3. REST API

- [x] 3.1 Add handlers in `internal/httpapi/favourites.go` for `GET/POST /api/favourites`, `PATCH`/`DELETE /api/favourites/{id}`, `POST /api/favourites/{id}/move` and `GET /api/tags`, registered with `requireUser`, with store errors mapped to the error codes from the design. Verify `httptest` tests for each endpoint, for 401 without a session, for 404 on another user's node, and for `root_immutable`

## 4. Favourites page

- [x] 4.1 Add the route `/favourites` with the signed-in guard, the `['favourites']` query and the API functions in `features/favourites`. Verify a test that a signed-out visitor ends up on `/login`
- [x] 4.2 Render the tree (root labelled "Favourites", collapsible folders expanded on load, links labelled by title or URL opening in a new tab, empty-state hint). Verify tests for the empty state, the title/URL fallback, `target="_blank"` with `rel="noopener noreferrer"`, and collapsing a folder
- [x] 4.3 Add the add/edit dialogs for folders and links with client-side validation mirroring the server. Verify tests for creating a nested folder, editing a link, and validation messages for an empty name and an invalid URL
- [x] 4.4 Add the tag input (Base UI `Combobox` with chips, debounced suggestions from `/api/tags`, attached tags filtered out, Enter adds a new tag). Verify tests for suggestions by prefix, filtering attached tags and adding a new tag on save
- [x] 4.5 Add the delete dialog (simple confirmation for links and empty folders; required choice between "delete contents" and "move contents to parent" for non-empty folders). Verify tests for both modes and for cancel
- [x] 4.6 Add dnd-kit and implement drag and drop over the flattened tree: before/after/into zones, the subtree of the dragged folder excluded, keyboard dragging, optimistic cache update with rollback on error. Unit-test the projection from a drop slot to `{parentId, position}` as a pure function. Verify component tests for reorder, move into a folder, a move rejected by the server (rollback) and a keyboard move
- [x] 4.7 Add all `favourites.*` translation keys in English and German. Verify that the i18n key-parity test passes

## 5. User menu

- [x] 5.1 Add the "Favourites" / "Favoriten" item (`Star` icon, `shell.favourites`) between the email address and "Settings" in `UserMenu.tsx`. Verify the updated app-shell tests for the item order with and without a name and that the link closes the menu and shows `/favourites`

## 6. Integration and docs

- [x] 6.1 Update the README's favourites section with what this change delivers (tags, drag and drop, the two folder-delete modes). Verify the flow in dev: create folders and links with tags, reorder and move by drag and drop, delete a folder in both modes, and inspect the tables in sqlite-web
- [x] 6.2 Run `make check` and `make test` and verify both pass

## Workflow follow-up

- Archive with `/opsx:archive` after review.
