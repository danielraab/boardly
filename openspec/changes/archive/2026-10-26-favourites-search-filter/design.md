# Design

## Context

`FavouritesPage` loads the whole tree, with tags, through `favouritesQueryOptions` and renders it in `TreeView`. `TreeView` keeps a `collapsed` set in component state (everything starts expanded, nothing is remembered) and turns it into rows with `flatten(root, collapsed)`. Drag and drop (`pointerSlot`, `keyboardSlots`, `indicatorFor`) computes positions from the same `root` and `collapsed`, so it assumes it sees the real tree.

The explorer widget keeps its expanded set in `localStorage` through `useExpanded(widgetId)` (`boardly.explorer.<widgetId>`) and already has an expand/collapse-all button. Its title is rendered in `ExplorerTree`. `index.tsx` knows whether the viewer is the owner, whether it is in edit mode, and (for the owner) the full favourites tree.

The `/favourites` route has no search params yet. Other routes (`settings`, `share`, `dashboards`) use `validateSearch`.

## Goals / Non-Goals

**Goals:**
- Filtering, highlighting and counts as pure functions in `tree.ts`, covered by unit tests.
- Leave the drag-and-drop code unchanged. It only ever runs on the unfiltered tree.

**Non-Goals:**
- Server-side search or a search API.
- Search in the explorer widget.
- Selection and bulk actions (`favourites-bulk-edit`), tag rename/merge (`favourites-tag-management`).

## Decisions

### Filtering produces a pruned tree
`filterTree(root, { terms, tags })` returns a copy of the tree that contains only shown nodes, or `null` when nothing matches. The rule from the spec is applied in one walk that carries the folder names on the path: a link is kept when it has every chosen tag and every term occurs in its title, URL, a tag name or a folder name on its path. A folder is kept when it has a kept descendant, or, with no tag filter, when its own name contains every term (then its whole subtree is kept). The page renders the pruned tree with the existing `flatten`, so the row rendering stays the same.
*Alternative:* a `visible` set checked inside `flatten`. It mixes two concerns into `flatten`, and counts and the empty state would need a second pass anyway.

### Two collapse sets
The remembered `collapsed` set is used only without a filter. While a filter is active, the page uses a separate in-memory set that starts empty, so all shown folders are expanded, and that is reset whenever the filter changes. Toggling and "Expand/Collapse all" act on whichever set is in use. This gives "the filter doesn't change the remembered state" without special cases.

### Shared stored-set hook
`useExpanded`'s storage logic moves to a generic `useStoredIdSet(key)` in `lib/`, used by the widget (`boardly.explorer.<widgetId>`, unchanged key, so existing state is kept) and the page (`boardly.favourites.collapsed.<userId>`). The page stores *collapsed* IDs because its default is expanded. The user ID is part of the key because several accounts can share a browser. Blocked storage falls back to in-memory state, as it does today.

### Filter state in the URL
`routes/favourites.tsx` gets `validateSearch` for `q?: string`, `tags?: string[]` and `folder?: string`. Typing calls `navigate({ search, replace: true })`, so keystrokes don't fill the history. Choosing or removing a tag pushes a history entry. Empty values are dropped from the URL. The filter is cheap (one walk over an in-memory tree), so there is no debounce.

### Tag filter control
The available tags and counts come from a `tagCounts(root)` walk, not from `/api/tags`, which only returns 10 by prefix. The control is a Base UI menu listing the tags as checkbox items with their counts (scrolling when long), plus removable chips below the search row. A text filter inside the menu was left out: a menu's typeahead already jumps to a tag by its first letters. Tag chips on rows become buttons that add their tag.

### Moving turned off while filtered
`TreeRow` gets a `movable` flag. When it is off, the drag handle is replaced by a spacer of the same width, so rows don't shift, and `DndContext` gets no draggables. Because nothing can be dragged, `pointerSlot` and the other helpers never see the pruned tree.

### Highlighting
`highlight(text, terms)` splits a name into marked and unmarked parts (case-insensitive, overlapping matches merged) and the row renders `<mark>` for marked parts. Only displayed names are highlighted. A match in a URL or tag is not.

### Counts
`linkCounts(tree)` returns a map from folder ID to the number of links below it, computed on the tree being shown (the full tree, or the pruned one while filtering). It is computed once per render with `useMemo`, not per row.

### Folder deep link
When the tree has loaded and `folder` names a folder, the page removes that folder and its ancestors from the remembered `collapsed` set, scrolls the row into view (`scrollIntoView({ block: "center" })`), adds a highlight class for about two seconds, and then removes `folder` from the URL with `replace`. Removing it keeps later collapsing and reloads from re-expanding the folder. Unknown IDs and link IDs are dropped the same way, with no other effect.

### Explorer title link
`ExplorerTree` gets an optional `titleLink` (search params for `/favourites`). With it, the heading's text is a TanStack `Link`. `index.tsx` passes it only when `isOwner && !editing && folder?.kind === "folder"`, with `{}` when the root folder is the tree's root and `{ folder: id }` otherwise. The heading element and level stay the same, and the link sits inside it.

## Risks / Trade-offs

- [Very large trees make filtering on every keystroke slow] → one linear walk per change, and trees are small (hundreds to a few thousand nodes). If needed later, wrap the input in `useDeferredValue`.
- [`/` conflicts with typing in other inputs] → the shortcut is ignored when focus is in an input, textarea, select or contenteditable, or when a dialog is open.
- [Stale IDs pile up in the stored collapsed set after deletes] → harmless and small. Prune to existing folder IDs when the tree loads.
- [Search params change the generated route tree] → regenerate `routeTree.gen.ts` as part of the build, which the dev server already does.
