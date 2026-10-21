# Design

## Context

The `dashboards` table (migration `0006`) holds title, slug and flags. `internal/dashboards.Store.Lookup` resolves `/<user>/<dashboard>` with its visibility rules, and `GET /api/u/{userSlug}/{dashboardSlug}` returns `{dashboard:{slug,title}, ownerSlug, isOwner, dashboards}`. In the frontend, `DashboardView` renders the title and a placeholder, and both `/` (`Home`, for the default dashboard) and `routes/$userSlug/$dashboardSlug.tsx` use it. The favourites page already uses `@dnd-kit/core` with pointer and keyboard sensors, screen-reader announcements and a `parent_id` + `position` tree. Confirmations use Base UI `AlertDialog`, and routes validate search params with `validateSearch` (see `routes/dashboards.tsx`).

Requirements are in `dashboard-layout`, `dashboard-editing` and `dashboards` of this change.

## Goals / Non-Goals

**Goals:**
- A widget storage model that later widget types (link, navigation tree) extend without schema changes.
- One save per edit session, all-or-nothing, with server-side validation of the whole tree.
- One rendering path for view mode and edit mode, so what the owner edits is what visitors see.

**Non-Goals:**
- Per-widget API endpoints. Nothing outside the editor writes the layout yet.
- Stable widget IDs across dashboards, and copying widgets between dashboards.
- S/M/L row sizes (the row unit is a single constant for now).

## Decisions

### Schema (migration `0007_dashboard_layout.sql`)
```
ALTER TABLE dashboards ADD COLUMN desktop_columns   INTEGER NOT NULL DEFAULT 6   CHECK (desktop_columns BETWEEN 1 AND 12);
ALTER TABLE dashboards ADD COLUMN mobile_columns    INTEGER NOT NULL DEFAULT 2   CHECK (mobile_columns BETWEEN 1 AND 12);
ALTER TABLE dashboards ADD COLUMN mobile_breakpoint INTEGER NOT NULL DEFAULT 768 CHECK (mobile_breakpoint IN (640, 768, 1024));
ALTER TABLE dashboards ADD COLUMN layout_version    INTEGER NOT NULL DEFAULT 0;

dashboard_widgets
  id            TEXT PK (uuid)
  dashboard_id  TEXT NOT NULL → dashboards (id) ON DELETE CASCADE
  parent_id     TEXT            (NULL = top level)
  kind          TEXT NOT NULL CHECK (kind IN ('section'))
  position      INTEGER NOT NULL
  width         INTEGER NOT NULL CHECK (width BETWEEN 1 AND 12)
  config        TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(config))
  UNIQUE (id, dashboard_id)
  FOREIGN KEY (parent_id, dashboard_id) → dashboard_widgets (id, dashboard_id) ON DELETE CASCADE
  INDEX (dashboard_id, parent_id, position)
```
- This mirrors `favourite_nodes`. The composite foreign key keeps every child inside its parent's dashboard.
- `width` is a column because every widget kind can have one later. The settings that depend on the kind go in `config`: for a section, `{"title": string|null, "border": bool, "height": "auto"|int}`. A new widget kind only extends the `kind` CHECK (one table rebuild in its migration) and the validation.
- "Mobile ≤ desktop" is checked in the store, not in SQL, because `ALTER TABLE ADD COLUMN` can't add a check across columns.
- *Alternative:* a single `layout` JSON column on `dashboards`. It's simpler to write, but it can't be referenced or queried per widget later (navigation trees will point at favourite nodes, and a deleted favourite node should be found by query). The table was chosen in exploration.

### Layout version for the conflict check
`layout_version` only increases on a successful layout save. `updated_at` is not used, because a rename on `/dashboards` would bump it and turn an unrelated edit into a conflict. The save runs `UPDATE dashboards SET layout_version = layout_version + 1 WHERE id = ? AND user_id = ? AND layout_version = ?` first. Zero affected rows means not found (the dashboard isn't the user's) or a conflict (the version is stale). A follow-up `SELECT` tells the two apart.

### Save = replace the tree in one transaction
`PUT /api/dashboards/{id}/layout` with `{"version": n, "sections": [Section…]}`, where `Section = {id?, width, title, border, height, children: [Section…]}`. The store:
1. Validates the whole tree before writing: at most 5 levels, at most 500 widgets in total, width 1–12, title trimmed with at most 100 runes (empty → null), height `"auto"` or an integer 1–50, and no unknown fields (`DisallowUnknownFields`). Any error → `ErrInvalidLayout` → 400 `invalid_layout` with a message naming the problem.
2. Checks and bumps the version (see above) → `ErrLayoutConflict` → 409 `layout_conflict`.
3. Loads the current widget IDs of the dashboard, deletes all widgets, and inserts the tree depth-first with `position` = index. A submitted `id` is kept only if it was in this dashboard before. Otherwise (new, foreign or malformed) a new UUID is generated, so a client can never collide with or take over another dashboard's rows.
4. Responds 200 with `{version, sections}` read back from the database.

