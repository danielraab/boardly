# Design

## Context

Sections are rendered by `SectionFrame` in `layout/SectionGrid.tsx`, for both the dashboard view and the layout editor (`useEditor()` is non-null in edit mode). Outside edit mode, the heading row only exists when the section has a title. `lib/useStoredIdSet.ts` already keeps a set of IDs in localStorage (used by the favourite explorer's expanded folders and the favourites page).

## Goals / Non-Goals

**Goals:**
- Per-browser collapse state that survives reloads, with no server round-trip.

**Non-Goals:**
- Syncing collapse state across devices or storing it in the layout.
- A "collapse by default" section setting.
- Collapsing in edit mode or collapsing content widgets.

## Decisions

- **One storage key for all dashboards: `boardly.collapsedSections`, a set of section IDs.** Section IDs are unique across dashboards, and visitors don't get the dashboard's `id`, so a per-dashboard key isn't possible for them. Reuses `useStoredIdSet` as-is. Alternative: a key per dashboard slug — breaks when the slug changes, for no benefit.
- **Read the set once in `DashboardGrid` and pass it down via a small React context** (`CollapsedContext` with the set and a toggle), instead of calling `useStoredIdSet` in every `SectionFrame`. Each hook instance holds its own copy of the set, so separate instances would overwrite each other's writes.
- **Only titled sections get a toggle.** An untitled section has no heading outside edit mode; adding a heading row just for the toggle would change how borderless untitled sections look, and a collapsed untitled section would show nothing that tells the viewer what it is.
- **The toggle is a chevron button before the title** (`ChevronDown` / `ChevronRight` from lucide-react), with `aria-expanded`. No `aria-controls`: a collapsed body isn't rendered, so there is no element to point at. Labels reuse the wording of the existing `expand`/`collapse` strings under a new `dashboards.section.expand` / `dashboards.section.collapse` key with `{{title}}`.
- **Collapsed means: the `SectionList` is not rendered and the fixed height is dropped** (height `auto`, no `overflow-y-auto`). Widget data comes with the dashboard lookup, so hiding children only saves render work; nothing extra is fetched on expand.
- **Edit mode ignores the set** (`collapsed = !editor && title && set.has(id)`), so the editor never hides drop targets and the stored set is left untouched.

## Risks / Trade-offs

- [IDs of deleted sections stay in localStorage] → Negligible size (a UUID per collapsed section); not worth pruning, which would need all of the user's layouts.
- [Collapsed sections next to tall ones leave empty space in that grid row] → Expected with a CSS grid; the section keeps its width so the layout doesn't jump sideways.
