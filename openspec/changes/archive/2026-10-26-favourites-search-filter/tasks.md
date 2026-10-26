# Tasks

## 1. Tree helpers

- [x] 1.1 Add `filterTree(root, { terms, tags })` to `tree.ts` (terms on title, URL, tag names and folder path, all tags required, folder-name match keeps the whole subtree when no tag is chosen, `null` when nothing matches); verify with unit tests for each scenario of "Search favourites" and "Filter favourites by tag"
- [x] 1.2 Add `tagCounts(root)`, `linkCounts(tree)` and `highlight(text, terms)` to `tree.ts`; verify with unit tests (sorted tags with counts, nested link counts, case-insensitive and overlapping highlight ranges)

## 2. Stored collapse state

- [x] 2.1 Extract the storage logic of `useExpanded` into `lib/useStoredIdSet(key)` and use it in the widget with the same key; verify the existing explorer tests still pass
- [x] 2.2 Use `useStoredIdSet("boardly.favourites.collapsed.<userId>")` for the page's collapsed set, pruned to existing folders; verify with a test that a collapsed folder stays collapsed after remounting the page

## 3. Favourites page

- [x] 3.1 Add `validateSearch` for `q`, `tags` and `folder` to `routes/favourites.tsx` and regenerate the route tree; verify typecheck passes
- [x] 3.2 Add the search field (URL-backed, `replace` on typing, `/` shortcut) and render the pruned tree with a separate in-memory collapse set while filtering; verify tests: find by title, URL, tag, several terms, folder-name match, shortcut, reload restores the filter
- [x] 3.3 Add the tag filter popover with counts, removable chips, clickable row tags, "Clear filters" and the "No favourites match" state; verify tests for one tag, several tags, row tag click, search plus tag, clear, nothing matches
- [x] 3.4 Add `movable` to `TreeRow`, hide drag handles and show the reorder hint while filtering; verify tests that no move action exists while filtered and that editing a shown link keeps the filter
- [x] 3.5 Add the "Expand all" / "Collapse all" button acting on the collapse set in use; verify tests for both directions, and that clearing a filter restores the remembered state
- [x] 3.6 Show link counts on folder rows (full or filtered) and highlight matched text in names; verify tests for counts, filtered counts and a `<mark>` around the match
- [x] 3.7 Handle `?folder=`: expand it and its ancestors in the remembered set, scroll into view, highlight briefly, then drop the param; verify tests for a nested folder, an unknown ID and a link ID

## 4. Explorer widget

- [x] 4.1 Add `titleLink` to `ExplorerTree` and pass it from `index.tsx` for the owner outside edit mode with an existing folder (`{}` for the tree root); verify tests: owner title links to `/favourites?folder=<id>`, tree root links to `/favourites`, visitor, edit mode and missing folder show plain text

## 5. Translations and verification

- [x] 5.1 Add en/de strings (search label and placeholder, tag filter, clear filters, no match, reorder hint, expand/collapse all, link count); verify the i18n key parity test passes
- [x] 5.2 Run frontend lint, typecheck and `vitest` and verify they pass
- [x] 5.3 Run the app and check search, tag filter, counts, expand/collapse all, remembered state, the deep link from an explorer title, and phone width in light and dark themes
