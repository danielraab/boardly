## Why

With `BOARDLY_TRUST_PROXY_HEADERS=true`, the server takes the client IP from the *leftmost* `X-Forwarded-For` entry. Common reverse proxies (nginx `$proxy_add_x_forwarded_for`, Traefik, Caddy) append the real client address to whatever the client sent. The leftmost entry is therefore chosen by the client. Sending a different fake address with every request gets around the per-IP limit of 20 sign-in link requests per hour. That lets anyone make the server send sign-in emails to any number of addresses, which hurts the reputation of the operator's mail sender.

## What Changes

- Behind trusted proxies, the client IP is the `X-Forwarded-For` entry that the outermost trusted proxy added. The server counts from the right, by the number of trusted proxies, and no longer takes the leftmost entry.
- New optional setting `BOARDLY_TRUSTED_PROXY_COUNT` (default `1`, 1–10) gives the number of reverse proxies in front of Boardly that append to `X-Forwarded-For`, for setups such as a CDN in front of nginx. It is only read when `BOARDLY_TRUST_PROXY_HEADERS=true`.
- All `X-Forwarded-For` header lines count as one list. An entry that isn't a valid IP address falls back to the connection's address.
- README: the configuration table and the reverse-proxy section explain the new setting and the "rightmost" rule.
- Not breaking for the usual setup (one proxy). With two or more proxies, rate limits now apply to the outer proxy until `BOARDLY_TRUSTED_PROXY_COUNT` is set. The CHANGELOG says so.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-hosting`: adds a requirement for how the client IP is found behind reverse proxies, with the new setting.

## Impact

- Backend: `backend/internal/ratelimit/ratelimit.go` (`ClientIP`), `backend/internal/config/config.go`, `backend/main.go`, `backend/internal/httpapi/router.go` and `auth.go` (pass the proxy count), and their tests.
- Docs: `README.md` (configuration table, reverse-proxy notes), `CHANGELOG.md`.
- API routes and shapes are unchanged, so the OpenAPI documents are unaffected.
- Export and import are unaffected: no user data changes.
