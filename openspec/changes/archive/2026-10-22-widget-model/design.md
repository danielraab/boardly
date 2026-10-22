# Design

## Context

- `dashboard_widgets` already stores a generic widget (`kind`, `position`, `width`, `config` JSON), but `kind` has `CHECK (kind IN ('section'))`. SQLite can't change a check constraint in place.
- In Go, `dashboards.Section` holds `Children []Section`. `normalizeLayout`, `insertSections` and `loadSections` only handle sections. The section's settings are stored in `config` (`sectionConfig`), and `width` has its own column.
- In the web app, `Section` (`api.ts`) is the only node type. `layoutTree.ts` (reducer, `canMove`, `subtreeHeight`, `keyboardSlots`), `SectionGrid.tsx`, the dnd-kit wiring in `LayoutEditor.tsx` and `SectionDialog` all assume it.
- The follow-up changes `favourite-explorer-widget` and `link-widget` each add one kind. `image-store` is independent of this change.

## Goals / Non-Goals

**Goals:**
- One generic widget node in the API and the editor. A new kind is added by registering it in one place on each side, without touching the tree code.
- Kinds can resolve server-side data for the lookup and filter what visitors see.

**Non-Goals:**
- Any concrete content widget.
- Content widgets at the top level.
- A dashboard-wide S/M/L size.

## Decisions

### JSON shape: shared fields plus `config` and `data`
```
{ id, kind: "section", width, height, title, border, children: [...] }
{ id, kind: "<kind>",  width, height, config: {...}, data?: {...} }
```
Sections keep their flat fields so the existing section code and tests change little. Content widgets put everything kind-specific into `config`. `data` is response-only: `normalizeLayout` clears it before saving, and the web app strips it before sending. A custom `MarshalJSON` writes sections with their section fields and content widgets with `config`/`data`.

In the DB column `config`, a content widget stores `{"height": …, "config": {…}}`, because the table has no height column and the kind's config must not clash with the height key.

*Alternative*: everything in `config`, including the section's fields. That's more uniform, but it would rewrite all section code and tests for no user-visible gain.

The top-level JSON key stays `sections`, since the top level holds only sections.

### Go: `Widget` struct and a kind registry
`Section` becomes `Widget` with `Kind`, the section fields, `Config json.RawMessage`, `Data any` and `Children []Widget`. `Store.Kinds` (a `map[string]Kind`, empty from `NewStore`) maps a kind to:
```go
type Kind struct {
    // Normalize validates and canonicalises config for the owner (userID).
    Normalize func(ctx context.Context, userID string, config json.RawMessage) (json.RawMessage, error)
    // Resolve returns the data for a lookup; ok=false leaves the widget out. May be nil.
    Resolve func(ctx context.Context, viewer Viewer, config json.RawMessage) (data any, ok bool, err error)
}
type Viewer struct { OwnerID string; IsOwner bool }
```
A store field instead of a package-level map, so tests register a test-only kind on their own store, and kinds can capture the stores they need when `main.go` wires them. `Normalize` gets the owner's user ID because later kinds check references. `Resolve` runs in `Lookup` and after `SaveLayout` (for the owner), per widget. A visitor implies a public dashboard, so `Viewer` needs no public flag. With no kinds registered in this change, every non-section kind is rejected, and stored widgets of a kind the binary doesn't know are left out of lookups.

*Alternative*: hard-code a `switch` on the kind in `layout.go`. That's simpler now, but each new kind would touch the core file and its tests.

### Migration: rebuild `dashboard_widgets`
`0008_widget_kinds.sql` creates `dashboard_widgets_new` without the kind check (`kind TEXT NOT NULL`), copies the rows, drops the old table, renames the new one and recreates the index. It runs with foreign keys deferred, as SQLite's ALTER procedure recommends. The kind list is enforced in Go, so later kinds don't need another rebuild.

*Alternative*: widen the check list in every kind change. That means a table rebuild per kind, for little safety, because the store validates kinds anyway.

### Frontend: `Widget` union and registry
```ts
type Widget = Section | ContentWidget;
type Section = { id; kind: "section"; width; height; title; border; children: Widget[] };
type ContentWidget = { id; kind: string; width; height; config: unknown; data?: unknown };
```
`layoutTree.ts` operates on `Widget`. Only sections have children, `subtreeHeight` of a content widget is 0 (it adds no level), and `canMove` rejects a content widget at the top level and any parent that isn't a section. `keyboardSlots` skips the top-level slots for content widgets.

`widgets/registry.ts` maps a kind to `{ icon, labelKey, descriptionKey, defaults(), View, SettingsDialog }`. `SectionGrid` renders a content widget through `View` inside a `WidgetFrame` that does the grid span, the height, the scrolling and the edit controls. The picker lists the registry's entries.

### Editor flow
`LayoutEditor` has a separate state per dialog: the section dialog, the picker (`picking`: the parent section's ID) and the widget dialog (`{ widget, isNew, parentId }`). "Add widget" → picker → the kind's `Settings` with a new widget (full section width, height `auto`, `defaultConfig()`). A kind's `Settings` gets `{ widget, isNew, desktopColumns, onApply(widget), onDelete }` and is usually built on `WidgetSettingsForm`, which adds the shared width and height (`SizeFields`, also used by the section dialog) and the buttons, so every kind renders them the same way. The reducer's `update` action takes the whole new widget. Delete uses `DeleteWidgetDialog`, generalized from `DeleteSectionDialog`. A refused move shows "Sections can be nested at most 5 levels deep" or "Widgets can only be placed inside a section", depending on the reason.

### Dirty check and save
`isSameTree` and the save request strip `data` first, so resolved data never counts as a change and is never sent.

## Risks / Trade-offs

- [Table rebuild on a live DB] → It's a small table in one transaction, and the migration is tested with existing section rows.
- [dnd-kit collision between a leaf widget and a nested grid] → Leaf widgets register no droppable children area, so drops only land on section grids.
- [Picker that lists nothing in this change] → "Add widget" is hidden while the registry is empty.

## Migration Plan

The migration runs on start. Existing layouts keep all sections and get `kind: "section"` in the API. Rollback means restoring the DB backup, because the old binary's check would still accept the rebuilt table's section rows, but not rows of new kinds.
