# Tasks

## 1. Hook

- [x] 1.1 Add `usePageTitle` in `frontend/src/lib/usePageTitle.ts` that sets `document.title` to `<title> - Boardly` (or `Boardly` without a title) and resets it on unmount; verify with a unit test in `frontend/src/lib/usePageTitle.test.tsx` covering set, update and reset

## 2. Pages

The page tests live together in `frontend/src/pageTitle.test.tsx`, one per spec scenario.

- [x] 2.1 Call the hook in `DashboardPage` with the dashboard title; verify a test that `/<user>/<dashboard>` and `/` with a default dashboard show "<dashboard> - Boardly"
- [x] 2.2 Call the hook in `DashboardsPage`, `FavouritesPage` and `SettingsPage`; verify tests for "Dashboards - Boardly", "Favourites - Boardly" and "Settings - Boardly", and that switching to German gives "Favoriten - Boardly"
- [x] 2.3 Call the hook in `LoginPage`, `VerifyPage`, `SharePage` and `NotFound`; verify tests for the login and not-found titles
- [x] 2.4 Verify the landing page keeps the title "Boardly" with a test

## 3. Checks

- [x] 3.1 Run `make check` and `make test` and verify both pass
