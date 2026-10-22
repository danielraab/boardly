# Tasks

## 1. Backend: storage and model

- [x] 1.1 Add migration `0008_widget_kinds.sql` that rebuilds `dashboard_widgets` without the kind check and keeps rows, keys and the index. Verify with a migration test that existing section rows survive and a row of kind `x` can be inserted
- [x] 1.2 Replace `Section` with `Widget` (kind, section fields, `Config`, `Data`, `Children`) in `backend/internal/dashboards/layout.go`, and add the kind registry (`Normalize`, `Resolve`). Verify that `go build ./...` passes
- [x] 1.3 Update `normalizeLayout` for sections only at the top level, unknown kinds, content widgets without children, common width/height ranges, nesting counted on sections only, the 500-widget limit and `Normalize` per kind. Verify with table tests in `layout_test.go` for each case, using a test-only kind
- [x] 1.4 Update `insertSections`/`loadSections` (rename to widgets) to store and load every kind, with `config` from `Normalize` and `data` never stored. Verify with a round-trip test using a test-only kind
- [x] 1.5 Call `Resolve` in `Lookup` for content widgets, drop widgets for which it returns `ok=false`, and pass owner/visitor/public info. Verify with a lookup test where a test kind hides a widget from visitors only
- [x] 1.6 Update `httpapi/dashboards_test.go` for the new JSON (`kind` on every widget), and verify that `go test ./...` passes

## 2. Frontend: types and tree

- [x] 2.1 Change `api.ts` to the `Widget`/`Section`/`ContentWidget` types, and verify that `pnpm tsc` passes after the follow-up tasks
- [x] 2.2 Make `layoutTree.ts` generic over `Widget` (children only on sections, `subtreeHeight` 0 for content widgets, `canMove` rejecting top level and non-section parents, `keyboardSlots` without top-level slots for content widgets), and strip `data` in `isSameTree` and before saving. Verify with new cases in `layoutTree.test.ts`
- [x] 2.3 Add `widgets/registry.ts` with the registry type and an empty registry, plus `WidgetSettingsForm` with `SizeFields` (width, height) shared by kind dialogs. Verify that `pnpm tsc` passes

## 3. Frontend: grid and editor

- [x] 3.1 Render content widgets in `SectionGrid.tsx` through a `WidgetFrame` (span, height, `overflow-auto`, dashed outline, drag handle and edit button in edit mode) and the kind's `View`. Verify with a test using a test-only registered kind
- [x] 3.2 Show "Add widget" next to "Add section" in every section (alone at level 5, never at the top level, hidden when the registry is empty), and add the `WidgetPicker` dialog listing registry entries. Verify with editor tests
- [x] 3.3 Generalize the `LayoutEditor` dialog state (section / picker / widget), wire add, edit and delete for content widgets, and generalize `DeleteSectionDialog` to name widgets. Verify with editor tests for add, cancel, edit and delete
- [x] 3.4 Allow pointer and keyboard moves of content widgets between sections and block top-level drops. Verify with editor tests for both
- [x] 3.5 Add the en/de keys (`dashboards.editor.addWidget`, picker title and texts, widget delete texts), and verify that the i18n check passes

## 4. Verification

- [x] 4.1 Run `make check` and `make test` and verify that both pass
