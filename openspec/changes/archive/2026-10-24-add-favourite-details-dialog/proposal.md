# Proposal

## Why

Links now have a description, a preview image and tags, but on a dashboard the owner only sees a favourite's icon, title and URL. To look at the rest, or to change it, they have to leave the dashboard and find the link on `/favourites`.

## What Changes

- A new read-only "details" dialog for a favourite link shows its website icon and title, its preview image, its URL (a link that opens in a new tab), its description ("No description" in muted text when it has none) and its tags. It has an "Open" button that opens the link in a new tab and an "Edit" button that opens the existing link edit dialog. After a save, the details and the widget show the new values.
- An info button (ⓘ) next to a favourite link opens the dialog in:
  - the link widget, when its source is a favourite that still exists (not for plain URL links);
  - the favourite explorer, on each link row (not on folders), always visible and smaller than the row's text;
  - the link list, in all three layouts: beside the row in "List" and "Compact", in the tile's top-right corner in "Tiles".
- The info button is shown only to the signed-in owner of the dashboard, and never in edit mode. Visitors see no change, and the dashboard lookup response stays the same.

## Capabilities

### New Capabilities

- `favourite-details`: the details dialog of a favourite link: what it shows, its "Open" and "Edit" actions, and who can open it.

### Modified Capabilities

- `link-widget`: a favourite link widget gets an info button that opens the details for the owner.
- `favourite-explorer-widget`: link rows of the explorer get an info button that opens the details for the owner.
- `link-list-widget`: links of a link list get an info button, in every layout, that opens the details for the owner.

## Impact

- Frontend: new `FavouriteDetailsDialog` in `features/favourites/`; `dashboards/widgets/link` (`index.tsx`, `LinkView.tsx`), `dashboards/widgets/favouriteExplorer` (`index.tsx`, `ExplorerTree.tsx`), `dashboards/widgets/linkList` (`index.tsx`, `LinkListView.tsx`); locales (en, de); their tests.
- The link widget and the link list load the owner's favourites in view mode as well (the explorer already does). Nothing more is fetched for visitors.
- Backend: no change. No API, data or migration changes.
