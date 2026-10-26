# Design

## Context

`TreeRow` on `/favourites` is a flex row with `paddingLeft: depth × 1.5rem`, a 28px move handle, a 28px toggle, a 16px icon, the name in a `min-w-0 flex-1` box (with the link count and tags), and `shrink-0` action buttons (up to four, 112px). Only the name can shrink, so at depth ≥ 6 on a phone it reaches 0px and the `shrink-0` parts overflow the `<li>`, the rounded `<ul>` and the page. The drop indicator is positioned with `left: depth × INDENT_REM + 2.25rem`. `ExplorerTree` indents with `depth × style.indent` px and truncates names in the same way, with only the info button on link rows.

## Goals / Non-Goals

**Goals:**
- No overlap and no page-level horizontal scroll at any depth or width.
- Unchanged look for trees that fit.

**Non-Goals:**
- Moving actions into a menu, or a drill-down view of one folder.
- A fixed-height tree with its own vertical scrollbar.

## Decisions

### A scroll container with per-row widths
The `<ul>` is wrapped in a `div` with `overflow-x-auto` that keeps the border and rounding (so the scrollbar sits inside the frame) and is a size container (`@container`). The `<ul>` is `w-max min-w-full`, so it is as wide as its widest row. Each row is `w-min min-w-[calc(100cqw-0.5rem)]`: as wide as its own min-content, but at least as wide as the frame. Rows that fit therefore look exactly as before, with their buttons at the frame's right edge, and only rows that need more room get wider.
*Alternatives:* all rows as wide as the widest row (`ul w-min min-w-full`) pushed the buttons of every row out of view as soon as one deep row overflowed; `overflow-x-auto` per row would scroll rows independently. Both rejected.

### Minimum name width
The name box (name, link count, tags) is `flex-1` with `min-w-24` (6rem) below `sm` and `min-w-32` (8rem) from `sm`; the name inside keeps `truncate`. The minimum raises the row's min-content width, which is what makes a deep row grow and the frame scroll. A short name leaves empty space inside the box, which isn't visible. 6rem on phones keeps folder rows up to depth 1 (and link rows deeper) within a 375px screen.

### No pinned buttons
The action buttons stay a `shrink-0` flex item after the name box, so by construction they can't overlap it. Pinning them to the frame's edge with `position: sticky` was tried first and dropped: in rows wider than the frame, the pinned buttons covered the end of the name and the link count, which is exactly the overlap this change removes.

### Indentation as a CSS variable
The `<ul>` sets `--indent: 0.75rem` and `sm:[--indent:1.5rem]`. Rows use `padding-left: calc(var(--indent) * depth + 0.25rem)` and the drop indicator `left: calc(var(--indent) * depth + 2.25rem)`, so both follow the breakpoint without JavaScript. `INDENT_REM` is removed.

### Explorer widget
`ExplorerTree`'s list is wrapped the same way (`@container overflow-x-auto`, list `w-max min-w-full`), each row is `w-min min-w-[100cqw]`, names get `min-w-24` (6rem; the widget is narrower and has smaller text), and the info button stays a `shrink-0` flex item after the link. Its indentation stays as it is: one or two widget columns are narrow on every screen, so a breakpoint wouldn't help.

## Risks / Trade-offs

- [Drag and drop inside a scrolled container: dnd-kit measures rects in viewport coordinates and already handles scrollable ancestors] → covered by the existing drag tests; checked in the real app with the tree scrolled.
- [On a phone, folder rows from depth 2 need a sideways scroll to reach their four buttons] → accepted trade-off of keeping all buttons visible on wide screens; moving them into a menu on phones was considered and left out.
- [jsdom doesn't do layout, so overflow can't be unit-tested] → tests assert the classes and structure; the real behaviour is checked in Chromium at 375px with a ten-level tree.
