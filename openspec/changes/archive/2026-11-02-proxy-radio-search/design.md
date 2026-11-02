# Design

## Context

`frontend/src/features/media/radioBrowser.ts` fetches `https://all.api.radio-browser.info/json/servers`
once per page load, picks a random mirror (falling back to `de1.api.radio-browser.info`) and calls
`/json/stations/search`. `toResult` maps raw stations to `StationResult`. `AddStationDialog.tsx`
renders `<img src={station.logoUrl}>` for `https` logos and posts `logoUrl` with `POST /api/media`.
The server then fetches the logo with `Favicons.FetchLogo`, which uses `images.GuardedTransport`,
`ReadLimited` and `Sniff`.

## Goals / Non-Goals

**Goals:**
- No browser request to a third party during search.
- The existing `StationResult` shape and dialog behaviour stay as they are, so the frontend change
  is small.

**Non-Goals:**
- Proxying audio streams.
- Caching search results on the server.
- Saving logos of results that aren't added.

## Decisions

**Server-side client in `internal/media` (`radiobrowser.go`).**
- Mirror selection: fetch `https://all.api.radio-browser.info/json/servers`, cache the list for 1
  hour, pick at random per search, and fall back to `de1.api.radio-browser.info` when the list
  fails. This is the same behaviour as the frontend today, moved to the server.
- Request: `GET https://<mirror>/json/stations/search?name=<q>&limit=30&order=clickcount&reverse=true&hidebroken=true`
  with `User-Agent: Boardly (+https://github.com/danielraab/boardly)` (radio-browser asks for a descriptive agent; Boardly has no version string at runtime), a 10 s timeout
  and a 1 MB body limit. Nothing about the user is forwarded: no headers from the incoming request,
  and in particular no `X-Forwarded-For`.
- Transport: `images.GuardedTransport(cfg.FetchAllowPrivate)`. The directory is public anyway, and
  reusing the guarded transport keeps a single outbound policy and honours the operator's setting
  for tests and dev.
- `toResult` moves from TypeScript to Go with the same rules: `url_resolved` before `url`, trimmed
  name falling back to the URL, up to 3 tags, bitrate 0 → null, and `playable` = the URL starts
  with `https://`.

**Logo links signed with HMAC, not an allow-list in memory.** For each result with a logo URL,
the server returns `logo = /api/media/search/logo?u=<url>&s=<sig>`, where
`sig = base64url(HMAC-SHA256(key, userID + "\n" + url))`. The key is 32 random bytes generated at
start, so links stop working after a restart. That's acceptable, because they only live as long as
an open dialog. Alternatives:
- An in-memory map from user to recent logo URLs. It needs expiry and grows with use; HMAC is
  stateless.
- An open `?url=` proxy for signed-in users. Any account could then make the server fetch
  arbitrary URLs. The guard blocks internal addresses, but signing limits the server to URLs the
  directory actually returned.
- Dropping logos from results. That is simpler, but the results become harder to scan, and the
  fetch path already exists.

The logo handler checks the session and the signature (constant-time), then fetches with the
favicon download rules. That means refactoring `Favicons.download` (or adding an exported
`Favicons.FetchImage(ctx, url) ([]byte, contentType, error)`) so that it doesn't save to the image
store. It responds with the sniffed type, `nosniff`, `CSP: sandbox` and
`Cache-Control: private, max-age=3600`. On any failure it responds 404 with no body detail.

**Rate limit.** A new `ratelimit.New(...)` keyed by user ID: 30 searches per minute with a burst
of 10, which is generous for typing after a pause. The logo endpoint isn't limited separately; one
search makes at most 30 logo links, and the browser caches them.

**Frontend.** `searchStations(query, signal)` becomes
`GET /api/media/search?q=` through the app's API helper (same-origin, with the session cookie),
returning `StationResult[]` with the extra `logo` field. The 502 response maps to the existing
"directory isn't reachable" state and 429 to the new "too many searches" text. `pickServer`,
`resetRadioBrowser` and the mirror constants are removed. The dialog renders `<img src={station.logo}>`,
keeps `logoFailed`, and still posts `logoUrl`.

## Risks / Trade-offs

- [The server's IP address is rate-limited or blocked by radio-browser.info when many users
  search] → The per-user limit and the descriptive User-Agent help. Failures show the existing
  "not reachable" message, and "Enter URL" keeps working.
- [The logo fetch is used to make the server fetch arbitrary URLs] → It only fetches URLs that are
  signed for this user, over the guarded transport, and returns only images, sniffed, under the
  size limit.
- [A restart breaks open logo links] → The radio icon is shown, and the next search makes new links.
- [Search terms still go to radio-browser.info] → They are no longer linked to the user's IP
  address. The notice says so, and the operator's privacy policy lists radio-browser.info as a
  recipient.

## Migration Plan

Deploy backend and frontend together, as one binary. No data migration. The server needs outbound
HTTPS to `*.api.radio-browser.info`, which it already needs for logos and stream checks. Rolling
back restores the direct calls from the browser.
