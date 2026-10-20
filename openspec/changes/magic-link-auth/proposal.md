# Proposal

## Why

Famodash is a per-user app (dashboards, favourites and API tokens all belong to one user), but `project-foundation` has no users at all. Passwordless sign-in by magic link is the base login method. It also creates accounts (anyone can sign up) and acts as the fallback and step-up path for passkeys in the follow-up `passkeys` change.

## What Changes

- Add accounts identified by email address. Anyone can sign up: the first successful magic-link sign-in for an address creates the account.
- Add the magic-link flow. On `/login`, the visitor enters their email and gets a one-time, short-lived link in their language (English or German). Opening it and confirming signs them in. The link is safe against mail scanners that pre-open links.
- Add rate limiting for magic-link requests, per email address and per client IP.
- Add server-side sessions with an HttpOnly cookie, a current-user endpoint, logout, and protection against cross-site requests.
- Add SMTP mail delivery, configured through environment variables. Without SMTP settings, the link is written to the server log for local development.
- Show the signed-in state in the user menu: name (if set) and email, a "Profile" link and a "Sign out" action.
- When signed in, `/` shows a home placeholder instead of the landing page.
- Add `/profile` with the account's email and an optional, editable display name.

## Capabilities

### New Capabilities

- `auth-magic-link`: requesting, delivering and redeeming one-time email sign-in links, including account creation and rate limits.
- `auth-session`: sessions after sign-in: the cookie, lifetime, current-user lookup, logout, cross-site request protection and route protection in the web app.
- `user-profile`: the `/profile` page and the user's editable profile data (display name).
- `home-page`: what a signed-in user sees at `/` (placeholder for now).

### Modified Capabilities

- `app-shell`: adds the signed-in user menu content (name/email, Profile, Sign out) and the menu state while the session is loading. Existing requirements stay unchanged.

## Impact

- **Backend**: new tables `users`, `sessions` and `magic_link_tokens` (migration `0002`). New endpoints under `/api/auth/*` and `/api/me`. New config variables `FAMODASH_BASE_URL`, `FAMODASH_SMTP_*` and `FAMODASH_TRUST_PROXY_HEADERS`. New dependencies are an SMTP client library and `golang.org/x/time/rate`.
- **Frontend**: new routes `/login` (replaces the placeholder), `/login/verify` and `/profile`. `/` switches between landing and home. New dependency `@tanstack/react-query` for server state.
- **Depends on**: `project-foundation` (archived first).
