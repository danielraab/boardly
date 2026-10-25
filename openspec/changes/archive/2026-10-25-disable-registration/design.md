# Design

## Context

Accounts are created implicitly: `auth.Service.Redeem` and `RedeemCode` call `users.Store.EnsureByEmail`
when a link or code is redeemed. `RequestLink` deliberately does the same work for known and unknown
addresses, so the 202 response reveals nothing. Per-email rate limits are computed from the rows in
`magic_link_tokens` (`emailRetryAfter`), and the per-IP limit from `Service.IPLimiter`. Configuration is
read in `internal/config` and copied onto the auth service in `main.go` (as done for `AppTitle`).

## Goals / Non-Goals

**Goals:**
- One switch that stops new accounts from being created, while existing users sign in as before.
- Unknown addresses learn why they can't sign in, by email only, so the HTTP API keeps not revealing
  which addresses have accounts.

**Non-Goals:**
- Invitations, allow-lists of addresses or domains, or an admin UI to manage registration.
- Showing "registration closed" on the login page (it would make no difference to known users and would
  need a public config endpoint).
- Changing passkey sign-in; it only works for existing accounts already.

## Decisions

### Config: `BOARDLY_ALLOW_REGISTRATION`, boolean, default `true`

Parsed like `BOARDLY_TRUST_PROXY_HEADERS`: exactly `true` or `false`, empty or unset means the
default. Stored as `Config.AllowRegistration` (default `true`).

On the auth service the field is `RegistrationClosed bool`, and `main.go` sets it to
`!cfg.AllowRegistration`. The negated name makes the zero value mean "open", so existing tests and any
`Service` built without the field keep today's behaviour.

Alternative: `BOARDLY_DISABLE_REGISTRATION=false`. Rejected: double negatives in env files are
confusing, and the user-facing variable should read as "allow".

### Decide in `RequestLink`, after the rate limits

`RequestLink` keeps checking the per-email and per-IP limits first. Then, if registration is closed, it
looks up the address with a new read-only `users.Store.ExistsByEmail` (or `GetByEmail`). For an unknown
address it:

1. Inserts a row into `magic_link_tokens` with a random token hash, `code_hash` NULL and `used_at`
   set to now. Nobody ever receives that token, and the row makes `emailRetryAfter` count this request,
   so rate limits stay identical without a second limiter or a schema change.
2. Renders and sends the "registration closed" email.
3. Returns the browser secret exactly as in the normal path, so the cookie behaviour doesn't differ.

Alternative: a separate table or in-memory limiter for refused requests. Rejected: more code, and an
in-memory limiter would reset on restart and diverge from the per-email limit of real requests.

Alternative: send nothing to unknown addresses. Rejected by the requirement; a silent failure also
leaves legitimate newcomers guessing.

### Guard account creation in `Redeem` and `RedeemCode`

When registration is closed, both use a lookup instead of `EnsureByEmail`. If no account exists, the
transaction is rolled back and they return `ErrInvalidToken` / `ErrInvalidCode`. This covers links and
codes issued before a restart with the switch flipped. Rolling back leaves the token unused, which is
harmless: it can only ever fail the same way, and it expires after 15 minutes.

`EnsureByEmail` stays the single creation path; a small helper in `auth` picks between it and the
lookup based on `RegistrationClosed`.

### Mail: new `registration_closed` templates

`internal/mail` gets `templates/{en,de}/registration_closed.{txt,html}` following the
`magic_link` layout (txt defines `subject` and `body`), plus `RenderRegistrationClosed(locale, to,
RegistrationClosedData{AppName})`. The shared locale fallback and template execution in
`RenderMagicLink` are factored into an internal `render(name, locale, to, data)` helper so both
functions stay short. Golden files `registration_closed_{en,de}.golden` pin the output, like the
existing magic-link goldens.

Subjects: "Registration for {{.AppName}} is closed" / "Registrierung bei {{.AppName}} ist geschlossen".

## Risks / Trade-offs

- [Timing: the closed path does a user lookup and skips code generation, so response time could differ
  slightly from the normal path] → Both paths do a DB insert and a mail render; sending is the dominant
  cost in both and is synchronous in both. Not worth extra padding for this threat model.
- [The closed-registration email is sent to any address someone types in, like sign-in emails are
  today] → Same per-email and per-IP rate limits apply; content contains no link or code.
- [Flipping the switch to `false` doesn't affect existing sessions or accounts] → Intended; closing
  registration is not account removal.

## Migration Plan

No migration. The default keeps current behaviour. Admins opt in by setting
`BOARDLY_ALLOW_REGISTRATION=false` and restarting; rollback is removing the variable.
