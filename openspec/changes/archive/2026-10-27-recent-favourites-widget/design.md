# Design

## Context

Content widget kinds live in `backend/internal/widgets` as a `dashboards.Kind` with `Normalize` (validates the config when a layout is saved) and `Resolve` (loads data per viewer on lookup). The explorer and the link list both resolve a folder via `loadFolder` → `favourites.Store.Subtree`, a nested tree without timestamps. `favourite_nodes.created_at` is stored as fixed-width RFC 3339 UTC text (`db.TimeFormat`), so string comparison in SQL is chronological. Imports write the source's creation date into `created_at` (`favourites/import.go`). `favourites/subtree.go` already has `subtreeCTE`, a recursive CTE that selects a user's folder and every node below it.

On the frontend, each kind is a folder under `frontend/src/features/dashboards/widgets/` that exports a `WidgetKind` (view, settings form, default config) listed in `registry.ts`. The link list's `LinkListView` already renders the "List" layout rows, the info button and the S/M/L sizes.

## Goals / Non-Goals

**Goals:**
- One SQL query per widget that returns only the links to show, already sorted and limited.
- Reuse the folder picker, sizes, title heading, missing-folder message and details button of the link list.

**Non-Goals:**
- A separate "added at" timestamp distinct from `created_at`.
- An index on `created_at`; see Decisions.
- Live updating of the relative times while the page stays open.
- Tag filters, layouts other than a list, or an "Add favourite" button.

## Decisions

**Config shape.** `{folderId, limit, days, title, size, showIfPublic}`, where `limit` is a required int (1–100) and `days` is `*int` (nil or 1–365). Decoded strictly like the other kinds. A single config with an always-present limit and an optional age filter, rather than a "mode" switch, gives a bounded result in every case and lets both limits combine.

**New store query instead of `loadFolder`.** Add `favourites.Store.RecentLinks(ctx, userID, folderID string, since time.Time, limit int) ([]RecentLink, error)`. It reuses `subtreeCTE` and selects `kind = 'link'` nodes of the subtree with `created_at >= ?` (when `since` is non-zero), `ORDER BY created_at DESC, id DESC LIMIT ?`. It returns `ErrNotFound` when the user has no such folder, like `Subtree`, so the kind can return `missingFolder` like the others. Loading the whole subtree and sorting in Go would work too, but it reads and sends far more than needed for a "top 10" of a large tree. The ID tie-break gives a stable order for imports that share a timestamp.

**No index.** The query starts from the folder's subtree via the recursive CTE, which walks `favourite_nodes_parent`; an index on `(user_id, created_at)` wouldn't be used for a scoped walk. Personal trees are at most a few thousand nodes, so sorting the subtree is cheap.

**Rolling window, computed on the server.** `since = now − days × 24h` in UTC at resolve time. This avoids timezones and calendar-day boundaries; "last 7 days" means the last 168 hours. The widget kind takes `now` from a function it is given, so tests can fix it.

**Response data.** `{links: [{id, title, url, iconImageId?, createdAt}]}` with `createdAt` in RFC 3339. Favicons are set through `images.Favicons` like for other links (wrap the results in `favourites.Node`s or add a small helper to resolve icons for a flat list, whichever `SetIcons` supports with less code). Folder names, tags and descriptions are never included, so the visitor response is the same as the owner's.

**Edit-mode preview from the owner's tree.** In edit mode the link list and explorer render their draft from the owner's favourites tree (`GET /api/favourites`), since a changed or new widget has no server data yet. That tree has no timestamps, so `Store.Tree` also sets `createdAt` on every node (`Node.CreatedAt`, omitted when unset, so the widget loaders and visitor responses don't get it). The draft then applies the same filter and order in the browser: links under the folder, `createdAt >= now − days×24h`, newest first, then ID descending, limited. Alternatives: showing the last saved server data in edit mode (wrong as soon as a setting changes) or a preview endpoint (more API for a preview).

**Relative time on the client.** `Intl.RelativeTimeFormat` in the i18n language with `numeric: "auto"` (gives "yesterday" / "gestern"), choosing the largest fitting unit (seconds → minutes → hours → days → months → years), with `Intl.DateTimeFormat` (date + time) for the tooltip. Computed at render time; no timer.

**Frontend reuse.** The new `recentFavourites` kind reuses the folder picker and `WidgetSettingsForm`, and the row markup of the link list's "List" layout plus a right-aligned date. If the row is not easy to share as is, extract a small shared row component instead of copying it.

## Risks / Trade-offs

- [Imported links with old dates don't count as "recent"] → Intended and stated in the spec; users who want them can drop the day limit.
- [Visitors with "Show on public dashboard" on see the newest links of the whole folder, possibly the root] → The settings dialog says so next to the flag; the flag is off by default.
- [Relative times go stale on a page left open for days] → Acceptable for a start page that is reloaded often; no timer for now.
