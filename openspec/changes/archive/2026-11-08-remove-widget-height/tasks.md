# Tasks

## 1. Backend layout

- [x] 1.1 In `backend/internal/dashboards/layout.go`, stop returning and storing a height for content widgets, limit the height range check to sections, and always clear a content widget's height in `normalizeContent`; verify with `layout_test.go` cases for a sent height of 3 and 51, and for an out-of-range section height still returning `invalid_layout`
- [x] 1.2 Remove `FitContent` from `dashboards.Kind` and `backend/internal/widgets/link.go`, and verify `link_test.go` still passes without its fixed-height case
- [x] 1.3 Update `backend/internal/httpapi/dashboards_test.go` so content widgets have no `height` in responses, and verify `go test ./internal/httpapi/...` passes

## 2. Migration

- [x] 2.1 Add migration `0026_remove_widget_height.sql` as described in `design.md`, and verify with a `db_test.go` test on existing data covering: auto-height section with a fixed-height only child, section already fixed, several children, a nested section as the only child, and unchanged widths
- [x] 2.2 Verify `TestShippedMigrationsApply` passes with the new migration

## 3. Export and import

- [x] 3.1 In `StoredWidget`, apply the single-child rule to older exports, and verify with `backend/internal/dashboards/import_test.go` cases for the data-import delta's three scenarios
- [x] 3.2 Confirm the export format stays at version 1 and that a new export has no widget height: add a check to the export tests, and verify a dashboard round-trips through export and import with its section height and widget width intact
- [x] 3.3 Keep `backend/internal/dataimport/rewrite_test.go` passing for configs with and without `height`

## 4. API docs

- [x] 4.1 Remove `height` from the content widget schema in `backend/internal/httpapi/openapi/boardly.yaml`, keeping it on sections, and verify `TestOpenAPIDocumentsMatchRoutes` passes and a search of the schema shows `WidgetHeight` used only by the section

## 5. Frontend settings

- [x] 5.1 Remove `height` from `ContentWidget` in `features/dashboards/api.ts`, and have `LayoutEditor` create new widgets with width 12; verify with `pnpm exec tsc --noEmit` and a `DashboardWidgets.test.tsx` case for a new widget following its section
- [x] 5.2 Make `WidgetSettingsForm` offer only the width, with a "Full width" / "Volle Breite" option for the stored value 12 (new key in `en.json` and `de.json`), remove `fitContent`, and leave the section dialog unchanged; verify with tests that a widget dialog has no height field, a width-12 widget shows "Full width", and the section dialog has no "Full width" option

## 6. Frontend rendering

- [x] 6.1 In `SectionGrid.tsx`, remove the height from `WidgetFrame`, and let its content area scroll horizontally only; verify with a `DashboardLayout.test.tsx` case that a widget frame has no height style
- [x] 6.2 Let a content widget that is the only child fill a fixed-height, expanded section as described in `design.md`, and verify with `DashboardLayout.test.tsx` cases that the fill applies to that case only, not to several children, a nested section as only child, or an auto-height section
- [x] 6.3 Check in a real browser with the `run` skill: a short note fills a fixed-height section, a long note scrolls with the section with no scrollbar inside the widget, several widgets scroll together, wide content scrolls horizontally, and edit mode keeps the "Add" row at its natural height

## 7. Final checks

- [x] 7.1 Run `make check` and `make test`, since both sides change, and verify both pass
- [x] 7.2 Run `openspec validate remove-widget-height --strict` and verify it passes
