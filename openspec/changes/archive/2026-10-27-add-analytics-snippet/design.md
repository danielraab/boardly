# Design

## Context

`internal/web` already rewrites the embedded `index.html` once at startup to fill in the app title (see the `custom-app-title` change) and serves the rewritten bytes for `/`, `/index.html` and the client-side route fallback. The server sends no Content-Security-Policy for the HTML document, and the web app has no service worker, so the document always comes fresh from the server (`Cache-Control: no-cache`).

## Goals / Non-Goals

**Goals:**
- Any analytics tool that is set up by pasting HTML into `<head>` works without rebuilding the image.

**Non-Goals:**
- Built-in support for a specific tool, a consent banner, or tracking of navigation inside the single-page app (the tools do this themselves).
- Reading the snippet from a file (`*_FILE` variables); no other setting supports that either.
- Injecting the snippet in the Vite dev server.

## Decisions

- **Raw HTML instead of a script URL.** Tools differ: Umami and Plausible use one `<script src>` with `data-*` attributes, Matomo and Google Analytics need an inline script. One HTML value covers all of them. Alternative: `BOARDLY_ANALYTICS_SCRIPT_URL` (+ attributes); rejected because it can't express inline scripts.
- **Inserted before `</head>`, in the same startup rewrite as the app title.** The rewrite runs once, so there is no per-request cost, and the snippet reaches every response that serves the document. Placing it last in `<head>` keeps it after the theme script, which must stay first to avoid a flash before first paint. The insertion uses the first `</head>` in the file; the built `index.html` has exactly one.
- **No escaping, no validation.** The value comes from the operator, who controls the server anyway; escaping would break every snippet. The README warns that the snippet runs with full access to the page.
- **`web.Handler` / `NewHandler` take the snippet as an extra argument** next to the app title, matching the existing style instead of introducing an options struct for two values.

## Risks / Trade-offs

- [A malformed snippet (e.g. an unclosed `<script>`) breaks the page] → Documented as operator responsibility; removing the variable and restarting restores the page.
- [A future CSP for the HTML document would block third-party scripts] → Whoever adds a CSP must take the snippet into account; noted here.
- [Analytics sees signed-in pages and URLs that contain user slugs] → The operator chooses the tool; the README mentions that the snippet runs on every page, including signed-in ones.
