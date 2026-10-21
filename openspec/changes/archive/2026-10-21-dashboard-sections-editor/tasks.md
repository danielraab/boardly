# Tasks

## 1. Data model

- [x] 1.1 Add migration `0007_dashboard_layout.sql`: `desktop_columns`, `mobile_columns`, `mobile_breakpoint` and `layout_version` on `dashboards`, and the `dashboard_widgets` table with the composite parent foreign key, the CHECKs and the index. Verify migration tests that existing dashboards get 6/2/768/0, that a child with another dashboard's parent is rejected, that an invalid breakpoint, a width of 13 and invalid JSON config are rejected, and that deleting a dashboard or a parent section removes its widgets

## 2. Column settings

- [x] 2.1 Add the column fields to `Dashboard`, `Fields` and `Patch` in `internal/dashboards`, with validation (`ErrInvalidColumns`, mobile ≤ desktop also when only one count is patched) and defaults on create. Verify store tests for the defaults, each range, mobile > desktop on create and on a one-field patch, and that a rename doesn't change `layout_version`
- [x] 2.2 Accept and return `desktopColumns`, `mobileColumns` and `mobileBreakpoint` in `GET/POST /api/dashboards` and `PATCH /api/dashboards/{id}`, mapping `ErrInvalidColumns` to 400 `invalid_columns`. Verify `httptest` tests for create with and without the fields, a patch of one field, and the 400 cases

## 3. Layout store and API

- [x] 3.1 Add the `Section` type, layout loading and tree building to `Lookup`, and `ID` (owner only), `Columns`, `LayoutVersion` and `Sections` to `View`. Verify store tests: ordered children, an empty dashboard gives `[]`, a visitor gets the layout of a public dashboard, and `id` is missing for visitors
- [x] 3.2 Implement `SaveLayout(userID, dashboardID, version, sections)`: whole-tree validation (kind, depth ≤ 5, ≤ 500 widgets, width, title, height), the versioned `UPDATE`, the delete and re-insert with kept IDs only for this dashboard's existing widgets, and the read-back. Verify store tests for a valid save (version +1, order kept), 5 vs 6 levels, each invalid field, the 500 cap, a stale version → `ErrLayoutConflict` with the layout unchanged, a foreign dashboard → `ErrNotFound`, and that a foreign widget ID is replaced with a new one
- [x] 3.3 Add `PUT /api/dashboards/{id}/layout` (requireUser, 256 KiB body cap, `DisallowUnknownFields`) with 400 `invalid_layout`, 404 `not_found` and 409 `layout_conflict`, and extend the lookup response. Verify `httptest` tests for 200 with the new version, 401 without a session, 404 for Bob's dashboard, 409 on a stale version, 400 on an unknown field, and the lookup JSON with the layout for owner and visitor

## 4. Frontend foundations

- [x] 4.1 Install `@dnd-kit/sortable` with pnpm (pinned like `@dnd-kit/core`) and verify `pnpm install` and the type check pass
- [x] 4.2 Extend `features/dashboards/api.ts` with the column fields, the layout types, the `id`/`columns`/`layoutVersion`/`sections` of `DashboardViewData`, and `saveLayout` (an `ApiError` with status 409 is detectable). Verify the type check passes and a unit test for the 409 detection
- [x] 4.3 Add `features/dashboards/layout/layoutTree.ts` (reducer `add`/`update`/`remove`/`move`, `depthOf`, `subtreeHeight`, `isDescendant`, `canMove`, `countDescendants`, `isSameTree`). Verify unit tests for each action, a move into its own descendant being refused, depth 5 allowed vs 6 refused for a section with children, and that a removed section takes its subtree with it

## 5. Rendering

