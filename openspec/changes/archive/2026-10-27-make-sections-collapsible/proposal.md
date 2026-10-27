# Proposal

## Why

Dashboards with many sections get long, and a viewer often only needs a few of them at a time. Viewers can't hide sections they don't need right now ([#38](https://github.com/danielraab/boardly/issues/38)).

## What Changes

- A section with a title gets a toggle in its heading that collapses or expands it. A collapsed section only shows its heading; its children are hidden.
- The collapsed sections are remembered in the viewer's browser (localStorage), so they stay collapsed after a reload. Nothing is stored on the server, and the layout and its API don't change.
- It works for everyone who can see the dashboard, owners and visitors of public dashboards alike.
- In edit mode every section is shown expanded and has no toggle, so the layout can still be edited and widgets can be dropped into any section.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `dashboard-layout`: adds a requirement for collapsing and expanding sections in the viewer's browser.

## Impact

- Frontend only: `frontend/src/features/dashboards/layout/SectionGrid.tsx` (section heading and body), reusing `frontend/src/lib/useStoredIdSet.ts` for the stored state.
- New translations in `en.json` and `de.json`.
- No backend, API or database changes.
