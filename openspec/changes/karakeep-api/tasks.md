# Tasks

## 1. tRPC layer

- [x] 1.1 Add superjson output metadata to `internal/trpc`: a reflection walk that records `Date` paths for `trpc.Date` and `map` paths for `trpc.Map`, dates written in `toISOString` format, `meta` left out when empty, `.` in keys escaped; verify with unit tests for nested structs, slices, pointers, maps, `trpc.Map` and an output without dates

## 2. Favourites store

- [x] 2.1 Add `NewNode.First` to `favourites.Store.Create` (insert at position 0, shift siblings); verify with a favourites test that the new link comes before the existing children and that the default still appends
- [x] 2.2 Add `Store.SubtreeNodes(ctx, scope)` returning every node of the scope's subtree with parent, kind, title, URL, times and tags; verify with a favourites test that nodes outside the scope and other users' nodes are left out
- [x] 2.3 Add `Store.TagLinkCount`, `Store.RenameTag` (tag-name rules, `ErrTagExists` for a name taken regardless of case) and `Store.DetachTag` (removes a tag from the given links, then drops it if unused); verify with favourites tests for both, including a tag also used outside the given links

## 3. Karakeep service (`internal/karakeep`)

- [x] 3.1 Add the Karakeep shapes (bookmark, list, tag, settings, stats) and the bookmark mapping from a link (fixed fields, tags, favicon URL, crawl status); verify with a unit test of the full JSON of one mapped bookmark
- [x] 3.2 Implement `Bookmarks` (order, `listId`, `tagId`, `archived`/`favourited`, limit, `{createdAt, id}` cursor), `Search` (terms, `#tag`, offset cursor) and `CheckURL` (normalised compare); verify with service tests for paging over 25 links, list and tag filters, search terms and tags, and URL normalisation
- [x] 3.3 Implement `CreateBookmark` (first in the root, duplicate → `alreadyExists`, links only, unsupported fields), `UpdateBookmark`, `DeleteBookmark` and `UpdateTags`; verify with service tests for each, including a rejected `text` bookmark, `archived: true` leaving the link unchanged, and attaching a new tag by name
- [x] 3.4 Implement lists: `Lists`, `List`, `ListStats`, `ListsOfBookmark`, `CreateList`, `EditList` (rename, move, unsupported fields), `DeleteList` (contents move up), `AddToList` and `RemoveFromList` (move to position 0); verify with service tests for nested folders and `parentId`, stats, moving between lists, removing a link that isn't in the list, and deleting a list with links and a subfolder
- [x] 3.5 Implement tags: `Tags` (scope, counts, filters, sorting, paging), `Tag`, `RenameTag` (`ErrTagShared`, conflict), `DeleteTag` (subtree only) and `CreateTag` refusal; verify with service tests for a tag used inside and outside the subtree
- [x] 3.6 Implement `Settings` and `Stats` (counts, top domains, tag usage, activity in UTC); verify with a service test using a fixed clock

## 4. Procedures

- [x] 4.1 Register all procedures from the spec in `httpapi/karakeep.go` with input decoding and error mapping (`NOT_FOUND`, `BAD_REQUEST`, `FORBIDDEN`, `CONFLICT`); verify with httptest requests shaped like the clients': save a page from the extension (`createBookmark`, `checkUrl`, `getBookmark` with `Date` meta), browse in the app (`getBookmarks` with cursor, `lists.list` + `lists.stats` with `map` meta, `tags.list`), a call outside the subtree (`NOT_FOUND`) and an unsupported procedure

## 5. Docs and checks

- [x] 5.1 Update the README's "Karakeep-compatible server" section: what works in the extension and the app, how lists and tags map to folders and tags (move semantics, new links first), what isn't supported, and the Karakeep version the shapes come from; verify by reading it against the spec
- [ ] 5.2 Run `make check` and `make test`, then save a page with the Karakeep browser extension and browse bookmarks, lists and tags in the mobile app against a local server; verify that both commands pass and the clients work without errors on the supported screens
