# Tasks

## 1. Prerequisites

- [ ] 1.1 Archive the `app-icon` change and verify `openspec list` no longer shows it

## 2. Icons

- [ ] 2.1 Add `frontend/icons/app-icon.svg` (opaque background, fixed dark outline, rose star) and `app-icon-maskable.svg` (drawing inside the central 80 % circle); verify by opening them in a browser
- [ ] 2.2 Add `frontend/scripts/render-icons.mjs` that renders `public/icons/icon-192.png`, `icon-512.png`, `icon-maskable-512.png` and `apple-touch-icon.png` (180) with headless Chromium, plus a `pnpm icons` script; run it and verify the PNGs exist with the right sizes
- [ ] 2.3 Extend the `AppIcon` test so the source SVGs keep the same shapes as `favicon.svg`; verify `pnpm test` passes

## 3. Manifest

- [ ] 3.1 Add `public/manifest.en.webmanifest` and `manifest.de.webmanifest` (id, start_url, scope, display, names, description, colours, icons); verify with Chrome DevTools → Application → Manifest showing no errors
- [ ] 3.2 Link the manifest and the Apple touch icon in `index.html`; choose the manifest language in the pre-paint script and update the link on language switch; verify with an i18n test that switching to German links `manifest.de.webmanifest`
- [ ] 3.3 Add a test that every icon referenced by both manifests and `index.html` exists and has the declared PNG size, and that both manifests share the same id and non-text fields

## 4. Theme colour

- [ ] 4.1 Add `<meta name="theme-color">`, set it in the pre-paint script from a paper-colour map, and update it from the theme module on every theme change; verify with a theme test (White → Fjord changes the meta content)
- [ ] 4.2 Add a test that the paper-colour map matches the `paper` token of each theme in the CSS

## 5. Backend

- [ ] 5.1 Serve `.webmanifest` files with `Content-Type: application/manifest+json` in `internal/web`; verify with a new `web_test.go` case

## 6. Verification

- [ ] 6.1 Build the app, run it behind HTTPS (or on localhost), and verify that Chrome shows its install option, the installed window opens the default dashboard, the title bar follows the theme and no service worker is registered; run `make check` and `make test`
