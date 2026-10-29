# Design

## Context

`NoteDialog` (`widgets/note/NoteDialog.tsx`) is a Base UI dialog with no history entry. Each of the note, recent notes, note list and selected notes widgets keeps its own open state and renders its own `NoteDialog`, always with `onChanged`/`onDeleted` refreshing the dashboard view query. `DashboardView` already keeps edit mode in the search (`?edit=true`, `validateDashboardSearch`), shared by `/` and `/$userSlug/$dashboardSlug`. The editor's autosave flushes on unmount.

## Goals / Non-Goals

**Goals:**
- Back closes the modal on every platform, using the router only.
- No history entry is left behind when the modal is closed by its own controls.

**Non-Goals:**
- History entries for other dialogs (settings, widget settings, delete confirmation).
- Opening a note of another user.

## Decisions

**Search parameter.** `DashboardSearch` becomes `{edit?: true; note?: string}`; `validateDashboardSearch` keeps `note` when it is a non-empty string of at most 64 characters. Alternative: a route `/$user/$dashboard/notes/$id`, which needs a second route for `/` and remounts nothing useful.

**One modal per dashboard.** `DashboardView` renders a `DashboardNoteDialog` outside edit mode for the owner. Widgets call `useOpenNote()` → `navigate({to: ".", search: prev => ({...prev, note: id}), state: {noteOpened: true}})`. This removes the per-widget open state and dialogs, and two widgets showing the same note can't open two modals. The last note ID is kept in state, so the modal can play its closing animation after `note` is gone.

**Closing.** If the current history entry carries `noteOpened` (we pushed it), closing calls `router.history.back()`, which is exactly what the back button does. Otherwise (reload, link) it navigates with `replace` and without `note`. Alternative: always `replace`, which leaves a duplicate dashboard entry so the next back seems to do nothing.

**Ignoring the parameter.** Visitors (`!view.isOwner`) and a note query that fails with 404 replace the address without `note`. Edit mode doesn't render the modal, and `setEditing` drops `note`.

**Accessible name.** `NoteDialog` no longer takes `name`; it reads the note query (the same key the editor uses) and labels the popup with its display title, or "Untitled note".

## Risks / Trade-offs

- [`history.back()` when the entry was pushed but the user navigated in between] → The flag lives on the entry itself, so it's only set on the entry we pushed.
- [History state is typed as empty by the router] → A small typed accessor around `location.state`.
