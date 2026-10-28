# Tasks

## 1. Width constants

- [x] 1.1 Create `frontend/src/components/pageWidths.ts` exporting `fullPage` (`w-full`), `widePage` (`mx-auto w-full max-w-6xl`), `narrowPage` (`mx-auto w-full max-w-3xl`) and `focusPage` (`mx-auto w-full max-w-md`), each with a one-line doc comment naming the width; verify `pnpm biome check` passes on it

## 2. Apply widths to pages

- [x] 2.1 Full: add `fullPage` to the root of `DashboardPage` and of `LayoutEditor` in `features/dashboards/`, to `NotesPage`'s root section, and to `PageSkeleton` in `routes/index.tsx`; verify by opening a dashboard and `/notes` at 1920 px that they're still 1536 px wide
- [x] 2.2 Wide: add `widePage` to the root of `FavouritesPage`, `DashboardsPage` and `Landing`; verify at 1920 px that each is 1152 px wide and centred
- [x] 2.3 Narrow: add `narrowPage` to the root of `SettingsPage`, `Home`'s `Greeting`, and `NoteEditorPage` (only when `onClose` is not set); replace `max-w-prose` in `NotFound` with `narrowPage`; verify at 1920 px that each is 768 px wide and centred, and that a note opened from a note widget still shows in its 48rem dialog
- [x] 2.4 Focus: replace `mx-auto max-w-md` with `focusPage` on the root sections of `LoginPage`, `VerifyPage` and `SharePage`; verify these pages look unchanged
- [x] 2.5 Check with `grep -rn "max-w-md\|max-w-prose" frontend/src/features frontend/src/components` that no page root still uses a raw page cap; inner caps stay

## 3. Tests

- [x] 3.1 Add tests that the root of the settings page carries `max-w-3xl`, of favourites `max-w-6xl`, of login `max-w-md`, and that a dashboard page has no `max-w-*` below the shell; verify with `pnpm vitest run` on the touched test files
- [x] 3.2 Keep the existing `AppShell` test that header and main both carry `max-w-screen-2xl`; verify it still passes

## 4. Docs and verification

- [x] 4.1 View settings (all four tabs), favourites, notes, a note, a dashboard (view and edit), the landing page and login at 1920, 1280 and 375 px; confirm the widths from the spec and that nothing inside a page changed. Note any cramped page as a follow-up instead of restyling it
- [x] 4.2 Add a CHANGELOG entry for the page widths
- [x] 4.3 Run `make check` and `make test` and verify both pass
