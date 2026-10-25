# Tasks

## 1. Configuration

- [x] 1.1 Add `AllowRegistration bool` (default `true`) to `internal/config`, read from `BOARDLY_ALLOW_REGISTRATION` (exactly `true`/`false`, empty or unset = default, otherwise an error naming the variable); verify with config tests for unset, `false` and `maybe`
- [x] 1.2 Add `RegistrationClosed bool` to `auth.Service` and set it to `!cfg.AllowRegistration` in `backend/main.go`; verify `go build ./...` succeeds

## 2. Mail

- [x] 2.1 Factor the locale fallback and template execution of `RenderMagicLink` into an internal `render` helper in `internal/mail/templates.go`; verify the existing magic-link golden tests still pass unchanged
- [x] 2.2 Add `templates/{en,de}/registration_closed.{txt,html}` and `RenderRegistrationClosed(locale, to, RegistrationClosedData)` with subjects "Registration for {{.AppName}} is closed" / "Registrierung bei {{.AppName}} ist geschlossen", no link and no code, using the app title throughout; verify with new `registration_closed_{en,de}.golden` tests and a custom app title test

## 3. Auth behaviour

- [x] 3.1 Add a read-only lookup of a user by email to `users.Store` (no creation); verify with a users test for a known and an unknown address
- [x] 3.2 In `RequestLink`, after the rate-limit checks, when `RegistrationClosed` and the address has no account: insert an already-used token row (random hash, no code) so rate limits count it, send the registration-closed email, and return the browser secret as in the normal path; verify with auth tests that an unknown address gets the closed email without link or code, a known address gets the sign-in email, and a second request within 60 s is rate limited
- [x] 3.3 In `Redeem` and `RedeemCode`, when `RegistrationClosed`, use the lookup instead of `EnsureByEmail` and return `ErrInvalidToken` / `ErrInvalidCode` for an address without an account, creating no account and no session; verify with auth tests for a link and a code issued while open and redeemed after closing, and that existing accounts still sign in
- [x] 3.4 Add an HTTP test in `internal/httpapi/auth_test.go` that, with registration closed, requests for a known and an unknown address return identical 202 responses and set the login-request cookie the same way; verify the test passes

## 4. Documentation and checks

- [x] 4.1 Document `BOARDLY_ALLOW_REGISTRATION` in `.env.example` and the README configuration table, and mention closed registration in the README Authentication section; verify by reading the rendered README
- [x] 4.2 Update the Purpose of `openspec/specs/auth-magic-link/spec.md` and the `context` in `openspec/config.yaml` so they no longer state that anyone may sign up unconditionally; verify `openspec validate disable-registration --strict` passes
- [x] 4.3 Run `make check` and `make test`; verify both pass
