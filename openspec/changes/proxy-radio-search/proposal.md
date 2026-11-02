# Proposal

## Why

The "Add station" search calls radio-browser.info straight from the browser, and the results load
each station's logo straight from whatever host the directory lists. Both send the user's IP
address to third parties: the directory, and dozens of unknown logo hosts. An instance open to
others has no legal basis for that under GDPR Art. 6, and the server can easily avoid it, because
it already makes guarded outbound fetches for station logos and stream checks.

## What Changes

- New endpoint `GET /api/media/search?q=` for signed-in users. The server searches radio-browser.info
  (it picks a mirror, sends a descriptive User-Agent and limits searches per user) and returns the
  results in the shape the dialog already uses.
- Logos in the results are served through the server: each result carries a signed
  `/api/media/search/logo` link, which the server fetches with the favicon rules (guarded
  connections, image types only, size limit) and serves without scripts.
- The "Add station" dialog uses these endpoints. The browser no longer contacts radio-browser.info
  or logo hosts while searching.
- **BREAKING (spec)**: the requirement that the search goes "from the browser directly … without
  going through Boardly's server" is reversed.
- The notice under the search says that search terms reach radio-browser.info through this server,
  without the user's IP address.
- Playing streams is unchanged and out of scope.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `media-library`: "Media API" gains the search and search-logo endpoints, "Station search" goes
  through the server, and "External directory notice" gets the new wording.

## Impact

- Backend: new `radiobrowser` client (in `backend/internal/media`), handlers and routes in
  `backend/internal/httpapi`, a per-user rate limiter, an HMAC key generated at start for logo links,
  and reuse of `images.GuardedTransport` and the favicon download rules.
- Frontend: `frontend/src/features/media/radioBrowser.ts` calls `/api/media/search`, and
  `AddStationDialog.tsx` shows the proxied logo. Tests are updated (they currently mock
  `*.api.radio-browser.info`).
- Outbound HTTPS from the server to `*.api.radio-browser.info` and to logo hosts. Without it, the
  search shows "directory not reachable", as it does today.
- Docs: README (adding stations, outbound connections), CHANGELOG.