The request body is capped with `http.MaxBytesReader` (256 KiB). *Alternative:* diff the tree and update rows in place. That's more code for no visible difference, since no row outside the editor references widget IDs yet.

### Lookup and API shape
- `Dashboard` gains `DesktopColumns`, `MobileColumns` and `MobileBreakpoint` (`desktopColumns`, `mobileColumns`, `mobileBreakpoint`). `Fields` and `Patch` gain the same values as pointers or defaults. `Create`/`Update` validate them (`ErrInvalidColumns` → 400 `invalid_columns`). A `PATCH` that changes only one of the two counts is checked against the stored value of the other.
- `View` gains `ID` (only for the owner, `omitempty`; needed for the save URL), `Columns {desktop, mobile, breakpoint}`, `LayoutVersion` and `Sections []Section`. `Lookup` loads the widgets in one query ordered by `parent_id, position` and builds the tree in Go.

### Frontend: one renderer, edit mode as a layer
```
DashboardView (view: DashboardViewData, edit: boolean)
├── header: title + [Edit] | [Save and exit][✕]       (owner only)
├── LayoutProvider  ← draft (edit) or saved sections (view)
│   └── Grid cols = useColumns(view.columns)  (matchMedia on `(min-width: Bpx)`)
│       └── SectionBlock (section, parentCols, level)
│             span = min(width, parentCols); inner grid cols = span
│             edit: drag handle, ⚙, "Add section" (level < 5), dashed outline if borderless
└── SectionDialog / DeleteSectionDialog / DiscardDraftDialog / conflict banner
```
- `useColumns` uses `window.matchMedia` with the stored breakpoint, because Tailwind breakpoints are fixed at build time. The grid gets `style={{ gridTemplateColumns: \`repeat(${n}, minmax(0, 1fr))\` }}`, and each section gets `gridColumn: \`span ${Math.min(width, parentCols)}\``.
- Row height: a CSS custom property `--dashboard-row: 4rem` on the grid. A fixed height of N rows is `calc(N * var(--dashboard-row) + (N - 1) * var(--dashboard-gap))` with `overflow-y: auto`. S/M/L can later change the variable only.
- In edit mode, a section with a border keeps its normal border. A borderless section gets a dashed `outline` in edit mode only, so it is still visible and can be grabbed.

### Draft state
- `useLayoutDraft(saved)` holds the draft in a `useReducer` with the actions `add(parentId|null, section)`, `update(id, patch)`, `remove(id)` and `move(id, toParentId|null, index)`. New sections get `crypto.randomUUID()` as a client ID, which the server replaces (see above). The reducer is a pure module (`layoutTree.ts`) with unit tests, including `depthOf`, `subtreeHeight`, `isWithin`, `canMove` and `keyboardSlots`.
- `dirty` = a deep comparison of the draft with the saved tree (the trees are small).
- The draft lives in `DashboardView` while `edit` is set and is dropped when edit mode ends.

### Edit mode in the URL
`validateSearch` on `/` and `/$userSlug/$dashboardSlug` accepts `edit` (only `true` is kept). `editing = search.edit && view.isOwner`. "Edit" navigates with `search: { edit: true }`. Save and cancel navigate with `search: {}` and `replace: true`, so Back doesn't reopen edit mode.

### Unsaved-changes guard
TanStack Router's `useBlocker({ shouldBlockFn, enableBeforeUnload, withResolver: true })`, both returning "dirty and not leaving on purpose", covers in-app navigation (switcher, user menu, links) and reload/close (the browser's own dialog). The resolver drives a `DiscardDraftDialog` (Base UI `AlertDialog`). ✕ opens the same dialog directly when `dirty`. Leaving edit mode after a successful save turns the blocker off first (`dirty` is false once the saved data is set in the query cache).

### Save flow
`saveLayout(id, version, sections)` → on 200, `setQueriesData` on every `['dashboard-view', …]` entry of this dashboard (it can be cached under `/` and under its address) with the new `sections` and `layoutVersion`, then navigate without `edit`. On 409, a conflict banner has a "Reload" action: it invalidates the view queries, reads the fresh data from the cache and resets the draft and the base version from it. On 400 or a network error, an inline error stays in the toolbar, and the draft is kept.

