# Design

## Context

This builds on `project-foundation`: a single Go binary (`net/http` mux, `apiError` helpers, SQLite with embedded migrations, `FAMODASH_*` config) and a React SPA (TanStack Router, Base UI `Menu` in `UserMenu`, i18next with `en`/`de`). In dev the SPA runs on `:3000` and proxies `/api` to Go, so the browser only ever sees one origin. Requirements are in the specs of this change: `auth-magic-link`, `auth-session`, `user-profile`, `home-page` and `app-shell`.

## Goals / Non-Goals

**Goals:**
- A secure, passwordless sign-in that works for self-hosters with any SMTP provider, and without SMTP in dev.
- A session model that the `passkeys` change can reuse. It records how the session was created, but adds nothing passkey-specific.

**Non-Goals:**
- Passkeys, the step-up window and profile tabs (`passkeys` change).
- Session management UI (listing or revoking other devices).
- Changing the email address, deleting the account, admin users.

## Decisions

### Data model (migration `0002_auth.sql`)

```
users                       sessions                         magic_link_tokens
─────                       ────────                         ─────────────────
id          TEXT PK (uuid)  id_hash      BLOB PK (sha256)    token_hash   BLOB PK (sha256)
email       TEXT UNIQUE     user_id      → users ON DEL CASC email        TEXT  (normalized)
name        TEXT NULL       auth_method  TEXT ('magic_link') locale       TEXT
created_at  TEXT            created_at   TEXT                created_at   TEXT
updated_at  TEXT            last_used_at TEXT                expires_at   TEXT
                            expires_at   TEXT                used_at      TEXT NULL
                            user_agent   TEXT                request_ip   TEXT
```
Timestamps are RFC 3339 UTC text. User IDs are UUIDv4 strings, so they're safe to expose in the API and in later Karakeep-compatible responses. `auth_method` is stored now so that `passkeys` can add `'passkey'` and a step-up column without reshaping the table. Tokens reference the email, not a user, because the account is created only when the link is redeemed.

### Packages
`internal/auth` (tokens, sessions, middleware, handlers), `internal/users` (store + `/api/me`), `internal/mail` (mailer interface, SMTP + log implementations, embedded templates), `internal/ratelimit`. `httpapi` gains `requireUser` middleware that loads the session into the request context.

### Tokens and session IDs
Both are 32 bytes from `crypto/rand`, base64url-encoded, and stored as SHA-256. A plain hash is enough because the inputs are high-entropy, so no slow KDF is needed. Lookup is by hash, so there's no timing-sensitive comparison of secrets. Redemption runs in one transaction: `UPDATE magic_link_tokens SET used_at=? WHERE token_hash=? AND used_at IS NULL AND expires_at>?`, check that exactly one row was affected, upsert the user by email, insert the session. Two concurrent redemptions therefore can't both succeed.

### Scanner-safe link: fragment + explicit confirm
The link is `<FAMODASH_BASE_URL>/login/verify#token=…`. The fragment never reaches the server, proxies or `Referer` headers, so tokens don't end up in access logs. The verify page reads it, removes it from the address bar with `history.replaceState`, and only POSTs it after the user clicks "Complete sign-in". That one click protects against scanners that render the page and run its JavaScript (e.g. Defender Safe Links detonation).
- *Alternative*: auto-submit on page load. It saves a click, but some scanners run JS and would use up or redeem the link. Rejected.
- *Alternative*: `GET /api/auth/verify?token=` with a server redirect. Scanners would follow it. Rejected.

### Session handling
The cookie is `famodash_session`, with attributes as in the spec. `Secure` is set when `FAMODASH_BASE_URL` is `https`. `expires_at = last_used_at + 30d`. The middleware updates `last_used_at`/`expires_at` only when the last update is older than one hour, which keeps writes down. The cookie's `Max-Age` is renewed at the same time. A janitor goroutine deletes expired sessions and tokens (tokens that expired more than 24h ago) every hour.

### CSRF: SameSite=Lax + JSON content type + Origin check
A middleware applies to all non-GET/HEAD `/api` requests. It requires `Content-Type: application/json`, even for bodyless endpoints like logout (the SPA always sends it). If `Origin` is present, it must equal the origin of `FAMODASH_BASE_URL`. Together these block form-based and cross-origin fetch attacks without a token round-trip.
- *Alternative*: a double-submit CSRF token. It adds state and client complexity for little gain in a same-origin SPA. Rejected.

