# Tasks

## 1. Storage and generation

- [x] 1.1 Add migration `0010_magic_link_code.sql` (`code_hash BLOB`, `code_attempts INTEGER NOT NULL DEFAULT 0`) and verify the migration test passes on a fresh and an existing database
- [x] 1.2 Add code generation (31-symbol alphabet, `crypto/rand`, no modulo bias), normalization (upper case, strip spaces and hyphens) and display formatting (`ABC 123`), with unit tests
- [x] 1.3 Generate and store the code hash in `RequestLink` and pass the code to the mail data; verify with a test that the stored row contains no plain code

## 2. Redemption

- [x] 2.1 Add `Service.RedeemCode(ctx, email, code, userAgent, browserSecret)` with the single-transaction lookup, constant-time match, attempt increment and session creation; cover correct, other browser, wrong address, expired, used-by-link, link-after-code and lock-after-5 cases with Go tests
- [x] 2.2 Add `POST /api/auth/magic-link/code` in `internal/httpapi` (200 with user + session cookie + cleared login-request cookie, 400 `invalid_code`, 400 for a malformed body); verify with httptest cases

## 3. Email

- [x] 3.1 Update the en and de templates (text and HTML): button and link, "or use this one-time code" with the code in monospace, validity of both, ignore note; update the golden files in `mail/testdata` and verify the mail tests pass
- [x] 3.2 Log the code in the development mail fallback and verify with a test that the log contains link and code

## 4. Login page

- [x] 4.1 Add the code form to the "Check your inbox" view (label, `autocomplete="one-time-code"`, sign-in button, error message), signing in like the passkey path; verify with Testing Library tests for success and a wrong code
- [x] 4.2 Add the en and de strings and verify the i18n key check passes

## 5. Verification

- [x] 5.1 Without SMTP, request an email, take the code from the log, enter it on the login page and verify the sign-in; then verify the same email's link no longer works; run `make check` and `make test`
