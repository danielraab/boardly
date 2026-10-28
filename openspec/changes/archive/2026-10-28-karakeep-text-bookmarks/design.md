# Design

## Context

- `app_tokens` has `root_id … REFERENCES favourite_nodes (id) ON DELETE SET NULL`; a folder delete revokes affected active tokens in Go (`favourites.go`, `UPDATE app_tokens SET revoked_at …`) in the same transaction. Tokens can't be changed (`PATCH` answers 405).
- `httpapi/karakeep.go` authenticates the token and calls each procedure with `favourites.Scope{UserID, RootID}`. `karakeep.Service` loads the scoped tree (`s.load`) and filters, sorts and pages links in Go; cursors are `{createdAt, id}`.
- Notes and categories live in `internal/notes`. Category delete moves subcategories up and sets notes' category to `NULL`. `notes-in-import-and-share` adds `FromMarkdown`, which this change reuses.

## Goals / Non-Goals

**Goals:**
- Karakeep clients see and edit notes as text bookmarks, limited to the token's note scope.
- Editing a token's scope keeps its secret, so the app stays connected.

**Non-Goals:**
- Notes in the Linkwarden API (Linkwarden has no text type).
- Tags on notes, or categories as Karakeep lists.
- Renaming tokens or changing their expiry.

## Decisions

- **Two columns instead of one nullable ID** — migration `0019_app_token_note_scope.sql` adds `note_scope TEXT NOT NULL DEFAULT 'none' CHECK (note_scope IN ('all', 'category', 'none'))` and `root_category_id TEXT REFERENCES note_categories (id) ON DELETE SET NULL`, then sets `note_scope = 'all'` for `app = 'karakeep'`. A single nullable `root_category_id` can't tell "top level" from "deleted category": `ON DELETE SET NULL` would silently widen a deleted category's token to all notes. With `note_scope = 'category'` and `root_category_id IS NULL` the token reads as "Deleted category", and it is revoked anyway. Alternative: `ON DELETE CASCADE`/no FK — would lose the token row or leave a dangling ID.
- **Revocation in `notes.Store.DeleteCategory`**: in its transaction, before deleting, `UPDATE app_tokens SET revoked_at = ? WHERE root_category_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)`. The same `app_tokens` query pattern lives in `favourites` for folders. `DeleteCategoryDialog` counts the affected tokens on the client from `appTokensQueryOptions` (active Karakeep tokens with `noteScope` `category` and this `rootCategoryId`), like the folder `DeleteDialog` does, and invalidates the app tokens query after the delete. No new endpoint.
- **`PATCH /api/app-tokens/{id}`** replaces the 405: `apptokens.Store.Update(ctx, userID, id, Patch{RootID *string, RootCategoryID **string})` in one transaction: load (404), active check (409 `token_inactive`), validate the folder (`invalid_root`), validate the category and app (`invalid_root_category`), update. Unknown JSON fields → 400 `invalid_request` (`DisallowUnknownFields`). Authentication reads the scope on every request, so the change applies at once.
- **Token API fields** `noteScope` and `rootCategoryId` on `apptokens.Token`; `NewToken.RootCategoryID *string` (nil = `all` for Karakeep, `none` for Linkwarden).
- **`karakeep.Scope`** = `{Links favourites.Scope; Notes notes.Scope}` where `notes.Scope{UserID string; Mode ScopeMode; CategoryID string}`. The handler builds it from the token; every service method takes it. `notes.Store.ScopedNotes(ctx, sc)` returns the notes in scope with one recursive CTE over `note_categories` for `category`, all of the user's notes for `all`, nothing for `none`. `ScopedNote(ctx, sc, id)` returns `notes.ErrNotFound` outside the scope.
- **Merged listing in Go**: `Bookmarks` and `Search` build one slice of `item{createdAt, id, link *ScopedNode, note *Note}`, then reuse the existing sort/cursor/page code (generalised from `*ScopedNode` to `item`). `listId`/`tagId` filters drop notes first. Note and node IDs are both UUIDs, so one cursor and one ID space work; `GetBookmark`/`Update`/`Delete` try the link first, then the note. Alternative: SQL `UNION` paging — faster for huge sets, but the service already pages in memory and personal scale is small.
- **`notes.ToMarkdown(Doc) string`** in `internal/notes/markdown.go` next to `FromMarkdown`. Escapes `\ * _ ~ ` [ ] < >` and, at line start, `#`, `>`, `-`, `+`, digits followed by `.`/`)`, so the round trip holds. A test checks `FromMarkdown(ToMarkdown(d))` equals `d` without task IDs for documents without the three listed exceptions, plus a fuzz test over random valid documents.
- **`notes.ReplaceContentFromMarkdown(old Doc, md string) Doc`**: `FromMarkdown`, then walk task items in document order and pop the first unused old ID from a `map[text][]id` (text = trimmed plain inline text). `UpdateBookmark` compares `md` with `ToMarkdown(current)` first and skips the content change when equal, so a client resending unchanged text doesn't drop empty paragraphs. Updates go through `notes.Store.UpdateNote` with the current version, so version, `updated_at` and validation stay in one place; a concurrent web edit makes the call fail with a version conflict, which maps to tRPC `CONFLICT`.
- **Create**: `CreateBookmark` with `type: "text"` calls `notes.Store.CreateNote` with `CategoryID` from the scope (`nil` for `all`), title and `FromMarkdown(text)`; note-scope `none` → `InputError`. `invalid_note`/`invalid_document` map to `BAD_REQUEST`.
- **Stats**: count notes into `numBookmarks` and the activity buckets; domains only from links.
- **Frontend**: `CreateAppTokenDialog` gets the `CategoryPicker` (from `notes-in-import-and-share`) with "All notes" as the empty choice, rendered for Karakeep only. A new `EditAppTokenDialog` reuses the folder and category fields and sends only the changed values. `ConnectedAppsTab` shows the category path and an "Edit" action for active tokens. `DeleteCategoryDialog` shows the connected-apps warning.

## Risks / Trade-offs

- [Karakeep app UI around tags/lists may call `updateTags` or `addToList` on a text bookmark] → Empty `updateTags` succeeds; real changes return a clear `BAD_REQUEST` message, which the apps show.
- [Markdown round trip drops empty paragraphs on app edits] → Only when the app actually changes the text; documented in the note-document spec.
- [Editing a token's scope widens access without re-entering the secret] → Only the signed-in owner can edit, from the session-authenticated settings page; same trust as creating a token.
- [Loading all notes in scope for every listing] → Personal scale; the same trade-off the service already makes for links.

## Migration Plan

Additive migration: two columns, existing Karakeep tokens get `note_scope = 'all'` (decided with the user), Linkwarden tokens `none`. Rollback: older binaries ignore the columns; `PATCH` disappears again.
