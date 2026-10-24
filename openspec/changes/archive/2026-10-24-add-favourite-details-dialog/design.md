# Design

## Context

In view mode, the three favourite widgets render from `widget.data`, which the server resolves per viewer. That data is deliberately thin: `linkData` has only `url`, `title` and `iconImageId`, and the explorer's `Store.Subtree` selects no description, preview or tags, so visitors never get them (favourite-explorer-widget and link-list-widget "on public dashboards"). Only the owner's `favouritesQueryOptions` (`GET /api/favourites`) has the full node. The explorer already enables that query for `isOwner`. The link widget and the link list enable it only in edit mode.

`NodeDialog` already edits a link given `{ node }` and invalidates `favouritesQueryKey` after a save.

Every row the info button joins today is a single `<a>` (or a `LinkView` `<a>`) that fills the row.

## Goals / Non-Goals

**Goals:**
- One shared dialog component for all three widgets.
- No server change, and no new data in the dashboard lookup response.

**Non-Goals:**
- Details for visitors, folders or plain URL links.
- An info button on `/favourites` (it already has an edit action per row).
- Editing inside the details dialog itself; it stays read-only and hands off to `NodeDialog`.

## Decisions

### Read the node from the owner's favourites, not from `widget.data`
The dialog takes a favourite ID and looks the node up with `findNode` in `useQuery(favouritesQueryOptions)`. The link widget decides whether to show the button with the same lookup: owner, not editing, the query has data, and `findNode(...)?.kind === "link"`. The explorer and the link list show buttons once the owner's tree has loaded. Their view data is resolved from that same tree, so every link they show exists in it. This keeps visitor responses unchanged, gives tags and fresh values, and makes "deleted favourite → no button" fall out naturally.
*Alternative:* add description, preview and tags to `widget.data`. Rejected because it would widen what the server resolves for every viewer, only to hide it again for visitors.

The link widget and the link list change `enabled: editing` to `enabled: editing || isOwner` (`useCurrentDashboard()?.isOwner`), like the explorer. For the owner, that is one extra cached request per dashboard, shared by all widgets.

### The info button is a sibling of the `<a>`, never inside it
A `<button>` inside an `<a>` is invalid HTML, and a click on it would also follow the link. Each row becomes a flex container `[<a class="min-w-0 flex-1">…</a><button/>]`. In "Tiles", the `<li>` gets `relative` and the button is absolutely placed top-right over the tile, still a sibling of the `<a>`. Tab order is link, then its info button.

### A small shared trigger plus the dialog
In `features/favourites/`:
- `FavouriteDetailsDialog({ favouriteId, onClose })`: a Base UI `Dialog` with `dialogBackdrop` / `dialogPopup`, like `NodeDialog`. It is open while `favouriteId` is set.
- `DetailsButton({ link, onClick, className, iconClassName })`: an `Info` lucide icon button, `iconButton` by default. `className` replaces the classes. The explorer and the tiles pass a compact padded button with a smaller icon (one step below the row icon), so it stays visible without making the rows taller.
- `useFavouriteLink(id)`: the link from the owner's tree, used by the dialog.

Each widget view keeps `const [detailsId, setDetailsId] = useState<string | null>(null)` and renders a single dialog, so a list of 50 links still mounts one dialog. `LinkView`, `ExplorerTree` and `LinkListView` get an optional `onDetails?: (id) => void` (for `LinkView`, `onDetails?: () => void`). When it is absent, nothing renders, which keeps them pure and their existing tests unchanged.

### Edit hands off to `NodeDialog`
"Edit" closes the details content and renders `NodeDialog` with `target={{ node }}`. On close or cancel, the details show again. On save, `NodeDialog` already invalidates the favourites. The details caller also passes `onSaved` that invalidates `dashboardViewQueryKey`, because view-mode widgets render the server's `widget.data` (title and URL) and need the refetch. This is the same pattern as the explorer's "Add favourite".
*Alternative:* stack `NodeDialog` on top of the open details. That works with Base UI nested dialogs, but swapping is simpler for focus handling and Escape.

### Dialog layout
```
┌ [favicon] <title or URL> ───────────── ✕ ┐
│ [ preview image, if any ]                │
│ https://go.dev/tour ↗  (link)            │
│ A tour of Go … | "No description" muted  │
│ [learning] [go]           (if any tags)  │
│                         [Edit]  [Open ↗] │
└──────────────────────────────────────────┘
```
The preview is an `<img src={imageUrl(previewImageId)}>` (from `components/ImageIcon`), like the preview in `NodeDialog`. "Open" is an `<a target="_blank" rel="noopener noreferrer">` styled as `primaryButton`. "Edit" is a `secondaryButton`. The description keeps line breaks (`whitespace-pre-line`) and wraps long words.

### i18n keys
Under `favourites.details`: `button` ("Details for {{title}}" / "Details zu {{title}}"), `open`, `edit`, `noDescription`, `close`, `tags`, `previewAlt` (alt text of the preview image).

## Risks / Trade-offs

- [The owner's view now fetches the whole favourites tree for link and link-list widgets] → The request is shared and cached across widgets, and the explorer already does it. The button simply appears once the data arrives, without blocking the widget.
- [Rows get a second focus stop per link] → This is acceptable for the owner only. It keeps the button keyboard-reachable instead of hover-only.
- [Small widgets (size S, narrow tiles) lose some width to the button] → The `<a>` is `min-w-0 flex-1` and truncates as before. On tiles, the button overlays the corner and takes no width.
