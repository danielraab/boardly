# Tasks

## 1. App shell

- [x] 1.1 In `frontend/src/components/AppShell.tsx`, move the shared container classes (`mx-auto w-full max-w-screen-2xl px-4 sm:px-6`) into one constant used by both `<header>` and `<main>`, replacing `max-w-6xl`; verify no `max-w-6xl` remains in `frontend/src`
- [x] 1.2 Add a test to `AppShell.test.tsx` that the header and the main element both carry `max-w-screen-2xl`; verify it passes with `pnpm vitest run AppShell`

## 2. Docs and verification

- [x] 2.1 Open a dashboard, `/settings`, `/favourites` and the landing page at 1920 px and 1280 px wide and check the header lines up with the content and no page looks broken; give a page its own inner cap if it does
- [x] 2.2 Add a CHANGELOG entry for the wider layout
- [x] 2.3 Run `make check` and `make test` and verify both pass
