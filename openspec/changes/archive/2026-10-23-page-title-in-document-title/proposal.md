# Proposal

## Why

Every Boardly page has the same browser tab title, "Boardly", so several open tabs can't be told apart and the browser history shows the same entry for every page (issue #23).

## What Changes

- The document title becomes `<page title> - Boardly`, where the page title is the page's main heading: the dashboard's title on dashboard pages (including the default dashboard on `/`), "Dashboards", "Favourites", "Settings", "Sign in", "Complete sign-in", "Save link" and "Page not found".
- Pages without a page title of their own (the landing page, the signed-in greeting on `/` when no default dashboard is set, loading states) keep the plain title "Boardly".
- The title follows the current language and updates when the language changes, and it follows the dashboard's title when that is renamed.

## Capabilities

### New Capabilities

### Modified Capabilities
- `app-shell`: adds a requirement for the document title of every page.

## Impact

- Frontend only: a small hook that sets `document.title`, called by each page component. No backend, API or dependency changes.
- `index.html` keeps "Boardly" as the initial title.
