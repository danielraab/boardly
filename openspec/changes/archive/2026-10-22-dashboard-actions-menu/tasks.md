# Tasks

## 1. Backend: owner fields in the view

- [x] 1.1 Add `IsDefault` and `IsPublic` (`*bool`, `omitempty`) to `View` in `backend/internal/dashboards/dashboards.go`, select `is_default` in `Lookup`, and set both only for the owner. Verify by extending `TestLookup`: the owner gets both flags (also when `false`), and a visitor gets neither
- [x] 1.2 Extend `TestViewDashboard` in `backend/internal/httpapi/dashboards_test.go` with the "Owner fields" and "Visitor fields" scenarios, and verify that `go test ./...` passes

## 2. Frontend: dialog callback and view data

- [x] 2.1 Add `isDefault?: boolean` and `isPublic?: boolean` to `DashboardViewData` in `frontend/src/features/dashboards/api.ts`, and verify that `pnpm tsc` (via `make check`) passes
- [x] 2.2 Add an optional `onSaved(saved: Dashboard)` to `DashboardDialog`, called with the response before the queries are invalidated and the dialog closes. Verify that the existing `Dashboards.test.tsx` still passes

## 3. Frontend: actions menu

- [x] 3.1 Rename the translation key `dashboards.editor.edit` to `dashboards.editor.editLayout` ("Edit layout" / "Layout bearbeiten"), add `dashboards.view.actions` and `dashboards.view.settings`, and update `dashboards.view.emptyOwnerText` in en and de. Verify that both locale files have the same keys
- [x] 3.2 Replace the "Edit" button in `DashboardView` with a `DashboardActions` menu (an `Ellipsis` trigger with an accessible name, plus "Edit layout" and "Settings" items using `menuPopup` and `menuItem`). "Settings" opens `DashboardDialog` with the dashboard built from the view. Verify in the browser that the menu opens, both items work, and visitors and edit mode show no menu
- [x] 3.3 Add the `atHome` prop to `DashboardView`, pass it from `DefaultDashboard` in `Home.tsx`, and implement the `onSaved` navigation (slug change off `/`, default removed on `/`, with `replace`). Verify with the tests in 4.2

## 4. Tests

- [x] 4.1 Update `DashboardEditor.test.tsx` to enter edit mode through "Dashboard actions" → "Edit layout", assert that the menu button is missing for visitors and in edit mode, and assert the new empty-state text. Verify that the tests pass
- [x] 4.2 Add tests for "Settings": the dialog opens with the current values, a title change stays on the page, a slug change navigates to the new address, removing the default on `/` navigates to the dashboard's address, and a slug change alone on `/` stays at `/`. Verify that the tests pass
- [x] 4.3 Run `make check` and `make test` and verify that both pass
