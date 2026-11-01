# Proposal

## Why

When saving a link, users can't tell whether they already keep links from the same site, so related links end up scattered or duplicated across folders. A hint with the number of favourites on the same domain, linking straight to them, makes that visible while the link is being entered.

## What Changes

- New **domain filter** on `/favourites`: a `domain` search param (`/favourites?domain=github.com`) that shows only links whose domain matches exactly. The domain is the URL's host in lower case without a leading `www.`; subdomains are different domains. It is shown as a removable chip next to the tag chips, applies together with the search text and tags, is kept in the URL, and is cleared by "Clear filters". Like the other filters, it turns off moving.
- New **domain hint** under the URL field wherever a favourite link is entered: the "New link" / "Edit link" dialog (on `/favourites`, in the favourite details, in the favourite explorer widget and from "New favourite…" in the link widget settings) and the "Save link" page (`/share`). For a valid `http`/`https` URL it says how many of the user's favourites have the same domain, e.g. "3 favourites on github.com", and links to `/favourites?domain=github.com` in a new tab so the form isn't lost. When editing, the link itself isn't counted ("2 other favourites…"). No hint when the count is 0 or the URL is invalid.
- The same hint under the **URL** source field of the link widget settings, so the owner sees that a favourite on that domain may already exist.
- No backend change: the count uses the favourites tree the client already loads.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: adds the domain filter on `/favourites` and the domain hint when entering a link's URL; the filter state, clearing and "no moving while filtered" requirements include the domain filter.
- `link-widget`: the "URL" source field shows the domain hint.

## Impact

- Frontend only:
  - `frontend/src/features/favourites/tree.ts`: domain helper, domain count, domain in `TreeFilter`/`filterTree`.
  - `frontend/src/routes/favourites.tsx` and `FavouritesPage.tsx`: `domain` search param, chip, clearing.
  - `frontend/src/features/favourites/LinkFields.tsx` (used by `NodeDialog` and `SharePage`) and `NodeDialog.tsx`: the hint, with the edited link excluded.
  - `frontend/src/features/dashboards/widgets/link/LinkSettings.tsx`: the hint under the URL field.
  - `frontend/src/locales/en.json`, `de.json`: new strings.
  - Tests for the helper, the filter and the hint.
- No API, database or dependency changes.