### Drag and drop with `@dnd-kit/sortable`
- One `DndContext` for the dashboard and one `SortableContext` per container (the top level plus each section's children), with `rectSortingStrategy` because the items are grid cells. Every section also registers a droppable "container" ID (`children:<id>`, and `children:root` for the top level), so empty sections accept drops.
- Inside one container, the sortable strategy shows the new order with transforms, and `onDragEnd` commits the move to the hovered sibling's index. A move into another container is **not** applied while dragging. `onDragOver` only records a drop target (parent, index, blocked or not) and highlights that container (red when blocked), and `onDragEnd` applies it. Applying it while dragging, the standard multi-container pattern, reflowed the whole layout under the pointer. In the browser check this made sections cascade through the nested sections they passed over instead of landing where they were aimed.
- Dropping on a section's header (or padding) places the dragged section next to it, while dropping on its content area (its children grid) puts it inside. Of all targets under the pointer, the smallest, which is the innermost, wins.
- **Blocking:** `canMove(tree, activeId, targetParentId)` checks `depthOf(targetParent) + subtreeHeight(active) ≤ 5` and that the target isn't inside the active section (`isWithin`). A blocked target is highlighted in red. Dropping there leaves the tree unchanged and shows the message "Sections can be nested at most 5 levels deep" in the toolbar for a few seconds, also announced through `announcements`. A custom collision detection also filters out the dragged section's own descendants and their containers, so the pointer can't target them at all.
- Sensors: `PointerSensor` with a 4 px distance (as on the favourites page). Announcements are translated, as on the favourites page.
- **Keyboard moves** don't use dnd-kit's `KeyboardSensor`. Its coordinate getter needs real layout to find targets across nested grids. Instead, like the favourites page, Space or Enter on a handle picks the section up, and `keyboardSlots(tree, id)` lists every place in reading order (before each section, inside it, at the end of each list, excluding the section's own subtree). The arrow keys step through the slots and apply each valid one as a preview move. A slot that would be too deep is announced as not possible and leaves the preview where it is. Space or Enter drops (a too-deep slot restores the snapshot and shows the message), Escape restores the snapshot, and moving focus elsewhere cancels. Focus is put back on the handle after each move, because a move to another grid remounts the section. This also makes keyboard moves testable in jsdom, which has no layout.
- Collision detection: `pointerWithin`, sorted by area so the innermost target wins, falling back to `closestCenter` when nothing is under the pointer.

### Section dialog
One `SectionDialog` (Base UI `Dialog`) for add and edit. Its fields are title (text), width (select, 1…desktopColumns), border (switch) and height (a segmented "Auto | Rows" choice plus a number input 1–50). The width select clamps to the desktop count, but an existing larger width (after the count was lowered) is shown and kept as long as the user doesn't change it. In edit mode, the dialog also has "Delete section", which opens `DeleteSectionDialog` with the title (or "untitled section") and the number of sections inside.

### Dashboard dialog
`DashboardDialog` gets a "Layout" group with desktop columns (select 1–12), mobile columns (select 1–12, with a hint and saving disabled when it's above desktop) and the breakpoint (select 640 / 768 / 1024 px, labeled as small, medium and large screens with the pixel value).

## Risks / Trade-offs

- [Nested sortable containers with grid items are fiddly; items jump when containers resize during the drag] → Moves between containers are applied on drop only, so nothing reflows while dragging. `DragOverlay` renders the dragged section's title, so the original keeps its slot. Reducer tests cover the tree logic, and component tests cover keyboard moves.
- [Pointer dragging can't be exercised in jsdom] → The tree logic (`canMove`, `move`) is shared by pointer and keyboard moves and covered by unit and keyboard tests. Pointer dragging is checked by hand in the browser.
- [Widget IDs change when the client sends new ones] → Nothing references widget IDs yet. Kept IDs are preserved, so later widget types that need stable IDs get them for every widget that already exists.
- [Lowering the desktop column count makes stored widths larger than the grid] → Widths are clamped when rendering, by design. Raising the count restores the original layout.
- [A visitor's page loads the whole tree] → The 500-widget cap bounds the size, and the tree is only returned under the same visibility rules as the dashboard.
- [`beforeunload` text can't be customized] → The browser's generic dialog is accepted, as the spec says.

## Migration Plan

`0007` runs on start: existing dashboards get 6/2/768, version 0 and no widgets, so they show the empty placeholder as before. Rollback means restoring the pre-upgrade backup (there are no down migrations).
