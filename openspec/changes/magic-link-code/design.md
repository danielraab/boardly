# Design

## Context

`magic_link_tokens` rows hold a SHA-256 hash of a 256-bit link token, the email, an expiry 15 minutes after issue, `used_at`, and `browser_hash`: the hash of the HttpOnly `famodash_login_request` cookie (path `/api/auth/magic-link`) of the requesting browser. Requests are limited to 1/min and 5/h per address and 20/h per IP. `Redeem` uses a single `UPDATE … RETURNING` to use up a token atomically.

## Goals / Non-Goals

**Goals:**
- Sign in where the request was made, without switching apps or browsers.
- Brute forcing a code stays out of reach.

**Non-Goals:**
- Codes that work in another browser or device. The link with "Complete sign-in" already covers that.
- Code-only emails or a code-only sign-in mode.
- WebOTP / SMS autofill.

## Decisions

### Code alphabet and format
6 characters from `ABCDEFGHJKMNPQRSTUVWXYZ23456789` (31 symbols, without 0/O, 1/I/L), drawn with `crypto/rand` without modulo bias. That is about 8.9 × 10⁸ codes. The email shows it as `ABC 123` (a space after the third character) in a monospace font. Input is case-insensitive and ignores spaces and hyphens. The field uses `autocomplete="one-time-code"`, `autocapitalize="characters"` and `maxlength` large enough for the separators.
*Alternative*: 6 digits (10⁶): rejected, as it gives a thousand times fewer combinations for the same length.

### Storage: same row as the link
New nullable columns `code_hash BLOB` and `code_attempts INTEGER NOT NULL DEFAULT 0`. `code_hash` = SHA-256 of `token_hash || normalized code`, which ties the code to its row. Because the row is shared, "using either uses both" follows from the existing `used_at`. A plain hash of a low-entropy code can be brute-forced offline, but that needs database access and the code lives at most 15 minutes, so this is accepted.

### Redemption
`POST /api/auth/magic-link/code` `{"email","code"}`. The login-request cookie is sent to it automatically because of the cookie path. In one transaction:
1. Select the unused, unexpired rows for `email` with `browser_hash = hash(cookie)`, `code_hash IS NOT NULL` and `code_attempts < 5`.
2. If one row's code matches (constant-time compare), mark it used, `EnsureByEmail`, create a session with method `magic_link`, clear the login-request cookie (as verify does) and return 200 with the user.
3. Otherwise increment `code_attempts` on all selected rows and return 400 `invalid_code`.
A missing cookie, an unknown address, an expired or used code, and a locked code all give the same 400 `invalid_code`.

### Attempt limit
5 wrong attempts per row. Every wrong attempt counts against all live codes of that browser and address, so asking for more emails doesn't give more guesses per email. Together with the request limits (5 emails/hour per address), that is at most 25 guesses per hour. The link isn't affected by the lock, since it carries 256 bits.
*Alternative*: a separate per-IP limiter for code attempts: not needed given the per-row limit and the cookie binding. It can be added later without changing the API.

### Login page
The "Check your inbox" view gains a form: label "One-time code", input, "Sign in" button. The resend and "use another address" actions stay. On success it behaves like a passkey sign-in on this page (sets the current user, navigates to `/`, or to `redirect` once that exists). On `invalid_code` it shows "This code is wrong or has expired. Check the latest email or request a new one."

### Email text
Below the button: "Or enter this one-time code on the sign-in page where you requested it:" followed by the code in large monospace. The validity line covers both: "The link and the code are valid for 15 minutes and can be used only once." The plain-text alternative has the same content.

## Risks / Trade-offs

- [Users type the code into another browser and are confused] → The email says "on the sign-in page where you requested it", and the error message points to the latest email or a new request.
- [Mail scanners don't affect codes] → Correct: codes require a POST with the cookie, so scanners can't use them up.
- [Rows issued before the migration have no code] → They expire within 15 minutes of deployment. The lookup ignores rows without `code_hash`.
