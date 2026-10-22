## Context

The explorer (`widgets/favouriteExplorer`) renders `ExplorerTree` from the server's widget data in view mode and from the owner's favourites query in edit mode. `NodeDialog` already creates links, and with `chooseFolder` lets the user pick the folder (used by the link widget's settings). The dashboard page publishes the shown `DashboardViewData`, including `isOwner`, through `useCurrentDashboard()`.

## Goals / Non-Goals

**Goals:**
- Owner can add a link to the explorer's folder (or another folder) from the dashboard.
- The explorer updates right after saving.

**Non-Goals:**
- Adding folders, editing or deleting nodes from the explorer.
- Adding to a subfolder via a per-row button.

## Decisions

- **Who sees the button**: read `isOwner` from `useCurrentDashboard()` in the widget's `View` rather than threading a new prop through `DashboardGrid`/`SectionGrid`. The context is already set by `DashboardView`, and only the owner's lookup can be followed by a successful create anyway. Hidden in edit mode, because the edit-mode tree is a preview and widget clicks belong to the editor.
- **Dialog**: reuse `NodeDialog` with target `{ kind: "link", parent: <root folder node> }` and `chooseFolder`, so the URL, title, tags and folder fields, validation and errors match `/favourites` and the link widget. The root folder node comes from `findNode` on the favourites query, which the view enables for the owner.
- **Refresh**: in view mode the tree comes from the dashboard lookup, so on save the view also invalidates `dashboardViewQueryKey` (NodeDialog already invalidates favourites and tags).
- **Placement**: `ExplorerTree` gets an optional `onAdd`; when set, a `Plus` icon button with the accessible name "Add favourite" / "Favorit hinzufügen" is rendered directly left of the expand-all button, with the same `iconButton` style and `size-4` icon (on the title line when there is a title). Keeps the tree component free of data fetching.

## Risks / Trade-offs

- [Extra favourites fetch for the owner in view mode] → It's the owner's own tree and cached by React Query; acceptable.
- [Saving to another folder outside the explorer's root] → The explorer simply doesn't show it; that is what the user chose.
