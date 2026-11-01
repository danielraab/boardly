# Proposal

## Why

On a phone, the "Add" button with its label takes room from the station name and details in the
radio station search results, so names get cut off early.

## What Changes

- Below the `sm` breakpoint, a search result's add button SHALL show only the plus icon (or the
  busy spinner), without the "Add" / "Hinzufügen" label. From `sm` up it keeps the label.
- The button's accessible name ("Add <station>") stays the same on all widths.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `media-library`: "Station search" says the add button is icon only on small screens.

## Impact

- Frontend: `frontend/src/features/media/AddStationDialog.tsx` (`ResultRow`).
- No backend, API, translation or configuration changes.
