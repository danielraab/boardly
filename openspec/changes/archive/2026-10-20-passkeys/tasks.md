# Tasks

## 1. Data and configuration

- [x] 1.1 Add migration `0004_passkeys.sql` (`passkeys`, `webauthn_challenges`, `sessions.passkey_enrollment_until`). Verify that the migration test applies on a DB with `0002` data and that deleting a user cascades to their passkeys
- [x] 1.2 Extend config with `FAMODASH_WEBAUTHN_RP_ID` (default: base URL host) and `FAMODASH_AAGUID_LIST_URL` (default upstream URL, empty disables), and build the `webauthn.WebAuthn` instance. Verify table tests for the default RP ID and the override

## 2. Provider list

- [x] 2.1 Implement the AAGUID directory: load the cached file, background refresh every 24 h with ETag, timeout, 20 MB cap, shape validation, atomic file replace and map swap. Verify tests against an `httptest` server for a fresh download, a 304, a malformed response (cache kept) and an unreachable source with no cache (empty map, no error)
- [x] 2.2 Implement the lookup (normalization, all-zero → none) and verify unit tests with a fixture containing Google Password Manager

## 3. Passkey backend

- [x] 3.1 Implement the challenge store (cookie, DB row, single use, 5 min expiry, janitor cleanup). Verify tests for reuse, expiry and wrong kind
- [x] 3.2 Set `passkey_enrollment_until` on magic-link redemption, expose `passkeyEnrollmentUntil` in `/api/me`, and record `auth_method`. Verify tests for after magic link (+5 min) and after passkey (null)
- [x] 3.3 Implement registration options/verify (window check, `excludeCredentials`, resident key required, challenge bound to session, label trim ≤64). Verify tests with a software authenticator (go-webauthn test helpers or a virtual authenticator fixture) for success, window closed → 403, window closing during the ceremony → saved, label 65 → 400 `invalid_label`
- [x] 3.4 Implement discoverable login options/verify (user handle → user, `passkey_unknown` vs `passkey_invalid`, update `sign_count`/`last_used_at`, passkey session without window). Verify tests for success, a deleted credential → `passkey_unknown` and a bad signature → `passkey_invalid`
- [x] 3.5 Implement `GET /api/passkeys` (own only, newest first, provider inline) and `DELETE /api/passkeys/{id}` (404 for foreign or missing, session stays valid). Verify tests for isolation between two users, deleting outside the window, and foreign delete → 404

## 4. Login page

- [x] 4.1 Add `@simplewebauthn/browser`, the WebAuthn support detection and the API calls. Verify that `pnpm build` succeeds
- [x] 4.2 Add conditional-UI autofill (`autocomplete="username webauthn"`) and the "Sign in with passkey" button to `/login`, with error mapping (cancel → silent, `passkey_unknown` → message + `signalUnknownCredential`, `passkey_invalid` → message), hidden without WebAuthn. Verify tests with a mocked simplewebauthn for each case and for the hidden button
- [x] 4.3 Manually verify autofill and button sign-in in Chrome (Chrome DevTools virtual authenticator) and in one real platform authenticator

## 5. Profile Passkeys tab

- [x] 5.1 Add Base UI `Tabs` under the profile header with `?tab=passkeys` search sync. Verify tests that the tab is selected by default and after reload
- [x] 5.2 Build the passkey list (provider icon via `<img>` with light/dark choice and `KeyRound` fallback, name fallback chain, localized dates, empty state). Verify tests for known/unknown provider and the empty state
- [x] 5.3 Build the enrollment UI: live countdown, "Add passkey" dialog with optional label, closed-window hint with "Send sign-in link". Verify tests with fake timers that the countdown hits zero and disables the action without reload, plus a test for the send-link call
- [x] 5.4 Build the delete confirmation dialog. Verify tests that confirm removes the entry and cancel keeps it

## 5b. Settings page

- [x] 5.5 Rename `/profile` to `/settings` ("Settings" heading, user menu link "Settings"), move the email and display name section into a "Profile" tab before "Passkeys", and make "Profile" the default tab (`?tab=profile|passkeys`). Verify tests for the menu link, the default tab, the name form inside the Profile tab and `?tab=passkeys` after reload

## 6. Integration and docs

- [x] 6.1 Document passkeys in the README (RP ID binding to the base URL host, the 5-minute window, outbound access for the AAGUID list and self-hosting the list). Verify the documented flow end to end in dev: magic-link sign-in, add a passkey, sign out, sign in with the passkey, delete it
- [x] 6.2 Run `make check` and `make test` and verify both pass

## Workflow follow-up

- Archive with `/opsx:archive` after review.
