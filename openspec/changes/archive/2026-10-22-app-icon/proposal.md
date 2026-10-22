# Proposal

## Why

Famodash has no icon: the browser tab shows the default blank page icon and the header shows only the word "Famodash". An app icon makes the tab easy to find among others and gives the product a recognisable mark.

## What Changes

- Add the app icon ("window badge"): a dashboard window holding a widget block and note lines, with a rose star badge cut into its top-right corner. It stands for the three core parts of Famodash: dashboards, favourites and notes.
- The icon is drawn in one colour on a transparent background, plus the rose star as the only accent, so it works on every theme.
- Show the icon directly before the product name in the global header, drawn in the theme's text colour.
- Use the icon as the favicon. Its outline is dark in a light browser and light in a dark browser.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `app-shell`: the global header shows the app icon before the product name, and every page uses the app icon as its favicon.

## Impact

- Frontend: new icon component, `AppShell` header, `index.html` favicon link, new static `favicon.svg` in the Vite public folder.
- Backend: none. The built favicon ends up in the embedded `dist` and is served by the existing static file handler.
