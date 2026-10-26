# Tasks

## 1. Backend

- [ ] 1.1 Add `Store.ListTags` (id, name, link count, sorted by name key); verify store tests including tags of another user being excluded
- [ ] 1.2 Add `Store.MergeTag` and `Store.DeleteTag` in one transaction each; verify store tests (merge dedupes shared links, source removed, target name kept; delete keeps links; unknown or foreign tag → `ErrNotFound`)
- [ ] 1.3 Handle `GET /api/tags?all=true`, add `PATCH /api/tags/{id}` (rename, 409 `tag_exists`, `merge: true`) and `DELETE /api/tags/{id}`; verify httpapi tests for each scenario of "Tag management API" and that `GET /api/tags?q=` is unchanged
- [ ] 1.4 Run `go test ./...` in `backend/` and verify it passes

## 2. Frontend

- [ ] 2.1 Add `listTags`, `renameTag` (with `merge`) and `deleteTag` to `api.ts`
- [ ] 2.2 Add `ManageTagsDialog` (list with counts and search, inline rename, merge question on 409, delete confirmation with count, empty state) and the "Manage tags" button on `FavouritesPage`; verify tests for list, rename, change case, merge, cancel merge, delete
- [ ] 2.3 Filter by a tag from the dialog, and keep the URL's tag filter in sync after rename, merge or delete; verify tests

## 3. Translations and verification

- [ ] 3.1 Add en/de strings (manage tags, rename, delete confirmation with plurals, merge question, no tags yet); verify the i18n key parity test passes
- [ ] 3.2 Run frontend lint, typecheck and `vitest` and verify they pass
- [ ] 3.3 Run the app and check rename, merge, delete and filtering from the dialog in light and dark themes
