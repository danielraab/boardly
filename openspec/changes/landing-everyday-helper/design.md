# Design

## Context

See proposal.md for motivation. Today `Landing.tsx` renders a dashed `aspect-video` frame of empty CSS tiles plus a "Screenshot coming soon" pill, and three feature rows from `features.ts` (`dashboards`, `favourites`, `karakeep`). Copy lives under `landing.*` and `features.*` in EN/DE. The page sits inside `AppShell` at `widePage` (`max-w-6xl`) and inherits the active theme tokens.

Real dashboard DNA to mirror (stylised, not pixel-perfect widgets): favourites explorer rows (folder + chevron + favicon links), link tiles/compact rows, pastel note card with checkboxes (`--color-note-*`), media widget as a short station list — see `ExplorerTree`, `LinkListView`, `NoteCard` / note widgets, `MediaWidgetView`. Do not recreate edit chrome or the app-wide `MiniPlayer` bar in the sketch.

## Goals / Non-Goals

**Goals:**

- One hero composition: brand, everyday-helper tagline, CTAs, then a dominant theme-aware dashboard sketch.
- Sketch cues for favourites, links, notes; media only as a grid widget.
- Feature list = Dashboards, Favourites, Notes; data-driven as today.
- Sketch and copy work in all themes and from 320px width.
- Light motion (about 2–3 cues) that supports hierarchy.

**Non-Goals:**

- Real screenshots or a seeded demo account.
- Full-bleed marketing layout outside `AppShell` / `widePage`.
- Competitor lineage strip or connected-apps as a landing headline.
- Elevating radio/player or passkeys to equal pillars.
- Interactive sketch (no live data, no clickable fake widgets beyond decorative).

## Decisions

### 1. CSS/HTML sketch component, not images

Build a dedicated presentational component (e.g. `DashboardSketch`) under `features/landing/`, composed of token-coloured panels and fake rows. Prefer theme CSS variables over hardcoded hex (except the brand rose star if the AppIcon appears).

**Alternatives:** PNG/WebP screenshots (theme-fragile, sync cost); inline SVG illustration file (harder to theme); reuse real widget components with mock props (heavier, couples landing to widget APIs).

### 2. Sketch layout (readable at hero scale)

Approximate composition inside the existing aspect-video frame (solid border, not dashed):

```
┌─────────────────┬──────────────────────────────┐
│ Favourites tree │ Link tiles / compact links   │
│ (left panel)    ├──────────────┬───────────────┤
│                 │ Note + todos │ Media widget  │
└─────────────────┴──────────────┴───────────────┘
```

Omit: mini-player strip, edit handles, tags, deep nesting, volume UI, third-party logos. Decorative labels inside the sketch use translation keys under `landing.sketch.*` only when visible meaningful text is needed; otherwise `aria-hidden` on the decorative grid and a figure accessible name from i18n.

### 3. Tagline and feature copy

- EN tagline direction: everyday helper — start page, bookmarks and notes, self-hosted.
- DE: equivalent meaning, natural German (not a calque).
- Feature ids: `dashboards`, `favourites`, `notes`. Remove `karakeep` from `features.ts` and locale keys used only by the landing list (or leave unused keys only if shared elsewhere — prefer delete if landing-only).
- Descriptions stay short (one sentence); no "compatible with …" wording.

### 4. Motion

Use CSS (or existing project patterns if any) for: (1) sketch panels fade/slide in once on mount, (2) one accent cue (e.g. soft highlight on a link row or note). Respect `prefers-reduced-motion: reduce` by disabling non-essential motion.

### 5. Tests

Update `Landing.test.tsx`: assert new tagline, sketch accessible name / absence of placeholder string, feature headings Dashboards / Favourites / Notes, no Karakeep/Linkwarden feature card. Keep GitHub and footer tests.

## Risks / Trade-offs

- [Sketch looks too abstract] → Anchor with signature shapes (folder rows, favicon squares, pastel note, station rows) taken from real UI vocabulary.
- [Sketch fights dark/light themes] → Only semantic tokens and note colour variables; manual check across White and one dark theme (e.g. Fjord or Black).
- [Clutter at 320px] → Stack panels in a simplified single-column sketch variant below `sm`, still inside the same aspect frame; avoid horizontal scroll.
- [Overselling unfinished polish] → Stylised sketch is clearly an illustration (accessible name says so), not a live screenshot claim.

## Migration Plan

Ship as a normal frontend change with the binary/image rebuild. No data migration. Rollback = revert the landing feature commit.

## Open Questions

_None._ Exact EN/DE strings are chosen at implementation to match the locked meaning; wording tweaks that keep the same meaning do not need a spec change.
