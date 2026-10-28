# Proposal

## Why

Since `widen-app-container`, every page inside the app shell may be up to 1536 px wide. That suits dashboards, but on settings the cards, tab rule and list rows stretch across the whole container while their inputs stay 576 px wide, leaving large empty boxes and actions far away from their labels. Pages pick their width ad hoc today (nothing, `max-w-prose`, `mx-auto max-w-md`), so the next new page would hit the same problem.

## What Changes

- Introduce four named page widths, each a horizontally centred column inside the app shell's 1536 px container:
  - **full**: the whole container (1536 px)
  - **wide**: at most 1152 px
  - **narrow**: at most 768 px
  - **focus**: at most 448 px
- Every page uses exactly one of them:
  - full: a dashboard (own, default on `/`, public), the notes overview
  - wide: favourites, the dashboards list, the landing page
  - narrow: settings, the full-page note editor, the not-found page, the home greeting shown when no default dashboard is set
  - focus: login, login verification, "Save link" (share)
- The header keeps the full container width. On pages narrower than full, the page title no longer lines up with the app icon when the viewport is wider than the page; it does on full pages.
- Nothing inside the pages is restyled: only each page's outer width and its horizontal position change.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-shell`: "Content width" now limits the header-to-content alignment to full-width pages, and a new requirement "Page widths" defines the four widths and which page uses which.

## Impact

- Frontend: a new module with the width class constants in `frontend/src/components/`, the outer wrapper of each page listed above (`SettingsPage`, `FavouritesPage`, `DashboardsPage`, `NotesPage`, `NoteEditorPage` route, `Landing`, `Home`, `NotFound`, `LoginPage`, `VerifyPage`, `SharePage`, `DashboardView`), and tests.
- No backend, API or translation changes.
