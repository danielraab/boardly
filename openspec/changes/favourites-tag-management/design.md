# Design

## Context

`favourites.Store` already has `RenameTag` (returns `ErrTagExists` when another tag has the name, ignoring case), `DetachTag` (removes a tag from given links and cleans up unused tags) and `TagLinkCount`. They are used by the Karakeep and Linkwarden APIs, which map `ErrTagExists` themselves. `GET /api/tags?q=` returns up to 10 suggestions by prefix. The favourites tree carries tag *names* on links, not IDs.

## Goals / Non-Goals

**Goals:**
- Reuse the existing store methods. Add only listing, merge and full delete.

**Non-Goals:**
- Tag colours, descriptions or tags on folders.
- A tag page outside `/favourites`.

## Decisions

### Listing through `GET /api/tags?all=true`
The dialog needs tag IDs and counts, which the tree doesn't provide. `ListTags(userID)` runs one grouped query over `tags` and `favourite_node_tags`. Reusing the existing path with a flag keeps one tags resource. A missing or false `all` keeps the suggestion behaviour unchanged.
*Alternative:* `GET /api/tags/all`. It reads like a sub-resource called "all". Rejected.

### Merge in the store
`MergeTag(userID, sourceID, targetID)` runs in one transaction: `INSERT OR IGNORE INTO favourite_node_tags (node_id, tag_id) SELECT node_id, target FROM favourite_node_tags WHERE tag_id = source`, touches `updated_at` of the affected links, deletes the source tag (its rows go with the foreign key) and returns the target. The handler first calls `RenameTag`. On `ErrTagExists` with `merge: true`, it looks up the conflicting tag by `name_key` and calls `MergeTag`. Otherwise it responds 409 `tag_exists`.

### Delete as detach-all
`DeleteTag(userID, tagID)` deletes all `favourite_node_tags` rows of the tag, touches the links and deletes the tag, in one transaction. `DetachTag` with every link would work too, but needs the IDs first.

### Dialog
`ManageTagsDialog` loads `GET /api/tags?all=true` when opened. Rename is inline (the name becomes an input, Enter saves, Escape cancels). A 409 switches the row to the merge question. Successful changes invalidate the tag list and the favourites query, so row tags, the tag filter and suggestions update. Activating a count navigates to `/favourites?tags=<name>` and closes the dialog.

## Risks / Trade-offs

- [Renaming a tag in the dialog while a tag filter uses the old name leaves the filter pointing at a name that no longer exists] → after a rename or merge, the page replaces the old name in the URL's `tags` with the new one, and after a delete it drops it.
- [Apps via Karakeep/Linkwarden see tag IDs change on merge] → the source tag ID disappears, as with any deleted tag. Those APIs already handle missing tags.
