# Proposal

## Why

The "Delete" text button added to the link edit dialog makes the button row too wide on phones, next to "Cancel" and "Save".

## What Changes

- The Delete button in the link edit dialog becomes an icon-only button (trash icon) in the danger color, with the accessible name and tooltip "Delete" / "Löschen".
- Its behavior, the confirmation and where it is shown stay the same.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: "Delete from the link edit dialog" now says the button is icon-only and red.

## Impact

- Frontend only: `frontend/src/components/buttonStyles.ts` (a new danger icon button style) and `frontend/src/features/favourites/NodeDialog.tsx`.
