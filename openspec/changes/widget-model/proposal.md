# Proposal

## Why

Dashboards can only hold sections, so they can't show any content yet. Before the first content widgets (favourite explorer, link) can be built, the layout has to hold widgets of other kinds inside sections, and the editor has to offer a way to add them.

## What Changes

- The layout becomes a tree of widgets with a `kind`. The top level still holds only sections. Inside a section, sections and content widgets of any registered kind are allowed.
- Every widget has an `id`, a `kind`, a width in columns and a height (`auto` or rows), like sections today. A content widget also has a `config` object whose shape is defined by its kind, and it can't have children.
- A content widget fills its grid cell. When its content doesn't fit, it scrolls inside the widget.
- The server validates each content widget's `config` with the rules of its kind and rejects unknown kinds with 400 `invalid_layout`.
- The dashboard lookup can attach read-only, server-resolved `data` to a content widget, for example a favourites subtree. `data` is ignored on save. Each kind decides what visitors of a public dashboard get, and it can leave a widget out of the response entirely.
- In edit mode, every section shows an "Add widget" / "Widget hinzufügen" button next to "Add section". It opens a picker dialog that lists the available widget kinds. Picking a kind opens that kind's settings dialog. Confirming adds the widget at the end of the section, and cancelling adds nothing.
- "Add widget" is also shown in sections at level 5, because content widgets don't add a nesting level.
- Content widgets get a drag handle and a settings button in edit mode. They are moved and deleted like sections, but nothing can be dropped into them.
- This change adds no widget kinds. With none registered, "Add widget" isn't shown. The favourite explorer and the link widget follow in their own changes.
- **BREAKING** (internal API only): every widget in the layout JSON now carries `kind`. The web app is updated in the same change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dashboard-layout`: sections may contain content widgets; common widget settings (width, height); content widget rendering and scrolling; `kind`, `config` and `data` in the lookup; per-kind validation and visibility.
- `dashboard-editing`: the "Add widget" button and picker, settings dialogs per kind, and edit controls, moving and deleting for content widgets.

## Impact

- Backend: migration `0008_widget_kinds.sql` (rebuilds `dashboard_widgets` to widen the `kind` check), `backend/internal/dashboards/layout.go` (widget union, per-kind registry for validation and resolving), `View`/`Lookup`, and their tests.
- Frontend: `api.ts` (widget types), `layoutTree.ts` (generic over widgets), `SectionGrid.tsx` (leaf widgets, "Add widget"), `LayoutEditor.tsx`, a new `WidgetPicker` dialog and a frontend widget registry, en/de locales, and the editor tests.
- No new dependencies.
