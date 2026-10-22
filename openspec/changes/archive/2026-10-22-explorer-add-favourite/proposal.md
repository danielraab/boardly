## Why

To add a link to a folder shown in a favourite explorer, the owner has to leave the dashboard, go to `/favourites`, find the folder and add it there. An add button right on the explorer saves that detour (issue #24).

## What Changes

- The favourite explorer gets an "Add favourite" button directly left of its expand/collapse-all button, in the same size. It is only shown to the signed-in owner of the dashboard when viewing it, not in edit mode and never to visitors.
- Activating it opens the existing "New link" dialog with the explorer's root folder preselected; another folder can be chosen.
- After saving, the explorer shows the new link without a reload.
- No button when the explorer's root folder no longer exists.

## Capabilities

### New Capabilities

### Modified Capabilities
- `favourite-explorer-widget`: new requirement for adding a favourite from the explorer.

## Impact

- Frontend only: `frontend/src/features/dashboards/widgets/favouriteExplorer/` (widget view and tree), translations (en, de), tests.
- Reuses `NodeDialog` from `features/favourites` and the existing `POST /api/favourites` endpoint. No backend or API change.
