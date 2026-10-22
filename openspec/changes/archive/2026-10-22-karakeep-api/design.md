# Design

## Context

- `app-tokens` added the tRPC layer (`internal/trpc`: registry, batching, envelopes, errors) and the Karakeep router with the login procedures (`httpapi/karakeep.go`). The token is in the request context. `favourites.Scope{UserID, RootID}` with `Contains`, `CheckCreate`, `CheckChange` and `CheckMove` already exists for this change.
- The favourites tree is `favourite_nodes` (folder/link, `parent_id`, `position`, `title`, `url`, `created_at`, `updated_at`), and tags are per user (`tags`, `favourite_node_tags`). Unused tags are removed after every tag change and delete (`removeUnusedTags`). `favourites.Store.Create` always appends; `Move` takes a position; `Delete` has the mode `DeleteMoveChildren` and already revokes app tokens.
- Favicons are resolved per origin when serving (`images.Favicons.Lookup`) and are public at `/api/images/{id}`.
- The shapes and inputs of the Karakeep procedures come from Karakeep's `packages/shared/types` (`zBookmarkSchema`, `zGetBookmarksRequestSchema`, `zBookmarkListSchema`, `zTagListRequestSchema`, `zUserStatsResponseSchema`, …), read at commit `85054d9`. The clients don't validate outputs, but they read nested fields without checks, so every non-optional field is always present.

## Goals / Non-Goals

**Goals:**
- Every rule about the subtree goes through `favourites.Scope`; no procedure queries nodes without it.
- Writes reuse `favourites.Store`, so validation, tag cleanup, favicon fetching and token revocation behave exactly as in the web app.
- Exact wire shapes, pinned by tests modelled on the requests the clients send.

**Non-Goals:**
- Performance for very large trees. A personal subtree has hundreds to a few thousand links.
- Karakeep's query language, smart lists, and anything listed as unsupported in the spec.

## Decisions

### Package layout

```
httpapi/karakeep.go     registers procedures; decodes inputs, maps errors
        │ scope from the token
        ▼
karakeep (new package)  Service: Bookmarks, Search, CheckURL, CreateBookmark,
        │               UpdateBookmark, UpdateTags, Lists, ListStats, CreateList,
        │               EditList, DeleteList, AddToList, RemoveFromList,
        │               Tags, RenameTag, DeleteTag, Stats
        ▼
favourites.Store        Create (new: First), Update, Move, Delete, Scope checks,
                        new: SubtreeNodes, TagLinkCount, RenameTag, DetachTag
```

`internal/karakeep` holds the mapping to Karakeep's shapes (Go structs with Karakeep's JSON names) and the rules from the spec. `httpapi/karakeep.go` stays a thin adapter: decode the input with `trpc.Decode`, call the service with the token's scope, and map errors (`favourites.ErrNotFound` → `NOT_FOUND`; validation errors and `karakeep.ErrUnsupported` → `BAD_REQUEST` with the message; `ErrTagShared` → `FORBIDDEN`; duplicate tag name → `CONFLICT`). Alternative: everything in `httpapi`. Rejected because the mapping is substantial and testable without HTTP.

### Reading the subtree

Each read loads the whole subtree in one recursive CTE from the scope's root (id, parent, kind, title, url, created and updated time), plus the tags of its links in a second query, and then filters, sorts and pages in Go. With personal-sized subtrees, this is simpler than translating every filter, sort order and cursor into SQL, and it keeps one code path for `getBookmarks`, `searchBookmarks`, `checkUrl`, `lists.*`, `tags.*` and `users.stats`. These read helpers go into `favourites` (`Store.SubtreeNodes(ctx, scope)`), next to `Scope`. Alternative: SQL per procedure with `LIMIT` and keyset cursors. Rejected as premature; it can replace the Go filtering later without changing behaviour.

