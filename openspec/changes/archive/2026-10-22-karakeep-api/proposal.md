# Proposal

## Why

With `app-tokens`, the Karakeep browser extension and mobile app can log in to Famodash, but every other call fails, so saving a page or browsing bookmarks doesn't work yet. This change implements the core of what these two clients call, mapped onto the token's folder in the favourites tree, so they become usable as a way to save and find links in Famodash.

## What Changes

- **Bookmarks** are the links in the token's subtree. New tRPC procedures: `bookmarks.getBookmarks` (newest first, filters by list and tag, cursor pagination), `bookmarks.searchBookmarks` (simple text search over title, URL and tags), `bookmarks.getBookmark`, `bookmarks.checkUrl` (the extension's "already saved" badge), `bookmarks.createBookmark` (links only, saved first in the token's root folder, an existing URL is returned instead of duplicated), `bookmarks.updateBookmark` (title and URL), `bookmarks.updateTags` and `bookmarks.deleteBookmark`.
- **Lists** are the folders below the token's root folder, nested like the folders. New procedures: `lists.list`, `lists.get`, `lists.stats`, `lists.getListsOfBookmark`, `lists.create`, `lists.edit` (name and parent), `lists.delete` (moves the contents to the parent, so no link is lost), `lists.addToList` (moves the link into the folder; a bookmark is in at most one list), `lists.removeFromList` (moves it back to the token's root folder) and `lists.getPendingInvitations` (always empty, so the app's list screen doesn't show an error).
- **Tags** are the user's tags that are on at least one link in the subtree. New procedures: `tags.list`, `tags.get`, `tags.update` (rename, refused when the tag is also used outside the subtree) and `tags.delete` (removes the tag from the links in the subtree). `tags.create` is refused, because a Famodash tag only exists while a link has it.
- **User**: `users.settings` (fixed settings: tapping a bookmark opens the original link, nothing is archived, AI features off) and `users.stats` (counts, top domains, tag usage and activity computed from the subtree).
- The tRPC layer learns superjson's `meta` for outputs with dates and maps, which bookmarks (`createdAt`), the cursor and `lists.stats` need.
- Karakeep features Famodash doesn't have aren't added: text and file bookmarks, notes, summaries, archiving, favouriting, smart and public lists, list icons and descriptions, highlights, reading progress, assets, list sharing. Setting such a field fails with `BAD_REQUEST`, and their procedures stay `NOT_FOUND`.
- New links saved through a connected app go **first** in the token's root folder, an exception to "new nodes go last".
- README updated.

## Capabilities

### New Capabilities
- `karakeep-api`: The bookmark, list, tag and user procedures for the Karakeep browser extension and mobile app, and how Karakeep's bookmarks, lists and tags map onto the token's subtree.

### Modified Capabilities
- `app-tokens`: The Karakeep tRPC transport also serialises dates and maps in outputs with superjson's `meta`, and accepts dates in inputs.
- `favourites`: Sibling order: links saved through a connected app are added first in the token's root folder.

## Impact

- **Backend**: new package `internal/karakeep` (mapping of the scoped tree to Karakeep's shapes: bookmark, list and tag queries, pagination, search, stats), new procedures in `httpapi/karakeep.go`, superjson `meta` in `internal/trpc`, a "first position" option for `favourites.Store.Create`, and scoped tag helpers in `favourites`.
- **Frontend**: none.
- **API**: about 25 tRPC procedures under `/api/karakeep/api/trpc/`. The web app's API is unchanged.
- **Docs**: README section "Karakeep-compatible server".
