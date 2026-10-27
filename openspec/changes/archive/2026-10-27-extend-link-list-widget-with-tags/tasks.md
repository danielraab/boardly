# Tasks

## 1. Backend

- [x] 1.1 Add `Store.FolderLinks` (links of a folder, optionally with subfolders, depth first, with `createdAt` and tag IDs; `ErrNotFound` for an unknown or foreign folder) and `Store.TagExists` to `internal/favourites`; verify with store tests for order, subfolders, tag IDs and other users' folders and tags
- [x] 1.2 Extend `linkListConfig` with `tagId` and `sort`, normalizing empty values (nil tag, `favourites`) and rejecting an unknown sort or an over-long tag ID; verify with `linklist_test.go` cases for an old config, an invalid sort and a valid tag/sort config
- [x] 1.3 Switch `Resolve` to the new loader (cached per lookup, favicons set), filter by tag, return `{missing: "tag"}` for a missing tag to the owner and leave it out for visitors; verify with tests for a tag across subfolders, a renamed tag, a deleted tag (owner and visitor), the folder check coming first, and a visitor response without tag names or dates
- [x] 1.4 Implement the sorts (`favourites`, `newest` with ID tie-break, `alphabetical` with `golang.org/x/text/collate`, then URL, then ID) and make `golang.org/x/text` a direct dependency; verify with tests matching the spec's sort scenarios, including "Ärzte" before "Zoo" and sorting across subfolders
- [x] 1.5 Run `go test ./...` and `go vet ./...` in `backend/` and verify they pass

## 2. Frontend

- [x] 2.1 Add `tagId` and `sort` to `LinkListConfig` and `defaultConfig`, treating configs without them as no tag and `favourites`; extend the preview helpers in `flatten.ts` with tag filtering by name and the three sorts (`Intl.Collator` with base sensitivity, same tie-breaks); verify with unit tests
- [x] 2.2 Settings dialog: "Tag" select ("No tag filter" + tags from `listTags`, sorted by name), missing-tag hint, "Sort" select; verify with tests in `LinkList.test.tsx`
- [x] 2.3 View: edit-mode preview applies tag (ID → name via the tag list, missing tag → missing state) and sort; `LinkListView` shows "This tag no longer exists" for `missing: "tag"` and "No links with this tag in this folder" for an empty filtered list; verify with tests in `LinkList.test.tsx`
- [x] 2.4 Add the English and German strings (tag label, "No tag filter", sort label and options, the two tag messages with "„Tag“" in German) and verify both locale files have the same keys
- [x] 2.5 Run the frontend tests, type check and lint and verify they pass

## 3. Wrap-up

- [ ] 3.1 In the running app, verify a link list on "Favourites" with subfolders and a tag shows tagged links from several folders, each sort order works, a tag rename keeps it working, and deleting the tag shows "This tag no longer exists"
- [x] 3.2 Run `openspec validate extend-link-list-widget-with-tags --strict` and verify it passes
