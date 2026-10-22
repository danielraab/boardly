# Design

## Context

The header link in `AppShell.tsx` holds only the text "Famodash". `index.html` has no icon link. Theme colours are CSS custom properties per `data-theme`; header text uses `--color-ink`. Vite copies `frontend/public/` into the build root, and the Go web handler serves any existing file from `dist` before falling back to `index.html`.

The icon was chosen from three candidates (tiles, bookmarked note, window badge); the window badge won.

## Goals / Non-Goals

**Goals:**
- One drawing of the icon, used both inline in the header and as the favicon.
- Follow every app theme in the header without per-theme assets.

**Non-Goals:**
- PNG favicons, `apple-touch-icon`, web app manifest and other PWA icons. These can follow in a PWA change.
- The icon on the landing page hero or the login page beyond the shared header.
- Switching the favicon with the app theme (it follows the browser's colour scheme instead).

## Decisions

- **Inline React component with `currentColor`.** `AppIcon` renders the 32×32 SVG with strokes and fills set to `currentColor`, so it takes the header's ink colour on every theme. The rose star keeps its fixed colour `#E11D48`, which has at least 3:1 contrast on the paper colour of all seven themes. Alternative: an `<img src="/favicon.svg">` — rejected because an image can't inherit the theme colour.
- **Masked gap around the star.** The window outline is masked by a circle around the star so the badge reads as separate at 16px. The mask id comes from `useId()` so several icons on one page don't share ids.
- **Separate static `favicon.svg`.** A copy of the same paths in `frontend/public/favicon.svg`, with an embedded `<style>` that sets the outline to `#18181b` and, under `@media (prefers-color-scheme: dark)`, to `#f4f4f5` (the white and black theme ink colours). Browser tabs use the browser's colour scheme, not the app theme, so this is the right signal. Alternative: rewrite the favicon from JavaScript on theme change — rejected as extra code for little gain.
- **Decorative in the link.** The icon gets `aria-hidden`, so the link's accessible name stays "Famodash" and existing tests and the spec keep matching.

## Risks / Trade-offs

- [Browsers without SVG favicon support (older Safari) show no icon] → Accept for now; PNG fallbacks belong to the PWA change.
- [Two copies of the paths can drift] → A frontend test compares the path data in `favicon.svg` with the component.
