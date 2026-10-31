# Proposal

## Why

The signed-out landing page still shows a dashed screenshot placeholder and three thin feature blurbs (including a connected-apps card). Visitors do not see Boardly as an everyday home base — start page, bookmarks and notes in one self-hosted place. The placeholders were always meant to be replaced; the product is ready enough to show a real overview.

## What Changes

- Replace the screenshot placeholder with a **theme-aware product sketch** of a dashboard that shows favourites (tree), links (tiles or compact list), notes (pastel card with todos), and a small media widget (station list/tiles). No fixed mini-player bar; media is only another widget in the grid.
- Reword the hero **tagline** around Boardly as the everyday helper: start page, bookmarks and notes, self-hosted.
- Replace the feature list with five benefit-led cards: **Dashboards**, **Favourites** and **Notes** first, then **connected apps** (Karakeep, Linkwarden, Memos apps and browser extensions save into Boardly) and **radio** (favourite stations one click away). Connected apps are a supporting card, not the hero headline.
- Keep hero structure: app title, tagline, Sign in + GitHub, sketch, feature list, open-source footer.
- No competitor lineage strip on the hero. Do not market Boardly primarily as a glue layer for Homepage / Karakeep / Linkwarden / Memos; app compatibility appears only as one feature card.
- Add light, intentional motion on the sketch (load-in / one accent cue) without clutter.
- Update EN/DE copy and landing tests accordingly.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `landing-page`: replace screenshot and feature placeholders with a product sketch and a benefit-led feature overview (everyday capabilities, connected apps, radio); update purpose and requirements so the page presents the everyday-helper story instead of temporary placeholders.

## Impact

- Frontend only: `frontend/src/features/landing/` (Landing UI, feature list, new sketch component), `frontend/src/locales/en.json` and `de.json`, `Landing.test.tsx`.
- No backend or API changes.
- Sketch must use theme tokens (`paper`, `surface`, `ink`, `muted`, `line`, `accent`, note colours) so it works across all themes.
- Spec purpose text for `landing-page` stops describing the page as placeholder-only.
