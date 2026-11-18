## Why

`POST /api/auth/passkey/login/options` needs no sign-in, and every call stores a WebAuthn challenge row in the database. Rows only expire after five minutes, and nothing limits the calls. A script calling it in a loop can grow the database and keep the server busy, without any account.

## What Changes

- The server limits passkey sign-in option requests per client IP: 120 per hour with a burst of 30. That is far more than a person needs on the login page, where autofill and the button each request options once per attempt. Over the limit, it responds 429 `rate_limited` with `Retry-After` and stores no challenge.
- The client IP is found as for sign-in link requests (`app-hosting`, see `fix-proxy-client-ip`).
- Login page: after a 429 from the button, it shows the existing "Too many requests. Try again in N s." message. A 429 during passkey autofill shows no message and does **not** restart autofill, so it can't loop on the limit.
- Passkey registration options (signed-in only) and the verify endpoints are unchanged.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `auth-passkey`: adds a rate limit on passkey sign-in options and how the login page handles it.

## Impact

- Backend: `backend/internal/httpapi/passkeys.go` (limit check), `backend/internal/httpapi/router.go`/`auth.go` (new limiter on `api`), tests.
- Frontend: `frontend/src/features/auth/LoginPage.tsx` (429 handling for the button and autofill) and its tests. No new strings: the existing `login.rateLimited` is reused.
- OpenAPI: `backend/internal/httpapi/openapi/boardly.yaml` gets the 429 response for `POST /api/auth/passkey/login/options`.
- Docs: `CHANGELOG.md`.
- Export and import are unaffected: no user data changes.
