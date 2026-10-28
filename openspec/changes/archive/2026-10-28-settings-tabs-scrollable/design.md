# Design

## Context

The tab list is a plain `flex` row with a bottom border; each tab uses `-mb-px` and a 2px bottom border so the active underline covers the list's border. Nothing limits its width, so on a phone the row overflows.

## Goals / Non-Goals

**Goals:**
- Tab bar scrolls horizontally inside its own box on narrow screens.
- Selected tab is visible after load and after changing tabs.

**Non-Goals:**
- Switching to a dropdown or a different navigation pattern on mobile.
- Scroll-shadow / fade indicators.

## Decisions

- **`overflow-x-auto` on the tab list, `shrink-0 whitespace-nowrap` on tabs.** Simplest fix that keeps the same look on desktop.
- **Draw the list's bottom line as an inset box-shadow instead of a border, and drop `-mb-px` on tabs.** With `overflow-x: auto` the list also clips vertically, so the tabs' negative margin would cut off the active underline or add a vertical scrollbar. An inset shadow sits under the tabs' own 2px border, giving the same visual without overlap tricks.
- **Hide the scrollbar** (`scrollbar-width: none` + `::-webkit-scrollbar` hidden). On touch devices the tabs are swiped; on desktop they fit, so a scrollbar would only add visual noise. The partially cut-off last tab signals that more content exists.
- **Scroll the selected tab into view** with `scrollIntoView({ block: "nearest", inline: "nearest" })` when the selected tab changes (including the first render). `block: "nearest"` keeps the page from jumping vertically.

## Risks / Trade-offs

- [Hidden scrollbar is less discoverable on desktop with a narrow window] → The clipped tab at the edge shows there is more; keyboard arrow navigation (Base UI) also moves focus and scrolls.
