# Tasks

## 1. Backend

- [x] 1.1 Move the explorer's subtree loading (owner lookup, not found → missing, `SetIcons`, `Cached` by `subtreeKey`) into a shared helper in `internal/widgets`, and verify that the existing explorer tests still pass
- [x] 1.2 Add the `link_list` kind with `Normalize` (strict decode, non-empty `folderId` ≤ 64 characters, layout `list`/`compact`/`tiles`, size S/M/L, shared title rules) and register it in `widgets.Register`. Verify with layout tests for a valid config, layout `grid`, size `XL`, a missing folder ID, a title of 201 characters, and saving with a deleted folder
- [x] 1.3 Implement `Resolve` with the depth-first flatten (links only, subfolders only when `includeSubfolders`): visitor with flag off → left out, missing folder → `{missing:true}` for the owner and left out for visitors, otherwise `{links}`. Verify with tests for the nested Work/Tools/Deep example with the flag off and on, owner vs. visitor with the public flag off and on, a deleted folder, and that a visitor's response contains no folder names and no hidden links

## 2. Frontend

- [x] 2.1 Add `flattenLinks(folder, includeSubfolders)` in `widgets/linkList/`, and verify with a unit test on the same nested example as the Go test (order and subfolder flag)
- [x] 2.2 Build the link list view with the three layouts (list rows like `LinkView`, compact rows, auto-fill tile grid), the S/M/L sizes, the optional heading at `headingLevel`, tooltips, new-tab links that don't navigate in edit mode, and the empty and missing messages. Verify with component tests for each layout, the globe fallback, the heading, the empty and missing states, and that links have `target="_blank"`
- [x] 2.3 Build the settings dialog (`FavouritePicker` in folder mode with the missing-folder hint, title, "Show items from subfolders", layout, size, "Show on public dashboard", common width/height, confirm disabled without a folder), and verify with a test that adding requires a folder and that the chosen settings are applied
- [x] 2.4 Register the kind (icon, labels, defaults; view uses `data` in view mode and the favourites query in edit mode), and verify with an editor test that adds a link list through the picker and sees its links in the draft
- [x] 2.5 Add the en/de keys (kind name and description, setting labels, layout names, empty and missing messages), and verify that the i18n check passes

## 3. Docs and verification

- [x] 3.1 Add the link list to the widget table in `README.md`
- [x] 3.2 Run `make check` and `make test` and verify that both pass
- [x] 3.3 Verify in the browser: add a link list on a folder with subfolders, switch the subfolder flag and all three layouts and sizes, narrow the widget to see tiles wrap, set a fixed height to see it scroll, and open the public dashboard signed out with the flag off (not shown) and on (shown)
