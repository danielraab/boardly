# Design

## Context

The title is set once in `frontend/index.html` ("Boardly") and never changed. Each page component already renders its own translated `<h1>`; dashboard pages render theirs through `DashboardPage` in `features/dashboards/DashboardView.tsx`, which is used both on `/<user>/<dashboard>` and for the default dashboard on `/`.

## Goals / Non-Goals

**Goals:**
- One place that formats `<page title> - Boardly`, used by every page.

**Non-Goals:**
- Server-side titles (the SPA fallback serves the same `index.html`; crawlers and link previews keep "Boardly").
- Changing the PWA manifest name or any other meta tag.

## Decisions

- **A `usePageTitle(title?: string)` hook in `frontend/src/lib/usePageTitle.ts`.** It sets `document.title` in an effect to `` `${title} - Boardly` `` or `"Boardly"` when `title` is empty, and resets to `"Boardly"` on unmount, so a page that has no title of its own never inherits the previous page's. Because the argument is the already-translated string, a language change or dashboard rename re-runs the effect.
  - *Alternative: TanStack Router `head()` per route with `<HeadContent />`.* Rejected: the dashboard title is only known after the route's query loads in the component, and translated titles would need i18n outside React; the hook keeps the title next to the heading it mirrors.
  - *Alternative: a context in `AppShell` that pages report into (like `currentDashboard`).* Unnecessary; `document.title` is already global state.
- **Call sites mirror the `<h1>`:** `DashboardPage` (dashboard title), `DashboardsPage`, `FavouritesPage`, `SettingsPage`, `LoginPage` ("Sign in" for both steps), `VerifyPage`, `SharePage` ("Save link" for both steps), `NotFound`. Landing, the home greeting and skeletons do not call it, so they show "Boardly".
- **"Boardly" stays a literal**, matching the header and `index.html`; the product name isn't translated.

## Risks / Trade-offs

- [A page forgets to call the hook] → its title falls back to "Boardly", which is the current behaviour; tests cover each page in the spec scenarios.
- [Effect order on navigation: old page's cleanup vs. new page's effect] → React runs unmount cleanups before the new mount's effects in the same commit, so the new title wins.
