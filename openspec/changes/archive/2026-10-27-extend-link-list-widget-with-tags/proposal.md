# Proposal

## Why

A link list can only show the links of one folder. Tags cut across folders, so "everything tagged `monitoring`" can't be shown on a dashboard today (issue #37). The order of a link list is also fixed to the order of the favourites, which suits a hand-sorted folder but not a list collected across folders.

## What Changes

- A link list keeps its required folder and gets an optional tag filter ("Tag" / "Tag", one tag). With a tag, it shows only the links of its folder (and, with "Show items from subfolders", of its subfolders) that have that tag. All links with a tag are the root folder with subfolders and the tag.
- A link list gets a sort setting: "Favourites order" (default, as today), "Newest first" or "Alphabetical".
- A link list whose tag no longer exists shows "This tag no longer exists" / "Dieser „Tag“ existiert nicht mehr" to the owner and is left out for visitors, like a deleted folder.
- Visitors of a public dashboard get only the links shown: no tag names, folder names or dates.
- Saved link lists keep working unchanged: no tag filter and favourites order.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `link-list-widget`: settings gain a tag filter and a sort; the links shown, the missing-folder rule and the public-dashboard rule cover the tag; a new requirement defines the sort orders.

## Impact

- Backend: `internal/widgets/linklist.go` (config, validation, loading links with tags and dates, filtering, sorting) and its tests; a link-list loader in `internal/favourites` that returns links with their tag IDs and creation times, plus a tag-exists check; `golang.org/x/text` becomes a direct dependency for collation.
- Frontend: `features/dashboards/widgets/linkList/` (config type, settings form, edit-mode preview, messages) and its tests; English and German translations.
- No database migration; widget configs are stored as JSON. The favourite explorer's data is unchanged.
