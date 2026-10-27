# Proposal

## Why

The app shell caps the header and page content at `max-w-6xl` (1152 px). On common desktop screens (1440 px and wider) a dashboard leaves a wide empty margin on both sides, although a start page is exactly where screen space pays off. As a first, easily reversible step, the global cap is raised.

## What Changes

- The header and the main content area of the app shell share one centred container that is at most 1536 px wide (`max-w-screen-2xl`) instead of 1152 px (`max-w-6xl`). Side padding stays as it is.
- This applies to every page that uses the app shell. Pages that already limit their own content (settings forms, help texts) keep their narrower widths inside the wider container.
- No per-route or per-dashboard width setting yet; those stay open as later options.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-shell`: new requirement "Content width" that fixes the shared container width of the header and page content.

## Impact

- Frontend: `frontend/src/components/AppShell.tsx` (two class names), a test for the shared width.
- No backend, API or translation changes.
