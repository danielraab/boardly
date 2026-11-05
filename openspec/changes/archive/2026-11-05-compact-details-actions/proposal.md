# Proposal

## Why

The details dialog of a favourite has three text buttons in its footer: "Edit", "Offline copy" and "Open". In German they read "Bearbeiten", "Offline-Kopie" and "Öffnen". Together they are wider than a narrow dialog, so the row overflows or squeezes the buttons on phones.

## What Changes

- The "Offline copy" / "Offline-Kopie" button in the details dialog becomes a square icon-only button with the archive icon. Its accessible name stays "Offline copy" / "Offline-Kopie", and it shows that name as a tooltip.
- It stays between "Edit" and "Open", and it still opens the offline copy dialog in place of the details.
- If the buttons still don't fit on one line, the footer row wraps instead of overflowing.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourite-details`: the requirement "Offline copy button in the details" now says the button is icon-only, with its name as the accessible name and the tooltip.

## Impact

- Frontend only: `frontend/src/features/favourites/FavouriteDetails.tsx` and its tests. It reuses `secondaryIconButton` from `frontend/src/components/buttonStyles.ts`.
- Backend, API and OpenAPI documents are unaffected.
- Export and import are unaffected: no user data changes.
