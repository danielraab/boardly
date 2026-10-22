# Tasks

## 1. Backend

- [x] 1.1 Add `Subtree(ctx, userID, folderID)` to `favourites.Store` (recursive CTE, folders and links with `iconImageId`, no tags), returning not found for a missing node or a link. Verify with `favourites_test.go` cases: nested folders, another user's folder, a link ID
- [x] 1.2 Register the `favourite_explorer` kind: `Normalize` (size S/M/L, non-empty `rootFolderId` ≤ 64 characters, default `showIfPublic`). Verify with layout tests for a valid config, size `XL`, a missing folder ID, and saving with a deleted folder
- [x] 1.3 Implement `Resolve` (visitor + flag off → left out, missing folder → `{missing:true}` for the owner and left out for visitors, otherwise `{children}`) with a per-lookup cache. Verify with lookup tests for owner, visitor flag off, visitor flag on, deleted folder, and that a visitor's response has no nodes outside the root folder
- [x] 1.4 Add the optional `title` to the explorer config (trimmed, ≤ 200, empty → null). Verify with layout tests for a trimmed title, an empty title and a title of 201 characters

## 2. Frontend

- [x] 2.1 Add `widgets/sizes.ts` with the S/M/L styles, and verify that `pnpm tsc` passes
- [x] 2.2 Build `ExplorerTree` (nested lists with `aria-expanded` folder buttons, folder/link rows, `ImageIcon`, truncation, empty and missing states, new-tab links), and verify with component tests for the tree, the globe fallback, the empty folder and the missing folder
- [x] 2.3 Add the collapse state hook (`famodash.explorer.<id>` in localStorage, collapsed by default, tolerant of blocked storage) and the expand-all/collapse-all button. Verify with tests: first render is collapsed, the state survives a remount, and expand all / collapse all switch every folder and the button label
- [x] 2.4 Build the settings dialog (`FavouritePicker` in folder mode, size, "Show on public dashboard", common width/height, confirm disabled without a folder), and verify with a test that adding requires a folder and that cancel adds nothing
- [x] 2.5 Register the kind (icon `FolderTree`, labels, defaults, view: `data` in view mode, favourites query in edit mode), and verify with an editor test that adds an explorer through the picker and sees its tree in the draft
- [x] 2.6 Add the en/de keys (kind name and description, setting labels, empty, missing, expand/collapse all), and verify that the i18n check passes
- [x] 2.7 Add the title field to the settings and the heading to the view, with `headingLevel` passed to widget views. Verify with tests: the heading and its level, no heading without a title, and saving the title

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify that both pass
- [x] 3.2 Verify in the browser: add an explorer, save, reload (state remembered), open the public dashboard signed out with the flag off (not shown) and on (shown)