- Cursor for `getBookmarks`: `{createdAt, id}` of the last item; the next page starts after it in the chosen order. `createdAt` is a date in the output (see superjson) and comes back as an ISO string in `json`, which is parsed as RFC 3339.
- `searchBookmarks`: offset cursor `{ver: 1, offset}`, as Karakeep.
- `checkUrl` and duplicate detection on create: normalise both URLs by dropping the fragment and a trailing slash of a non-root path (Karakeep's `normalizeUrl`), then compare.
- Favicons: `Favicons.Lookup` with the origins of the returned links; `favicon` is `<origin>/api/images/<id>`.

### Writes

- `createBookmark`: `favourites.Store.Create` with `ParentID` = scope root and the new `NewNode.First`, which inserts at position 0 and shifts the siblings up in the same transaction. Duplicate check first (read, then create; a race can create a duplicate link, which is harmless).
- `updateBookmark`: `CheckChange`, then `Store.Update` with title and URL.
- `updateTags`: `CheckChange`, then the link's current tag names plus or minus the requested ones, saved with `Store.Update(Tags)`. `tagId`s are resolved to names among the tags visible to the scope. The output IDs are read back after the update, so new tags have their IDs.
- `lists.create`: `CheckCreate(parent)`, then `Store.Create` (folder, appended). `lists.edit`: `CheckChange(list)`; rename with `Update`, move with `CheckMove` and `Move` to the end of the new parent. `Store.Move` already rejects moving a folder into itself (`ErrInvalidMove`).
- `lists.delete`: `CheckChange`, then `Store.Delete(DeleteMoveChildren)`, ignoring `deleteChildren`.
- `addToList`: `CheckMove(link, folder)` and `Move` to position 0. `removeFromList`: check that the link's parent is the list, then `Move` to position 0 of the scope root.
- `tags.update`: new `favourites.Store.RenameTag(userID, tagID, name)` with the tag-name rules and a uniqueness check (`ErrTagExists`); the service first compares `Store.TagLinkCount` with the number of links in the subtree that have the tag (`ErrTagShared` when they differ). `tags.delete`: new `Store.DetachTag(userID, tagID, nodeIDs)` with the subtree's link IDs, which removes the tag if it is then unused.
- Unsupported input fields are checked before any write, so a rejected call changes nothing.

### superjson output metadata

`trpc` serialises outputs like superjson for the two types the procedures return: `trpc.Date` (superjson `Date`; a `time.Time` type whose JSON is the `toISOString` format, which `time.Time`'s own RFC 3339 output isn't) and `trpc.Map` (superjson `map`, an ordered list of key/value pairs, encoded in `json` as `[[key, value], …]`). Before encoding, the layer walks the output with reflection, following JSON field names, slice indices and map keys, and records the path of each such value in `meta.values` (paths joined with `.`, a `.` inside a key escaped as `\.`, as superjson does). `meta` is left out when there's nothing to record. The paths and formats were checked against superjson 2.2.1's own `serialize`.

Alternative: handlers build `meta` by hand. Rejected because paths like `bookmarks.3.content.crawledAt` are easy to get wrong and would repeat in every procedure.

### Fixed values

- List `icon` `📁`: Karakeep requires an icon and Famodash folders have none.
- `users.settings`: `bookmarkClickAction: "open_original_link"`, so tapping a bookmark in the app opens the page instead of an empty reader view (Famodash stores no page content).
- `users.stats`: computed from the loaded subtree; "this week" starts on Monday 00:00 UTC, "this month" and "this year" on the first day, 00:00 UTC.

### `tags.create`

Refused with `FORBIDDEN`. A tag without links contradicts the favourites rule that unused tags are removed. Alternatives: create the tag and let the next cleanup remove it (confusing: the tag would vanish), or keep unused tags created by apps (a new exception to the tag rules). The app's "new tag" screen then shows the error; tags are still created by adding them to a bookmark.

## Risks / Trade-offs

- [Loading the whole subtree per request] → Fine for personal trees of a few thousand links. SQL pagination can replace it without behaviour changes.
- [Karakeep can change shapes or procedure names in a client update] → Shapes are pinned by tests; the README names the Karakeep version they were taken from. A renamed procedure shows up as `NOT_FOUND`.
- [Move semantics for lists differ from Karakeep, where a bookmark can be in several lists] → Specified and documented in the README. The app shows the bookmark in one list, which is consistent with what it gets from `getListsOfBookmark`.
- [`checkUrl` duplicate check isn't atomic with the create] → A rare duplicate link; no data loss.
- [Reflection-based superjson walk] → Limited to `trpc.Date` and `trpc.Map`; tested with nested structs, slices, pointers and maps.

## Migration Plan

No schema changes. Deploy the new binary; rollback is the previous binary.
