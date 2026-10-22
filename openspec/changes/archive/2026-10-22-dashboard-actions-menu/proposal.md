# Proposal

## Why

The "Edit" button next to a dashboard's title only opens the layout editor, but its name suggests it edits the whole dashboard. The dashboard's settings (title, slug, default, public, columns) can only be changed on `/dashboards`, so owners have to leave the dashboard they are looking at to change them.

## What Changes

- Replace the owner's "Edit" button next to the dashboard title with an actions menu: an icon-only button with three dots ("Dashboard actions" / "Dashboard-Aktionen") that opens a menu.
- The menu has two items: "Edit layout" / "Layout bearbeiten", which enters edit mode as "Edit" did, and "Settings" / "Einstellungen", which opens the same dashboard dialog as the edit action on `/dashboards`.
- After saving the settings from the dashboard page, the page follows the dashboard: when the slug changed, it navigates to the new address. When the page is `/` and the default flag was removed, it navigates to the dashboard's own address.
- The menu is shown in view mode only. Edit mode keeps its "Save and exit" and ✕ buttons.
- The empty-dashboard hint for the owner points to "Edit layout" instead of "Edit".
- `GET /api/u/{userSlug}/{dashboardSlug}` also returns `isDefault` and `isPublic` for the owner, so the dialog can start from the dashboard's values.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dashboard-editing`: the "Edit" button becomes the dashboard actions menu with "Edit layout" and "Settings". Edit mode, the toolbar and the empty-dashboard hint refer to "Edit layout".
- `dashboards`: the dashboard dialog can also be opened from the dashboard page, which follows a changed slug or a removed default flag. The dashboard view API returns `isDefault` and `isPublic` to the owner.

## Impact

- Backend: `backend/internal/dashboards` (`View` and `Lookup`) and its tests.
- Frontend: `DashboardView.tsx` (the actions menu), `DashboardDialog.tsx` (a callback after a successful save), `api.ts` (`DashboardViewData`), en/de locales, and the dashboard editor and view tests.
- No new dependencies. The menu uses the Base UI menu like the other header menus.
