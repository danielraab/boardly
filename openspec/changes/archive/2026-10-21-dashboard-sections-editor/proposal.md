# Proposal

## Why

Dashboards exist and have addresses, but every dashboard page still shows an "empty dashboard" placeholder. Owners need a way to build the layout on the page itself. This change adds the editor and the first, structural widget: the **section**. Every later widget (links, navigation trees) lives inside sections.

## What Changes

- **Edit mode**: a dashboard's owner sees an "Edit" button next to the dashboard title. It adds `?edit` to the URL and shows the editing controls. In edit mode, the title area shows "Save and exit" and a ✕ (cancel) button instead. Visitors never see the controls, even with `?edit`.
- **Draft editing**: changes in edit mode are kept in a local draft. "Save and exit" saves the whole layout in one request. ✕ drops the draft. Cancelling, navigating away or reloading with unsaved changes asks for confirmation first.
- **Conflict check**: a save made from an outdated layout (saved in the meantime from another tab or device) is rejected with 409. Nothing is overwritten, and the page says that the dashboard was changed elsewhere.
- **Sections**: the top level of a dashboard holds only sections. A section has a **width** in page columns, an optional **title**, a **border** (on by default; borderless is possible) and a **height** (`auto` by default, or a fixed number of rows of `4rem`, with a scrollbar when the content is taller). Sections can contain sections, up to 5 levels deep.
- **Widths count against the page**. A section never renders wider than its parent: the rendered width is `min(width, parent columns)`, and the stored value is kept.
- **Edit controls**: in edit mode, every section shows a dashed outline, a drag handle and an edit button. There is an "Add section" button at the top level and inside every section that is less than 5 levels deep. Sections are reordered and moved between parents by dragging, also with the keyboard. A drop that would nest deeper than 5 levels is blocked and shows a message. Deleting a section asks for confirmation and deletes everything inside it.
- **New dashboards** start empty. Sections can only be added in edit mode.
- **Column settings per dashboard** in the existing dashboard dialog: desktop columns (default 6, 1–12), mobile columns (default 2, 1 up to the desktop count) and the mobile breakpoint (640, 768 (default) or 1024 px). Below the breakpoint, the page uses the mobile column count and sections wrap.
- New frontend dependency `@dnd-kit/sortable`.
- README: the 12-column grid becomes the configurable grid (6 columns by default), the section description changes to width, title, border and height, and edit mode is described.

Not part of this change: link and navigation-tree widgets, S/M/L size presets for the row unit, undo, and live collaboration.

## Capabilities

### New Capabilities

- `dashboard-layout`: what a dashboard contains and how it renders. The section widget and its settings, the column grid with desktop/mobile counts and the breakpoint, width clamping, the nesting limit, and the layout data in the dashboard lookup.
- `dashboard-editing`: how an owner changes the layout. Edit mode and its URL parameter, the draft with save, cancel and the unsaved-changes guard, add/edit/delete/drag of sections with the depth check, and the layout save API with its conflict check.

### Modified Capabilities

- `dashboards`: the dashboard entity, API and create/edit dialog gain the desktop column count, the mobile column count and the mobile breakpoint. The dashboard address now shows the dashboard's sections instead of a placeholder, and the lookup response includes the layout.

## Impact

- **Backend**: migration `0007`, which adds the column settings and `layout_version` to `dashboards` and a new `dashboard_widgets` table. The `internal/dashboards` store gains column fields, layout loading in `Lookup` and `SaveLayout` with validation. A new handler for `PUT /api/dashboards/{id}/layout`. `POST`/`PATCH /api/dashboards` accept the column fields.
- **Frontend**: `DashboardView` renders the grid and sections and hosts edit mode. The `edit` search param is added on `/` and `/$userSlug/$dashboardSlug`. New editor components (toolbar, section dialog, delete confirmation, drag-and-drop with `@dnd-kit/sortable`). `DashboardDialog` gets the column fields. New translation keys (en and de).
- **Docs**: README dashboards and widgets sections, and the project context in `openspec/config.yaml` (12-column → configurable grid).
