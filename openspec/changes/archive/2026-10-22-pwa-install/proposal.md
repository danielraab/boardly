# Proposal

## Why

Famodash is a start page that people open many times a day, but it only runs in a browser tab. Making it an installable web app lets browsers offer to install it, gives it its own window, launcher icon and title bar, and is the base for the share target and app shortcuts (change `pwa-share-and-shortcuts`).

## What Changes

- Add a web app manifest so that browsers which support it (Chrome, Edge, Android browsers) offer to install Famodash with their own built-in prompt or address-bar button. Famodash shows no install button or banner of its own.
- The installed app opens `/` (the default dashboard) in a standalone window.
- Add home-screen icons generated from the app icon: PNGs at 192 and 512 pixels on an opaque background, a maskable variant whose drawing stays inside the safe zone, and a 180-pixel Apple touch icon for iOS "Add to Home Screen".
- The manifest's name and description follow the app language (English or German).
- The browser's title bar colour (`theme-color`) follows the active theme.
- The server serves the manifest with its proper content type.
- No service worker and no offline support: without a connection the installed app behaves like a browser tab without one.

## Capabilities

### New Capabilities

- `pwa`: installability of the web app: manifest, home-screen icons, standalone window, title bar colour, and the explicit absence of offline support.

### Modified Capabilities

(none)

## Impact

- Frontend: `index.html` (manifest link, Apple touch icon, `theme-color`, pre-paint choice of the manifest language), new static files in `public/` (two manifests, PNG icons), a source SVG for the opaque icons and a script that renders the PNGs, a hook that keeps `theme-color` in sync with the theme.
- Backend: `internal/web` serves `.webmanifest` files as `application/manifest+json`.
- Depends on the `app-icon` change (drawing of the icon); archive it first.
