# Proposal

## Why

Anyone who can reach a Boardly instance can create an account by requesting a sign-in link. For private
or family instances, the admin wants to close sign-ups once the intended users have accounts, without
locking those users out.

## What Changes

- New environment variable `BOARDLY_ALLOW_REGISTRATION` (`true` | `false`, default `true`). With the
  default, behaviour is unchanged.
- With `BOARDLY_ALLOW_REGISTRATION=false`:
  - A sign-in link request for an address **with** an account works as before: the user gets the
    sign-in email with link and one-time code.
  - A sign-in link request for an address **without** an account sends a localized email (en/de)
    saying that registration on this instance is closed. It contains no link and no code. The HTTP
    response stays the same 202 as for a known address, so the response still doesn't reveal whether
    an account exists. Rate limits apply the same way.
  - Redeeming a link or one-time code never creates an account. A link or code for an address that has
    no account (for example one issued before registration was closed) is rejected as invalid.
- Passkey sign-in is unaffected (it only works for existing accounts anyway).
- `.env.example` and the README document the new variable.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `auth-magic-link`: link requests for unknown addresses send a "registration closed" email instead of
  a sign-in link when registration is disabled; redeeming links and codes creates accounts only when
  registration is allowed; the purpose statement no longer says that anyone can sign up.
- `app-hosting`: new optional configuration variable `BOARDLY_ALLOW_REGISTRATION` with default `true`
  and validation of its value.

## Impact

- Backend: `internal/config` (new setting), `internal/auth` (`RequestLink`, `Redeem`, `RedeemCode`),
  `internal/users` (look up a user by email without creating it), `internal/mail` (new
  `registration_closed` templates in en and de, render function, golden files), wiring in `main`.
- No API shape changes, no database migration, no frontend changes.
- Docs: `.env.example`, README (configuration and authentication sections).
