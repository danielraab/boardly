# Proposal

## Why

On a dashboard, the only way to delete a favourite link is to leave for `/favourites`, find the link and delete it there. The edit dialog of a link, reached from the tree or from a widget's details, is where the user already looks at a link to decide what to do with it, so deleting should be possible right there.

## What Changes

- The edit dialog of an existing link gets a "Delete" / "Löschen" button, set apart from Cancel and Save.
- The button opens the same confirmation as deleting the link from the tree. Confirming deletes the link and closes both dialogs, and cancelling returns to the edit dialog with nothing lost.
- On `/favourites`, the tree shows the link removed. On a dashboard, the details dialog closes too and the dashboard's widgets refresh. Widgets that showed the link behave as their specs already describe for deleted favourites.
- The dialogs for new links, new folders and editing a folder get no Delete button.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: "Link edit dialog" gains a Delete action with the usual confirmation.
- `favourite-details`: "Open and edit from the details" describes what happens on a dashboard when the link is deleted from the edit dialog.

## Impact

- Frontend only: `frontend/src/features/favourites/NodeDialog.tsx` (Delete button, reuses `DeleteDialog`), `FavouriteDetails.tsx` (close and refresh after a delete), translations in `frontend/src/locales/en.json` and `de.json`, plus tests.
- No backend or API changes, because the existing `DELETE` favourite endpoint is used.
