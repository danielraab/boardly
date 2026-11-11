# Design

## Context

`FeedItemRow` (`frontend/src/features/feeds/FeedItemRow.tsx`) renders the rows on `/feeds` and in the feed widget. It is one flex row: icon, text column, play button, and the flag group. A news row's link covers the whole row through `after:absolute after:inset-0`, and the buttons sit above that layer with `relative z-10`. The backend uses one constant, `maxTextLength = 500` (`backend/internal/feeds/parse.go`), for feed titles, item titles and summaries, both in `parse.go` and for imported items in `refresh.go` (`insertImported`).

The change `sync-episode-progress`, which follows this one, adds a progress bar at the bottom edge of a row and a menu button next to the play button. The layout should leave room for both without a second redesign.

## Goals / Non-Goals

**Goals:**
- One row component whose layout follows the row's width, used in both places.
- An accessible summary toggle that doesn't collide with the row's link layer.
- Separate limits for titles and summaries.

**Non-Goals:**
- Show notes with HTML, links or images. Summaries stay plain text.
- Remembering which summaries are expanded.
- Backfilling summaries of stored items. They get longer on the next refresh that delivers the entry.

## Decisions

### Container query instead of a viewport breakpoint
The row's `<li>` becomes a container (`@container` in Tailwind 4), and the layout switches at `@[480px]`. A feed widget in a narrow desktop column then gets the same layout as a phone, and a wide widget on a tablet keeps the wide one. A viewport breakpoint (`sm:`) would get the widget case wrong. This is the first container query in the codebase, and Tailwind 4 supports it without a plugin.

### Grid areas instead of two separate markups
The row becomes a CSS grid whose areas move at the breakpoint:

```
narrow (< 480 px)                 wide (≥ 480 px)
┌──────┬──────────────────┐       ┌──────┬────────────────┬─────────┐
│ icon │ meta (feed · t)  │       │ icon │ title          │ actions │
├──────┴──────────────────┤       │      │ meta           │         │
│ title (2 lines)         │       │      │ summary        │         │
│ summary (+ toggle)      │       └──────┴────────────────┴─────────┘
│               actions   │
└─────────────────────────┘
```

Each element appears once in the DOM, so tests, focus order and accessible names stay the same in both layouts. Rendering the narrow and wide variants separately and hiding one would double every button in the accessibility tree. The order in the DOM is icon, title, meta, summary, actions. In the narrow layout the grid shows the meta before the title, while the title stays first for screen readers, so the row's heading still comes first.

The narrow title uses `line-clamp-2` instead of `truncate`. The wide layout keeps `truncate`, so rows in dense widgets keep one line per title. A row ends with an empty slot at its bottom edge (`relative` on the row and nothing inside yet), which `sync-episode-progress` fills with the bar.

### Summary toggle
- The summary keeps `line-clamp-2`. The toggle appears only when the text is cut off, measured with `scrollHeight > clientHeight`, in a layout effect and on `ResizeObserver` changes of the row, so that rotating a phone adds or removes it.
- The toggle is a `<button>` with `aria-expanded` and `aria-controls` pointing at the summary, placed after the summary. It gets `relative z-10` like the other buttons, so it sits above the news row's link layer, and activating it doesn't open the link.
- Expanded state is local `useState` per row, which matches "not remembered".
- In edit mode (`editing`), no toggle is rendered.
- Tapping the summary text still opens the link. The toggle is the only control that expands, so one tap never does two things.

### Limits
`maxTextLength` is replaced by `maxTitleLength = 500` (feed title, item title) and `maxSummaryLength = 1000` (summary). `insertImported` uses the same constants, so an import accepts summaries of up to 1000 characters. The "title from the first 100 characters of the summary" rule stays as it is. The refresh upsert already overwrites `summary` for entries it sees again, so stored items need no migration.

## Risks / Trade-offs

- [The measurement runs on every row of a long list] → It's one layout read per row, and the `ResizeObserver` watches the list once, not each row. Rows load in pages of 50.
- [Twice as much summary text per item in the database] → At most 1000 × 100 items per feed, about 100 KB per feed. That's negligible for SQLite.
- [An older Boardly that imports a newer export cuts summaries to 500] → That's acceptable, the format is unchanged and nothing fails.
- [Tests in jsdom can't measure layout] → Tests stub `scrollHeight` and `clientHeight` to check the toggle. The layout switch itself is checked by hand on a phone and in a narrow widget (task 4).
