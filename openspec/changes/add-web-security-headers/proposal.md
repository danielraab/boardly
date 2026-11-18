## Why

The web app's HTML and files are served without any security headers: no `Content-Security-Policy`, no protection against framing and no `nosniff`. Today the `SameSite=Lax` session cookie keeps other sites from framing a signed-in Boardly, and React escapes rendered text. Even so, a script policy is the safety net that turns a future XSS bug (in a widget, a note renderer or a dependency) into a blocked script instead of a stolen session. Framing protection should not depend on a cookie attribute alone.

## What Changes

- Every response of the web app (the HTML document, its client-side-route fallback and the static files) gets:
  - `Content-Security-Policy` that allows scripts only from Boardly itself and the inline scripts of the served document (by hash). It also sets `object-src 'none'`, `base-uri 'self'`, `form-action 'self'`, `frame-ancestors 'self'` and `worker-src 'self' blob:`.
  - `X-Frame-Options: SAMEORIGIN` (for browsers without `frame-ancestors`), `X-Content-Type-Options: nosniff` and `Referrer-Policy: strict-origin-when-cross-origin`.
- Images, audio streams, embeds (Spotify), map tiles, styles and `fetch` targets are **not** restricted. The app loads them from origins that users and operators choose (station streams, map style URL, favicons), and restricting them would break features without much security gain.
- The analytics snippet keeps working without extra configuration. The hashes of its inline scripts and the origins of its `<script src>` tags are added to `script-src` automatically.
- New optional setting `BOARDLY_CSP_SCRIPT_SRC`: extra script sources (space-separated), for analytics tools that load more scripts on their own.
- JSON responses of `/api` get `X-Content-Type-Options: nosniff` and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`. Responses that already set their own policy (attachments, images, offline copies) keep theirs.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-hosting`: adds a requirement for the security headers of the web app and the API, and the new setting.

## Impact

- Backend: `backend/internal/web/web.go` (headers, hashes computed from the final document at startup), `backend/internal/config/config.go`, `backend/main.go`, `backend/internal/httpapi` (`WriteJSON`/`WriteError`), and tests.
- Frontend: no code change expected. The policy must be checked against the production build (map with its worker, Spotify embed, offline copy iframe, radio playback, theme script).
- Dev server (Vite on :3000) is unaffected; the headers come from the Go server.
- Docs: `README.md` (configuration table, note on analytics snippets), `CHANGELOG.md`.
- API routes and shapes are unchanged, so the OpenAPI documents are unaffected. Response headers are not part of them.
- Export and import are unaffected: no user data changes.
