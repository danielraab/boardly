# Proposal

## Why

Tags only come and go as a side effect of editing links. A typo or near-duplicate ("Reading" and "reading-list") can only be fixed by editing every link that has it. Once search and the tag filter (`favourites-search-filter`) show all tags with counts, the mess becomes visible, and users need a way to clean it up (issue #40).

This is the third of three changes for issue #40. It builds on `favourites-search-filter`.

## What Changes

- `/favourites` gets a "Manage tags" dialog that lists all of the user's tags with how many links have each.
- A tag can be renamed. Renaming to the name of another tag (in any case) offers to merge the two: the links of both get the other tag, and the renamed one is removed.
- A tag can be deleted after a confirmation that says how many links have it. It is removed from all links, and the links are kept.
- Activating a tag's count in the dialog closes it and filters the tree by that tag.
- New endpoints: `GET /api/tags?all=true` (all tags with link counts), `PATCH /api/tags/{id}` (rename or merge) and `DELETE /api/tags/{id}`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: tags can be listed, renamed, merged and deleted on their own from the favourites page and the API.

## Impact

- Backend: `backend/internal/favourites` (`ListTags`, `MergeTag`, `DeleteTag`, building on the existing `RenameTag` and `DetachTag`), `backend/internal/httpapi` (routes, `tag_exists` error), Go tests.
- Frontend: `features/favourites` (new `ManageTagsDialog`, `api.ts`, button on `FavouritesPage`), locales (en, de), tests.
- No migration.
