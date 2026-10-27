# Proposal

## Why

Operators want to see how their instance is used with a self-hosted or hosted analytics tool such as Umami, Plausible or Matomo. These tools need a small script on every page. Today the only way to add one is to edit `frontend/index.html` and build a custom image. The script should be set per deployment at runtime, like `BOARDLY_APP_TITLE`, so the published image can be used as is (GitHub issue #50).

## What Changes

- New optional setting `BOARDLY_ANALYTICS_SNIPPET` (default empty). Its value is HTML, usually one or more `<script>` tags, that the operator copies from their analytics tool.
- The server inserts the snippet unchanged (not escaped) directly before `</head>` of the web app's HTML document on every response that serves the document, including the fallback for client-side routes. The value is read when the server starts; no rebuild is needed.
- An unset value, or one with only whitespace, inserts nothing, so the document is the same as today.
- Because the web app is a single-page app, the document, and therefore the snippet, is loaded once per full page load. Counting navigation between pages inside the app is left to the analytics tool (Umami, Plausible and Matomo can track history changes).
- The Vite dev server doesn't insert the snippet, the same as the app title.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-hosting`: new setting `BOARDLY_ANALYTICS_SNIPPET`; the HTML document carries the snippet before `</head>`.

## Impact

- Backend: `internal/config` (new field and variable), `internal/web` (inserts the snippet into `index.html`, needs the snippet as an extra argument), `main.go` wiring.
- Docs: README configuration table (with a note that the snippet runs with full access to the page, so only trusted code belongs there) and `.env.example`.
- No frontend, API, database or dependency changes. Existing deployments are unchanged.