### Rate limiting
- Per email: the limits are counted from `magic_link_tokens` (`created_at` within 60 s / 1 h). That survives restarts and needs no extra state.
- Per IP: an in-memory `golang.org/x/time/rate` limiter keyed by IP (20/h, burst 5), with an LRU-bounded map.
- Client IP: `RemoteAddr`, or the first `X-Forwarded-For` hop when `FAMODASH_TRUST_PROXY_HEADERS=true` (needed behind a reverse proxy in Docker).
- `Retry-After` is computed from the earliest moment the request would pass.

### Mail
The library is `github.com/wneessen/go-mail`: maintained, supports STARTTLS and implicit TLS, builds multipart messages. Mail is sent synchronously inside the request with a 10 s timeout. The request still returns 202 when delivery fails: the error is logged, and the response must not reveal anything. Templates are embedded `text/template` + `html/template` files per locale (`mail/templates/{en,de}/magic_link.{txt,html}`). A missing locale falls back to `en`.

| Variable | Default | Meaning |
|---|---|---|
| `FAMODASH_BASE_URL` | `http://localhost:3000` | public URL; used for links, Origin check, `Secure` |
| `FAMODASH_SMTP_HOST` | (empty → log mailer) | SMTP server |
| `FAMODASH_SMTP_PORT` | `587` | |
| `FAMODASH_SMTP_USERNAME` / `_PASSWORD` | empty | auth (optional) |
| `FAMODASH_SMTP_FROM` | `Famodash <noreply@localhost>` | sender |
| `FAMODASH_SMTP_TLS` | `starttls` | `starttls`, `tls` or `none` |
| `FAMODASH_TRUST_PROXY_HEADERS` | `false` | use `X-Forwarded-For` |

### Frontend: TanStack Query for server state, session in router context
`@tanstack/react-query` holds `['me']`, fetched through `GET /api/me`: 401 maps to `null`, `retry: false`, `staleTime` 5 min. A small `api.ts` fetch wrapper always sends `Content-Type: application/json`, parses the error format into an `ApiError`, and on 401 sets `['me']` to `null`. Logout calls the endpoint, then `queryClient.clear()`, then navigates to `/`.

The router gets `queryClient` in its context. Protected routes (`/profile`) use `beforeLoad: ensureQueryData(['me'])` and, when the result is null, `throw redirect({ to: '/login' })`. `/login` redirects signed-in users to `/`. `/` picks `<Home/>` or `<Landing/>` from the `me` query and renders a neutral skeleton while it's pending, so the landing page doesn't flash for signed-in users.
- *Alternative*: a hand-written React context with `useEffect` fetching. That means re-implementing caching, invalidation and pending states, and dashboards will need Query anyway. Rejected.

### UI pieces
- `/login`: one email input (`type=email`, `autocomplete="username"`, so `passkeys` can append `webauthn`), submit, then a "Check your inbox" state with a 60 s resend countdown. On 429, the countdown uses `Retry-After`.
- `/login/verify`: shows the "Complete sign-in" button, then a spinner, then success (navigate `/`) or an error with a link to `/login`. If there's no token in the fragment, it shows the error immediately.
- `UserMenu`: three states (`pending` → loading item, `null` → Sign in, user → name/email header, Profile, Sign out). The header uses Base UI `Menu.Group` + `Menu.GroupLabel` so it isn't focusable as an item.
- `/profile`: a header card with the email and an inline name form (input, Save button, disabled while unchanged). On success it calls `setQueryData(['me'], user)`.
- Home: greeting plus a placeholder card ("Your dashboards will appear here").

## Risks / Trade-offs

- [Open signup means anyone can make the server send mail] → Per-email and per-IP limits. The mail content can't be controlled by the requester (no name, no free text). Self-hosters can additionally rate-limit at the reverse proxy.
- [Synchronous SMTP makes the request slow with a slow server] → 10 s timeout. A background queue is overkill at this scale. We can revisit it if needed.
- [Timing differences between existing and new addresses] → Both paths do the same work, because the account isn't created on request. The only work is token insert + send.
- [The link was opened on a different device than the one that requested it] → That device gets signed in. This is accepted behavior for magic links and is documented in the mail text ("opens Famodash in this browser").
- [Wrong `FAMODASH_BASE_URL` behind a proxy breaks links and the Origin check] → Log the effective base URL at startup and document it in the README.
- [The in-memory IP limiter resets on restart] → Acceptable. The per-email limits are persistent.

## Migration Plan

Migration `0002_auth.sql` only adds tables. No existing data is affected. Rollback means restoring the DB file.
