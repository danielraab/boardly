# Design

## Context

The link list kind (`backend/internal/widgets/linklist.go`) stores `{folderId, title, includeSubfolders, layout, size, showIfPublic}`, decoded strictly. `Resolve` loads the folder via `loadFolder` → `favourites.Store.Subtree` (cached per lookup, favicons set) and flattens its links. `Subtree` returns neither tags nor `created_at`, and the favourite explorer sends the same nodes to visitors, so adding fields there would leak them.

Tags live in `tags` (ID, name unique per user ignoring case) and `favourite_node_tags`. Renaming keeps a tag's ID; merging and deleting remove it, and a tag on no link is removed automatically. `Store.Tree` (used by `GET /api/favourites` and the edit-mode previews) returns tag names and `createdAt` per node, but no tag IDs; `GET /api/tags?all=true` (`listTags`, `tagUsageQueryKey`) returns every tag with its ID. The recent favourites widget already orders by `created_at DESC, id DESC`.

## Goals / Non-Goals

**Goals:**
- Filter and sort on the server, so the response carries only the links shown, in their final order, with no tags or dates.
- Leave the favourite explorer's loader and response untouched.
- Saved link lists behave as before without a migration.

**Non-Goals:**
- Several tags, or tag logic (and/or).
- Following a tag through a merge or through being removed and re-created.
- Sorting in the viewer's language on the server.

## Decisions

**Config shape.** Add `tagId` (`*string`; nil or empty means no filter, normalized to nil; at most `maxIDLength`) and `sort` (`"favourites"`, `"newest"` or `"alphabetical"`; empty normalized to `"favourites"`). Old configs have neither field and decode to "no filter, favourites order". Alternative considered: a separate tag source instead of a filter — rejected in exploration; a filter on the folder is more general (tag within part of the tree) and needs no either/or validation.

**Tag by ID.** Renaming the tag keeps the widget working. A merged, deleted or auto-removed tag makes it "missing"; the owner picks the tag again. Storing the name was considered (survives re-creation) but breaks on rename. Like the folder, a tag that doesn't exist when the layout is saved is not rejected.

**Link-list loader.** Add `favourites.Store.FolderLinks(ctx, userID, folderID string, subfolders bool) ([]FolderLink, error)` returning the links of the folder (and, with `subfolders`, of every subfolder) depth first in favourites order, each with ID, title, URL, `createdAt` and tag IDs; `ErrNotFound` for an unknown folder. Implementation: the same recursive walk as `Subtree` with `created_at`, plus one query for the tag IDs of those links, flattened in Go. Favicons are set by origin as for other links (via `images.Favicons` on wrapper nodes or a flat helper, whichever needs less code). Add `Store.TagExists(ctx, userID, tagID) (bool, error)`. Cache both per lookup like `loadFolder`. Matching by tag ID needs no name lookup.

**Resolve order.** Visitor without the public flag → left out. Folder missing → `{missing: true}` for the owner, left out for visitors. Tag set and missing → `{missing: "tag"}` for the owner, left out for visitors. Otherwise filter by tag, sort, and return `{links}` as today. The folder check comes first so a list with both missing shows the folder message. Data shape: `missing` stays `true` for the folder (existing frontend check) and becomes `"tag"` for the tag, so the view can choose the message without the config.

**Sorting.**
- `favourites`: the depth-first order (unchanged).
- `newest`: `createdAt` descending, then ID descending — the same tie-break as the recent favourites widget.
- `alphabetical`: by the displayed text (title, or URL when there is no title), using `golang.org/x/text/collate` with the root locale and ignoring case and accents for the primary comparison, then URL, then ID. A plain lowercase comparison would sort "Ärzte" after "Zoo".

With subfolders on, sorting applies to the whole flattened list, so links are no longer grouped by subfolder.

**Frontend.** `LinkListConfig` gains `tagId: string | null` and `sort: LinkListSort`; a config without them is treated as no filter and `favourites`. The settings dialog adds, after "Show items from subfolders", a "Tag" select with "No tag filter" first and then the owner's tags from `listTags`, sorted by name, and a "Sort" select. A chosen tag that no longer exists shows the missing hint and falls back to "No tag filter" in the select, like a deleted folder. The edit-mode preview filters the owner's tree by the tag's name (tag ID → name via `listTags`; a missing tag gives the missing state) and sorts with `createdAt` from the tree and `Intl.Collator(undefined, { sensitivity: "base" })` for alphabetical, with the same tie-breaks. `LinkListView` shows the tag messages for `missing: "tag"` and for an empty list with a tag filter.

## Risks / Trade-offs

- [A tag removed because its last link lost it, and re-added later, gets a new ID] → The owner sees "This tag no longer exists" and picks the tag again.
- [Edit-mode preview collation can differ slightly from the server's] → Only in edge cases and only in edit mode; the saved widget shows the server's order.
- [Visitors could infer the tag from the shown links] → Acceptable; the flag is off by default and the response has no tag name.
