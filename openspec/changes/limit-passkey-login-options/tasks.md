## 1. Backend

- [x] 1.1 Add `passkeyLoginLimiter *ratelimit.Limiter` to `api`, created in `New` (`backend/internal/httpapi/router.go`) with `ratelimit.New(passkeyOptionsPerHour, passkeyOptionsBurst, 10_000)` and constants 120 and 30 in `passkeys.go`
- [x] 1.2 In `passkeyLoginOptions`, reserve for `ratelimit.ClientIP(r, …)` first. Over the limit, set `Retry-After` and return 429 `rate_limited` ("Too many passkey sign-in attempts, try again later") before calling `auth.PasskeyLoginOptions`
- [x] 1.3 Tests in `backend/internal/httpapi`: the 31st request in a burst gets 429 with `Retry-After`, sets no challenge cookie and stores no `webauthn_challenges` row; another IP is still allowed; a sign-in link request from the limited IP still succeeds
- [x] 1.4 `backend/internal/httpapi/openapi/boardly.yaml`: add the 429 response (error schema, `Retry-After` header) to `POST /api/auth/passkey/login/options`

## 2. Frontend

- [x] 2.1 In `LoginPage.tsx`, map an `ApiError` with code `rate_limited` from the passkey button to the `rateLimited` problem with the `Retry-After` seconds, as the email form does
- [x] 2.2 In `startAutofill`, don't show a problem and don't restart autofill when the error is `rate_limited`
- [x] 2.3 Tests (`passkey.test.tsx`): the button with a 429 options response shows "Too many requests. Try again in 40 s."; autofill with a 429 shows nothing and requests options only once

## 3. Docs and verify

- [x] 3.1 CHANGELOG `[Unreleased]` → `### Security`: passkey sign-in options are rate-limited per client IP
- [x] 3.2 Run the backend checks and tests and the frontend checks and tests (`make check && make test`)

Export and import are unaffected; no tasks needed for them.
