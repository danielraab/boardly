# Design

## Context

`AppShell.tsx` defines one `container` constant (`mx-auto w-full max-w-screen-2xl px-4 sm:px-6`) for `<header>` and `<main>`. Pages render directly into `<main>` and set their own outer width, if any:

- most pages: a bare `<section className="pt-12">` or similar, so 1536 px
- `NotFound`: `max-w-prose`, left-aligned
- `LoginPage`, `VerifyPage`, `SharePage`: `mx-auto max-w-md`, centred
- `NoteEditorPage`: a card with no cap; `NoteDialog` already uses 48rem for the same editor

Shared Tailwind class strings live as exported constants (`buttonStyles.ts`, `dialogStyles.ts`).

## Goals / Non-Goals

**Goals:**
- One place that defines the four widths, so every page picks a named width instead of a raw `max-w-*`.
- Each page in the spec's table uses its width on its outermost element.

**Non-Goals:**
- Changing the shell container, the header or the side padding.
- Restyling anything inside a page (cards, inputs, inner `max-w-xl` / `max-w-[60ch]` caps stay).
- A user or per-dashboard width setting.

## Decisions

- **Class constants, not a `<Page>` component.** A new `frontend/src/components/pageWidths.ts` exports `fullPage`, `widePage`, `narrowPage` and `focusPage`, in the style of `buttonStyles.ts`. Pages add the constant to the `className` of the element they already render, for example `` `${narrowPage} pt-12` ``. A wrapper component would add a DOM level to every page and would duplicate what each page's root element already does. Alternative considered: a route-level wrapper in each route file; rejected because some routes render different pages (e.g. `/` renders `Landing` or `Home`, and `Home` renders a dashboard or a greeting), so the width belongs to the page component.
- **Values:** `full` = `w-full`, `wide` = `mx-auto w-full max-w-6xl` (1152), `narrow` = `mx-auto w-full max-w-3xl` (768), `focus` = `mx-auto w-full max-w-md` (448). `w-full` keeps flex or grid children from shrinking the page below its cap. These are Tailwind's defaults, so no theme extension is needed.
- **Everything centred.** Login, Verify and Share already centre; the other pages follow. The header stays at the container width, so on non-full pages the title and the app icon only line up while the viewport is narrower than the page width. This is accepted (see spec "Content width").
- **Existing caps are replaced, not stacked.** `NotFound`'s `max-w-prose` becomes `narrowPage`; Login/Verify/Share's `mx-auto max-w-md` becomes `focusPage`. Inner caps on paragraphs and forms stay.
- **Home:** the `PageSkeleton` on `/` and a default dashboard use `fullPage` (the dashboard view already does); the `Greeting` without a default dashboard uses `narrowPage`.
- **Note editor:** the width goes on the route-level page only, not on the shared editor component, so `NoteDialog` keeps its own width. `NoteDialog` renders `NoteEditorPage` with `onClose` set, so the class is applied only when `onClose` is not set (the same condition the page already uses for its `mt-6 mb-16`).
- **Dashboards:** `DashboardPage` (view) and the `LayoutEditor` root (edit mode) both use `fullPage`, so entering edit mode doesn't change the width.
- **Tests:** one test per width group checks that a representative page's root carries the expected constant (settings → narrow, favourites → wide, login → focus, a dashboard → full). jsdom has no layout, so tests check classes, as the existing `AppShell` test does.

## Risks / Trade-offs

- [Title no longer under the logo on wide screens for most pages] → Accepted by the user; full pages, where alignment matters most, keep it.
- [Connected-apps rows or the favourites tree feel cramped at their new width] → Check at 1920 and 1280 px during apply; the spec allows no restyling here, so a problem goes into a follow-up change rather than this one.
- [A new page forgets to pick a width] → The spec table names every page; the constants make the choice obvious in review.
