## Context

`web.NewHandler` serves the embedded Vite build. At startup, `titledFiles` rewrites `index.html` with the app title, map style, legal links and the analytics snippet (`BOARDLY_ANALYTICS_SNIPPET`, raw HTML inserted before `</head>`). The built `index.html` has one inline script (theme and manifest language before first paint). The app also uses:
- a MapLibre worker loaded from a bundled same-origin URL (`setWorkerUrl`),
- the Spotify embed iframe (`https://open.spotify.com/embed/…`),
- a same-origin iframe for offline articles (`/api/favourites/{id}/offline/article`, which has its own sandbox policy),
- `<audio>` with stream URLs on any origin, images from `/api/images` and from anywhere,
- React `style` attributes.

## Goals / Non-Goals

**Goals:**
- Block injected and third-party scripts (the main XSS impact) without breaking any feature.
- Forbid framing by other sites.
- Zero configuration for the default setup and for the common analytics snippets (a `<script src>` tag, or an inline snippet).

**Non-Goals:**
- Restricting `img-src`, `media-src`, `connect-src`, `frame-src` or `style-src`. Stream, map and image origins are user-chosen. Inline `style` attributes are used throughout.
- HSTS. The TLS-terminating reverse proxy owns that, and a wrong `max-age` from the app is hard to undo. The README recommends setting it at the proxy.
- Nonces. The document is the same for every request and cached with `no-cache`, so hashes computed once at startup are enough and simpler.
- `Permissions-Policy`. The embed iframe delegates features via `allow`, and a header would have to repeat that list.

## Decisions

- **Policy** (one line, computed once at startup):
  `default-src 'self'; script-src 'self' <hashes> <snippet origins> <BOARDLY_CSP_SCRIPT_SRC>; worker-src 'self' blob:; img-src * data: blob:; media-src * data: blob:; connect-src *; frame-src *; font-src * data:; style-src * 'unsafe-inline'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'`.
  Spelling out the open directives instead of omitting them keeps `default-src 'self'` from closing them silently.
- **Hashes from the final document.** After all rewrites, parse the served `index.html` with `golang.org/x/net/html` (already a dependency). Add `'sha256-…'` for every inline `<script>` without `src`. Add the origin (`scheme://host[:port]`) of every absolute `<script src>`. This covers the theme script and any analytics snippet with no configuration.
  *Alternative:* ask operators to list snippet sources by hand. That is more error-prone, and a missed source silently breaks their analytics.
- **`BOARDLY_CSP_SCRIPT_SRC`**: space-separated sources added to `script-src` as given. Values containing `;` or `,` are rejected at startup, so no other directive can be injected.
- **Headers on every web response**, including static assets. Only the document needs the CSP, but setting it everywhere is simpler. A policy on a JS or CSS response has no effect and does no harm.
- **API JSON**: set `nosniff` and `default-src 'none'; frame-ancestors 'none'` in `WriteJSON`/`WriteError` (and the mux error writer), only when the handler hasn't set a CSP. File-serving handlers set their own before writing.
- **`'self'` in `frame-ancestors`**: Boardly frames its own offline article page. Nothing frames the app document itself, but `'self'` costs nothing and allows that in the future.

## Risks / Trade-offs

- [A library uses `eval` or `new Function` and breaks under the policy] → Task: run the production build and use the map, a Spotify embed, an offline copy, radio playback and theme switching with the browser console open, before merging. MapLibre 6 needs no `unsafe-eval`. Its worker is a same-origin module.
- [An analytics tool loads scripts from another origin than its tag] → `BOARDLY_CSP_SCRIPT_SRC` and a README note. The failure shows as a CSP error in the console.
- [Browser extensions that inject scripts] → Extensions aren't bound by the page's CSP.
