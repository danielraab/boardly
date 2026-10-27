# Proposal

## Why

Newly saved favourites disappear into the tree, and there's no place on a dashboard to see what was added lately (issue #36). A widget that lists the newest links of a folder makes recent additions easy to find again.

## What Changes

- New content widget kind "Recent favourites" (`recent_favourites`) that lists the newest links inside a chosen folder and all its subfolders, newest first.
- Settings: folder (the tree's root means all favourites), a required maximum number of links (1–100, default 10), an optional "only the last N days" (1–365), an optional title, a size (S/M/L) and "Show on public dashboard".
- Each row shows the favicon (or a globe), the title (or the URL), the full URL cut off with "…", and when the link was added as a relative time ("3 days ago") with the full date as tooltip.
- "Added" means the link's creation date; imported links keep the date from their source.
- Empty state "No favourites" / "No favourites in the last N days"; deleted folder handled like the link list.
- An info button opens a link's details for the owner, like in the link list. No "Add favourite" button.

## Capabilities

### New Capabilities
- `recent-favourites-widget`: a dashboard widget that lists the most recently added links of a folder of the owner's favourites, limited by count and optionally by age.

### Modified Capabilities

None.

## Impact

- Backend: new widget kind in `backend/internal/widgets`, registered in `widgets.Register`; new favourites store query for the newest links of a subtree; the owner's favourites tree (`GET /api/favourites`) also carries each node's `createdAt` for the edit-mode preview. No migration.
- Frontend: new kind in `frontend/src/features/dashboards/widgets/recentFavourites`, added to the widget registry; en + de translations.
- Public dashboard lookup: the widget is left out for visitors unless "Show on public dashboard" is on.
