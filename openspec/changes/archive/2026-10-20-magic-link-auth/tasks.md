# Tasks

## 1. Data and configuration

- [x] 1.1 Add migration `0002_auth.sql` (`users`, `sessions`, `magic_link_tokens` with indexes on `sessions.user_id` and `magic_link_tokens(email, created_at)`). Verify the migration test applies it on a fresh DB and that FK cascade deletes sessions with their user
- [x] 1.2 Extend `internal/config` with `FAMODASH_BASE_URL`, `FAMODASH_SMTP_*` and `FAMODASH_TRUST_PROXY_HEADERS` (validation: base URL is an absolute http(s) URL, TLS mode is one of three values). Verify with table tests

## 2. Mail

- [x] 2.1 Implement the `mail.Sender` interface with a log implementation and an SMTP implementation (`wneessen/go-mail`, 10 s timeout) plus the startup warning when SMTP is disabled. Verify a unit test for the log sender and a manual send against a local SMTP catcher (e.g. `mailpit`)
- [x] 2.2 Add embedded `en`/`de` magic-link templates (txt + html) and a renderer with locale fallback. Verify golden-output tests for both locales and for an unknown locale → `en`

## 3. Magic-link backend

- [x] 3.1 Implement email normalization and validation and `POST /api/auth/magic-link` (token generation, hash storage, 15 min expiry, mail send, identical 202). Verify `httptest` tests for valid, invalid email and known vs. unknown address (identical bodies)
- [x] 3.2 Implement rate limiting (per-email from the DB, per-IP in-memory, proxy-header option, `Retry-After`). Verify tests for an immediate repeat → 429, the sixth request in an hour → 429, and the IP limit
- [x] 3.3 Implement `POST /api/auth/magic-link/verify` with transactional single use, user upsert and session creation. Verify tests for first sign-in (user created, name null), existing user, reuse, expiry and unknown token (all `invalid_token`), and two concurrent redemptions where only one succeeds

## 4. Sessions backend

- [x] 4.1 Implement session create/lookup/touch (hourly throttle, 30-day sliding expiry), the cookie writer (`Secure` from base URL) and `requireUser` middleware. Verify tests for cookie attributes, the expiry after 31 days and that the touch is throttled
- [x] 4.2 Implement the CSRF middleware (JSON content type, Origin check) on mutating `/api` routes. Verify tests for a foreign Origin → 403 `forbidden_origin`, a missing content type → 403, and a same origin passing
- [x] 4.3 Implement `GET /api/me`, `PATCH /api/me` (trim, ≤100, empty → null) and `POST /api/auth/logout`. Verify tests for 200/401 on me, name validation cases, and logout invalidating the cookie
- [x] 4.4 Add the hourly janitor for expired sessions and tokens. Verify a test with an injected clock that the expired rows are deleted

## 5. Frontend session plumbing

- [x] 5.1 Add `@tanstack/react-query`, a `QueryClientProvider`, router context with `queryClient`, the `api.ts` fetch wrapper (`ApiError`, JSON header, 401 → `me=null`) and the `useMe` query. Verify unit tests for the wrapper's error parsing and its 401 handling
- [x] 5.2 Add route protection (`/profile` → `/login` when signed out, `/login` → `/` when signed in) and the pending skeleton at `/`. Verify tests with mocked `/api/me` for each redirect

## 6. Login UI

- [x] 6.1 Build the `/login` email form with the "Check your inbox" state, the 60 s resend countdown and 429 `Retry-After` handling, with `en`/`de` texts. Verify tests for submit, disabled resend and the rate-limited message
- [x] 6.2 Build `/login/verify` (read and strip the fragment, "Complete sign-in" button, success → `/`, error → link to `/login`, missing token → error). Verify tests that no request is sent before the click, plus success and failure

## 7. Signed-in UI

- [x] 7.1 Extend `UserMenu` with the pending and signed-in states (name if set, email, Profile, Sign out). Verify tests for menu order with and without a name, the loading state without actions, the Profile navigation, and that Sign out shows the landing page and "Sign in"
- [x] 7.2 Build the home placeholder at `/` for signed-in users (greeting by name or email). Verify tests for both greetings and the absence of landing content
- [x] 7.3 Build the `/profile` header with email and the editable name form (save, clear, 400 `invalid_name` message, menu updates without reload). Verify tests for set, clear and too long

## 8. Integration and docs

- [x] 8.1 Document the new env vars, the dev mail fallback, reverse-proxy settings and the sign-in flow in the README. Verify the documented local flow works end to end: request a link in the dev server, copy it from the log, confirm, see the home page, set a name, sign out
- [x] 8.2 Run `make check` and `make test` and verify both pass

## 9. Same-browser sign-in without confirmation

- [x] 9.1 Add migration `0003` (`magic_link_tokens.browser_hash`) and set or reuse the `famodash_login_request` cookie on link requests (also for unknown addresses, not on 400/429). Verify tests for the cookie attributes, reuse of an existing cookie value and no cookie on rejected requests
- [x] 9.2 Accept `confirm` in `POST /api/auth/magic-link/verify`: without it, redeem only with the matching login-request cookie, otherwise 403 `confirmation_required` leaving the token unused; clear the cookie on success. Verify tests for the same browser, a missing and a foreign cookie (token still redeemable afterwards) and confirmed redemption without the cookie
- [x] 9.3 Make `/login/verify` try the unconfirmed redemption on load and fall back to the "Complete sign-in" button on `confirmation_required`. Verify tests for direct sign-in, the fallback and confirmed sign-in, and that no request is sent without a token
- [x] 9.4 Update the README sign-in section, rerun the end-to-end flow in a browser (same browser signs in directly, a fresh browser context sees the button) and run `make check` and `make test`

## Workflow follow-up

- Archive with `/opsx:archive` before applying `passkeys`, because `passkeys` adds requirements to `auth-session`, `user-profile` and the login page.
