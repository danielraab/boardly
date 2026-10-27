# Design

## Context

`SharePage` preselects the folder stored under the localStorage key `boardly.share.folder` and writes it after each save. User settings live on the `users` table and are edited with `PATCH /api/me` (`name`, `slug`) from the Profile tab. App tokens already reference a folder with `root_id TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL` and check ownership in Go when the reference is written; foreign keys are enabled on every connection.

## Goals / Non-Goals

**Goals:**
- One server-side default folder per user, used by `/share` on every device.
- Deleting the folder never leaves a dangling reference.

**Non-Goals:**
- Using the default folder for links created through the Karakeep or Linkwarden APIs (they keep their token root) or on `/favourites`.
- Migrating the old localStorage value into the new setting.

## Decisions

- **Column `users.share_folder_id`** (migration `0016_share_folder.sql`), `TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL`, nullable. Same pattern as `app_tokens.root_id`: referencing only the node ID, because `SET NULL` on a composite `(id, user_id)` key would also clear `user_id`. Ownership is checked in Go on write. Deleting a folder (also with its parent, via the cascade on `favourite_nodes`) clears the setting automatically. Alternative: a separate `user_settings` table — not worth it for one field.
- **`null` means root; the root ID is stored as `null`.** Keeps "no default" and "root" a single state, and the root folder is created lazily (`EnsureRoot`), so a user may not have one yet when they first open settings.
- **Validation in `users.Store.Update`** via a new `Patch.ShareFolderID **string`: one `SELECT kind, parent_id FROM favourite_nodes WHERE id = ? AND user_id = ?`; not found or not a folder → `ErrInvalidFolder` (handler maps to 400 `invalid_folder`); `parent_id IS NULL` (root) → store `NULL`. A non-string, non-null JSON value is also `invalid_folder`, like `name` does with `invalid_name`.
- **API field `shareFolderId`** on `users.User`, so `GET /api/me` and the `PATCH` response include it without changing `writeMe`.
- **Profile tab form `ShareFolderForm`** next to `SlugForm`, built from the existing `FavouritePicker` in `folder` mode. Its value is `user.shareFolderId ?? root.id`; save sends `null` for the root. On success it updates the `me` query data like the other forms.
- **`SharePage`** reads `useMe()`; initial folder = `shareFolderId` if `findNode` finds a folder in the tree, otherwise root. The localStorage helpers and `SHARE_FOLDER_STORAGE_KEY` are removed; the stale key is left in browsers (harmless, not worth cleanup code).

## Risks / Trade-offs

- [Users relied on "last used" behaviour] → Documented as a behaviour change in the README/CHANGELOG; setting a default restores a stable target.
- [Default folder moved into another place] → The ID stays the same, so the setting follows the folder; no handling needed.
- [`me` is cached and stale after deleting the folder elsewhere] → `SharePage` falls back to root when the ID isn't in the current tree, so a stale value is harmless.

## Migration Plan

Additive migration with a nullable column; existing users start with `NULL` (root). Rollback: older binaries ignore the column.
