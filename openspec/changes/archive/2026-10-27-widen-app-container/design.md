# Design

## Context

`AppShell` wraps every route. Its `<header>` and `<main>` each carry `mx-auto w-full max-w-6xl` plus `px-4 sm:px-6`. The two have to stay in sync so the logo lines up with page titles. Dashboards render their grid as `repeat(desktopColumns, 1fr)` inside `<main>`, so they fill whatever width the shell gives them.

## Goals / Non-Goals

**Goals:**
- Raise the shared cap from 1152 px to 1536 px for header and content.
- Keep header and content aligned, and make it hard for them to drift apart again.

**Non-Goals:**
- Different widths per route or per dashboard, or a viewer toggle.
- Changing the column count or grid behaviour of dashboards. Widgets get wider; no more of them fit per row unless the dashboard's column count is raised.
- Changing vertical spacing above the dashboard grid.

## Decisions

- **`max-w-screen-2xl` (96rem = 1536 px).** Tailwind 4 resolves `max-w-screen-*` from the `--breakpoint-*` theme values, so no custom value is needed. Alternative `max-w-7xl` (1280 px) was too small a step to tell whether more width helps.
- **One shared constant for the container classes.** The width and padding classes move into a single string in `AppShell.tsx` used by both `<header>` and `<main>`, so a later change of the width can't leave one of them behind. A test asserts both elements carry the same width class.
- **Global, not route-specific.** Settings forms and help texts already cap themselves (`max-w-xl`, `max-w-[60ch]`), so a wider shell doesn't stretch them. The landing page and the favourites page are checked visually after the change.

## Risks / Trade-offs

- [Widgets stretch on wide screens; link lists get long rows] → Acceptable for a first try; users can raise a dashboard's desktop column count. Route- or dashboard-specific widths stay open as follow-ups.
- [Landing or favourites page looks too spread out at 1536 px] → Check during apply; if needed, give that page its own inner cap within this change.