- [x] 5.1 Add `useColumns` (matchMedia on the stored breakpoint), `DashboardGrid` and `SectionBlock` (span `min(width, parentCols)`, inner grid with the span as its column count, title heading, border flag, `--dashboard-row` 4rem with fixed heights and `overflow-y: auto`), and render them from `DashboardView` with the empty placeholder when there are no sections (owner text points to "Edit"). Verify component tests with a mocked `matchMedia`: the desktop vs mobile column count, the clamp of a width-6 section inside a width-3 section, a borderless section without a border, a 3-row section's height style, and the owner and visitor placeholder texts

## 6. Edit mode

- [x] 6.1 Add `validateSearch` with `edit` to `/` and `/$userSlug/$dashboardSlug`, and the toolbar next to the title ("Edit" for owners; "Save and exit" and ✕ in edit mode). Verify tests that the owner sees "Edit" and a visitor doesn't, that "Edit" sets `?edit=true`, that a visitor with `?edit=true` sees no controls, and that `/` with the default dashboard also offers "Edit"
- [x] 6.2 Add `useLayoutDraft`, the edit-mode controls on `SectionBlock` (dashed outline, drag handle, edit button, "Add section" at levels 1–4) and the top-level "Add section". Verify tests for the controls on a borderless section, no add button at level 5, only the top-level add button on an empty dashboard, and no controls in view mode
- [x] 6.3 Add `SectionDialog` (add and edit with title, width 1…desktopColumns keeping a larger stored width, border, Auto/Rows height) and `DeleteSectionDialog` (title or "untitled section" and the number of sections inside). Verify tests for adding a nested section at the end of its parent with the defaults, editing border and height, deleting a section with children, and a cancelled delete
- [x] 6.4 Implement "Save and exit" (PUT, cache update, leave edit mode with `replace`), the error message on failure, and the 409 banner with "Reload". Verify tests for a successful save leaving edit mode and showing the section, a network error keeping the draft, and a 409 showing the banner whose "Reload" resets the draft to the server's layout
- [x] 6.5 Add the unsaved-changes guard: `DiscardDraftDialog` for ✕ when dirty, and `useBlocker` with `enableBeforeUnload` for navigation. Verify tests that ✕ without changes leaves at once, ✕ with changes asks and both answers behave as specified, and that picking another dashboard in the switcher with changes asks first

## 7. Drag and drop

- [x] 7.1 Add the `DndContext` with a `SortableContext` per container, droppable containers for empty sections, the drop-target highlight in `onDragOver` with the move applied on drop (moves within one grid shown by the sortable strategy), cancel, the collision detection that excludes the active section's descendants, and `DragOverlay`. Verify tests for reordering two top-level sections and moving a section into an empty section with the keyboard (Space, arrows, Space), including its children
- [x] 7.2 Block moves that fail `canMove`, with the "at most 5 levels" message in the toolbar and in the screen-reader announcements, and translated announcements for pick up, move, drop and cancel. Verify a test that a keyboard move of a section with two child levels into a level-4 section leaves the draft unchanged and shows the message

## 8. Dashboard dialog, translations and docs

- [x] 8.1 Add the "Layout" fields (desktop columns, mobile columns with the "not above desktop" hint, breakpoint 640/768/1024) to `DashboardDialog`, with defaults on create and server `invalid_columns` shown. Verify tests for the defaults in the create dialog, the hint disabling save, and that editing sends the changed fields
- [x] 8.2 Add all new `dashboards.*` translation keys in English and German. Verify that the i18n key-parity check passes
- [x] 8.3 Update the README (configurable grid with 6 columns by default, the mobile settings, the section fields replacing icon/label/color, edit mode with save, cancel and the conflict message, the concept diagram) and the 12-column wording in `openspec/config.yaml`. Verify the flow in dev: create a dashboard, add nested sections, drag them around, hit the depth limit, save, cause a conflict from a second tab, and check the layout at a mobile width and as a visitor of a public dashboard

## 9. Integration

- [x] 9.1 Run `make check` and `make test` and verify both pass

## Workflow follow-up

- Archive with `/opsx:archive` after review.
