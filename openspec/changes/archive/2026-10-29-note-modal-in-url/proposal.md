# Proposal

## Why

The note editor modal on dashboards doesn't add a browser history entry, so the browser's or Android's back button leaves the dashboard instead of closing the modal. On phones, back is the natural way to close it.

## What Changes

- Opening a note from a dashboard widget (note, recent notes, note list, selected notes) adds `?note=<id>` to the dashboard's address. Back removes it and closes the modal; closing the modal with its button, Escape or a click outside goes back the same way.
- Reloading or opening an address with `?note=<id>` shows the dashboard with the modal open. Visitors, edit mode and notes that don't exist ignore the parameter.
- One modal per dashboard replaces the one each widget had.

## Capabilities

### New Capabilities

### Modified Capabilities
- `note-widget`: the note editor modal of dashboards is part of the address and closes with back.

## Impact

- Frontend: `DashboardSearch` gets `note`; `DashboardView` renders the one `NoteDialog` from it; the four note widgets only navigate to open a note; `NoteDialog` takes its accessible name from the loaded note. No backend change.
