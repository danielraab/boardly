# Tasks

## 1. Copy and feature data

- [x] 1.1 Update `landing.hero.tagline` in `en.json` and `de.json` to the everyday-helper meaning (start page, bookmarks and notes, self-hosted); verify both locale files keep matching keys (i18n tests pass)
- [x] 1.2 Replace landing feature `karakeep` with `notes` in `features.ts` (id + icon) and under `features.*` in both locales; remove unused `features.karakeep` landing keys; verify the feature list length stays 3 with ids `dashboards`, `favourites`, `notes`
- [x] 1.3 Add `landing.sketch` accessible-name (and any visible sketch labels if needed) in EN/DE; remove `landing.screenshot.placeholder`; verify matching keys in both locales

## 2. Product sketch

- [x] 2.1 Add `DashboardSketch` (or equivalent) under `features/landing/` with theme-token panels for favourites tree, links, note+todos, and a media widget — no mini-player bar, no third-party marks, no placeholder pill; verify it renders inside a fixed-aspect figure with the sketch accessible name
- [x] 2.2 Wire the sketch into `Landing.tsx` in place of the dashed empty-tile placeholder; verify "Screenshot coming soon" / "Screenshot folgt" no longer appear
- [x] 2.3 Add light load-in / accent motion that respects `prefers-reduced-motion`; verify reduced-motion disables non-essential animation
- [x] 2.4 Check the sketch at ~320px and in at least one light and one dark theme; verify no horizontal scroll and readable contrast

## 3. Feature overview and tests

- [x] 3.1 Confirm feature section still data-driven from `features.ts` with Dashboards, Favourites, Notes copy
- [x] 3.2 Update `Landing.test.tsx` for the new tagline, sketch (accessible name / no placeholder), and feature headings; verify the landing Vitest file passes
- [x] 3.3 Keep GitHub hero link and open-source footer behaviour; verify existing GitHub/footer assertions still pass

## 4. More feature cards and copy

- [x] 4.1 Add `apps` (Karakeep, Linkwarden, Memos apps and browser extensions) and `radio` feature cards to `features.ts` with icons; verify five entries in order `dashboards`, `favourites`, `notes`, `apps`, `radio`
- [x] 4.2 Rewrite all five card titles and descriptions in EN/DE following the copy rules in design.md; verify matching keys and that claims match current app behaviour
- [x] 4.3 Update `Landing.test.tsx` for the five headings and the apps/radio cards; verify the landing Vitest file passes

## 5. Wrap-up

- [x] 5.1 Run `make check` and `make test` and verify both pass
