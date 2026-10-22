# Design

## Context

The app icon exists as `frontend/public/favicon.svg` (transparent, outline switches with `prefers-color-scheme`) and as the `AppIcon` component. Vite copies `public/` into `backend/internal/web/dist`, which Go embeds and serves; files outside `assets/` get no `Cache-Control` header, and `index.html` gets `no-cache`. Go's built-in MIME table has no entry for `.webmanifest`.

All UI text is translated (en, de). i18next picks the language from localStorage first, then the browser languages. The theme is applied by an inline script in `index.html` before first paint.

## Goals / Non-Goals

**Goals:**
- Meet the installability criteria of Chromium browsers so their default install UI appears.
- Look right once installed: proper icons on Android (maskable), iOS and desktop, title bar in the theme's colour.

**Non-Goals:**
- A custom install button, banner or `beforeinstallprompt` handling.
- A service worker, offline pages, caching, push notifications or background sync.
- Window Controls Overlay, file or protocol handlers.

## Decisions

### Manifest without a service worker
Current Chromium browsers treat a page as installable with a manifest (name, icons 192 + 512, `start_url`, `display`) served over HTTPS, without a service worker. Leaving it out keeps "no offline" literally true and avoids stale-bundle bugs that a cache would introduce.
*Alternative*: a no-op service worker "just in case": rejected, as it adds an update lifecycle for no benefit.

### Two static manifests, chosen before first paint
`public/manifest.en.webmanifest` and `public/manifest.de.webmanifest` hold identical fields except the translated `name`, `short_name` and `description` (and, in the follow-up change, shortcut labels). The inline script in `index.html` that applies the theme also sets the `<link rel="manifest">` href from the stored language, then from `navigator.languages`, with English as the fallback, mirroring i18next's detection. When the user switches language in the app, the link's href is updated too. Both share `id: "/"`, so the installed app is the same app whichever language it was installed in.
*Alternative*: a manifest rendered by the Go server from `Accept-Language`: rejected, because it ignores the language chosen in the app and moves frontend content into the backend.

### Manifest fields
`id: "/"`, `start_url: "/"`, `scope: "/"`, `display: "standalone"`, `background_color` and `theme_color` from the White theme's `paper` colour (the app re-colours the title bar at runtime), `icons` with `any` 192/512 and `maskable` 512.

### Icons: committed PNGs rendered from source SVGs
New source SVGs (`frontend/icons/app-icon.svg` with a solid background and `app-icon-maskable.svg` with the drawing shrunk into the inner 80 % circle) use the same drawing as `favicon.svg`, with a fixed dark outline on a light background. A script (`frontend/scripts/render-icons.mjs`) renders them to PNG with headless Chromium (`--headless --screenshot --window-size=N,N`), which is already available on dev machines. The PNGs are committed to `public/icons/`, so neither the normal build nor the container build needs Chromium. A test checks that every icon the manifests and `index.html` reference exists and has the declared pixel size (read from the PNG header).
*Alternative*: `sharp` or `resvg` as a dev dependency: rejected, as it adds a native dependency for files that change rarely.

### `theme-color` follows the theme
`index.html` gets a `<meta name="theme-color">`. The pre-paint script sets it to the active theme's `paper` colour, and the theme module updates it whenever the theme changes (including System switching with the OS). The paper colours are duplicated in a small map next to the existing theme ids. A test compares the map with the CSS tokens.

### Content type
`internal/web` sets `Content-Type: application/manifest+json` for `.webmanifest` files before calling `http.ServeContent`, which keeps an already-set content type.

## Risks / Trade-offs

- [Some browsers still require a service worker for the install prompt] → Accepted. Chrome and Edge, the main targets, don't. If a target turns out to need one, a service worker without a fetch handler can be added later without changing the specs.
- [iOS ignores the manifest prompt, and Firefox desktop can't install] → Accepted. iOS users can still use "Add to Home Screen", which picks up the Apple touch icon and manifest.
- [The PNGs drift from the SVG drawing] → The render script is the only way to produce them, and the icon test compares the drawing's shapes with `favicon.svg` just as the existing `AppIcon` test does.
- [The installed app keeps its manifest language until the browser refreshes the manifest] → Acceptable. Chrome rechecks the manifest periodically.
