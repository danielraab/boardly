# Design

## Context

This builds on `magic-link-auth`: `users` (UUID ids), `sessions` (hashed id, `auth_method`), the `requireUser` middleware, CSRF middleware (JSON + Origin), the `apiError` format, TanStack Query `['me']`, the `/login` email form (`autocomplete="username"`) and the `/profile` page with email and display name. The base URL is `FAMODASH_BASE_URL`, which is `http://localhost:3000` in dev, where Vite proxies `/api`. Requirements are in `auth-passkey`, `auth-session` and `user-profile` of this change.

## Goals / Non-Goals

**Goals:**
- Usernameless passkey sign-in with autofill, built on maintained libraries on both sides.
- An enrollment window that the server enforces and the UI only displays.
- Provider names and icons without redistributing an unlicensed 7 MB file.

**Non-Goals:**
- Renaming a passkey after it was created (can come later).
- Attestation verification or authenticator allow-lists.
- Passkeys as a second factor, account recovery beyond the magic link.
- WebAuthn "related origins" or several RP IDs.

## Decisions

### Libraries
- Backend: `github.com/go-webauthn/webauthn` (v0.18.x). It's the de-facto Go RP library and supports discoverable login (`BeginDiscoverableLogin` / `FinishDiscoverableLogin`).
- Frontend: `@simplewebauthn/browser` (v14). `startRegistration({optionsJSON})` / `startAuthentication({optionsJSON, useBrowserAutofill})` handle base64url encoding, conditional mediation and aborting a pending autofill request when the button is used. The JSON shapes from go-webauthn are the standard `PublicKeyCredential*OptionsJSON`, so the two fit together without a translation layer.

### RP configuration
`RPID = FAMODASH_WEBAUTHN_RP_ID` or the host name of `FAMODASH_BASE_URL`. `RPOrigins = [origin of FAMODASH_BASE_URL]`. `RPDisplayName = "Famodash"`. In dev, `localhost` is a valid RP ID and browsers treat `http://localhost` as a secure context.

### Data model (migration `0004_passkeys.sql`)
```
passkeys                                  webauthn_challenges
────────                                  ───────────────────
id              TEXT PK (uuid)            id_hash     BLOB PK (sha256 of cookie value)
user_id         → users ON DELETE CASCADE kind        TEXT ('login' | 'register')
credential_id   BLOB UNIQUE               user_id     TEXT NULL
public_key      BLOB                      session_id  BLOB NULL (for 'register')
aaguid          BLOB (16)                 data        TEXT (go-webauthn SessionData JSON)
sign_count      INTEGER                   expires_at  TEXT (now + 5 min)
transports      TEXT (JSON array)
backup_eligible INTEGER, backup_state INTEGER
label           TEXT NULL
created_at, last_used_at TEXT NULL

sessions + passkey_enrollment_until TEXT NULL
```
The label is sent with `registration/verify`, so challenges don't store it. The migration is `0004` because `0003_login_request.sql` already exists.

The WebAuthn user handle is the 16 bytes of the user's UUID: opaque, stable and free of PII. Credentials are stored as the fields go-webauthn needs to rebuild `webauthn.Credential`, not as an opaque blob. That way `last_used_at` and `sign_count` can be updated, and the data stays inspectable.

### Ceremony state: DB-backed challenge with an HttpOnly cookie
`.../options` stores `SessionData` in `webauthn_challenges` and sets `famodash_webauthn=<random>` (HttpOnly, SameSite=Strict, Max-Age 300, Path `/api`). `.../verify` loads the challenge by cookie hash and kind, deletes it (single use), then verifies. Storing it in the DB rather than memory means a restart mid-ceremony doesn't break it and keeps the server stateless. The janitor from `magic-link-auth` also cleans up expired challenges.

### Endpoints
| Method & path | Auth | Notes |
|---|---|---|
| `POST /api/auth/passkey/login/options` | – | discoverable, `userVerification: preferred` |
| `POST /api/auth/passkey/login/verify` | – | body = credential JSON → 200 user + session (`auth_method='passkey'`, no window) |
| `GET /api/passkeys` | user | list with provider |
| `POST /api/passkeys/registration/options` | user + window | `residentKey: required`, `excludeCredentials` = user's passkeys, `attestation: none` |
| `POST /api/passkeys/registration/verify` | user | body `{credential, label}`; the challenge must belong to this session |
| `DELETE /api/passkeys/{id}` | user | 404 when the passkey isn't the user's own |

Login errors: `passkey_unknown` when the credential ID isn't found, and `passkey_invalid` for every other verification failure. On `passkey_unknown`, the frontend calls `PublicKeyCredential.signalUnknownCredential({rpId, credentialId})` when it's available, so the password manager can drop the stale passkey. go-webauthn's clone warning (sign count went backwards) is logged and not enforced, because synced passkeys report 0.

### Enrollment window
On magic-link redemption, `sessions.passkey_enrollment_until = now + 5 min`. Passkey sessions leave it `NULL`. `registration/options` checks `now < passkey_enrollment_until` (else 403 `enrollment_window_closed`) and binds the challenge to the session ID. `registration/verify` doesn't recheck the window. It requires a valid, unexpired challenge bound to the current session, which can only exist if the window was open at the start (spec scenario "Window closes during the prompt"). `/api/me` exposes `passkeyEnrollmentUntil`. The UI countdown is computed from it against the client clock. A clock offset only affects the display, because the server decides.

"Send sign-in link" on the Passkeys tab reuses `POST /api/auth/magic-link` with the user's own email and the current locale. Opening that link creates a new session with a fresh window. The old session stays valid.

### AAGUID provider list: runtime download + on-disk cache
- Source: `FAMODASH_AAGUID_LIST_URL`, default `https://raw.githubusercontent.com/passkeydeveloper/passkey-authenticator-aaguids/main/combined_aaguid.json`. Setting it to empty disables the feature.
- On startup, load `<dir of FAMODASH_DB_PATH>/aaguids.json` if present, then refresh in the background when it's older than 24 h and every 24 h after that. The refresh uses a conditional GET (`ETag` saved in a `aaguids.json.etag` sidecar), a 30 s timeout and a 20 MB size cap. It validates the JSON shape before atomically replacing the file and swapping the in-memory map (`sync/atomic.Pointer`).
- The lookup normalizes the AAGUID to lowercase hyphenated form. An all-zero AAGUID → no provider.
- Icons are `data:` URIs (mostly SVG). The list response inlines `iconLight`/`iconDark` per passkey. That's a few KB each and users have few passkeys. The frontend renders them as `<img src>`, never as inline HTML, so an SVG can't run scripts.
- *Alternative*: embed via `go generate`. The upstream repo has no license, so that would redistribute it, and it adds 7 MB to the repo and binary. Rejected.
- *Alternative*: the FIDO MDS3 blob. It's signed and licensed, but covers hardware keys more than the passkey providers people actually use (password managers). Rejected for now.

### Frontend
- `/login`: on mount, if `PublicKeyCredential.isConditionalMediationAvailable()` resolves true, start `startAuthentication({ optionsJSON, useBrowserAutofill: true })` and set the email input to `autocomplete="username webauthn"`. A "Sign in with passkey" button (lucide `KeyRound`) runs the modal flow. simplewebauthn aborts the pending conditional request. `NotAllowedError` (cancelled) → no message. Success → `setQueryData(['me'])`, then navigate to `/`. Without WebAuthn the button is hidden.
- `/settings` (formerly `/profile`, no redirect since the app isn't released): uses the full width of the app container like the start page (only the name input row keeps `max-w-xl`). The "Settings" heading, then Base UI `Tabs` "Profile" (email and display name form, unchanged) and "Passkeys". The tab value is synced with `validateSearch` (`tab: 'profile' | 'passkeys'`, `profile` default; unknown values are set to `undefined` because the raw search params are merged under the validated ones).
- Passkeys tab: list items with a provider icon (`<img>` with the variant for the app's theme; the app is light-only for now, so `iconLight`, then `iconDark`, then a generic `KeyRound` fallback), name (label → provider → "Passkey"), and dates formatted with `Intl.DateTimeFormat(i18n.language)`. An "Add passkey" `Dialog` holds the label input (max 64) and then the browser prompt. The delete `AlertDialog` confirms deletion. The countdown is a `useNow(1000)` hook that only ticks while the window is open. On zero it refetches `['me']`.
- Mutations invalidate `['passkeys']`.

## Risks / Trade-offs

- [Some browser/authenticator combinations (notably Firefox with `attestation: none`) report an all-zero AAGUID] → They show as "Passkey" with a generic icon. Requesting attestation would trigger extra consent prompts, which isn't worth it for a label.
- [The upstream list changes format or disappears] → Shape validation keeps the last good cached copy. Provider detection is cosmetic, and an unknown provider is a supported state.
- [The server has no outbound internet (air-gapped self-hosting)] → The feature degrades to unknown providers. Admins can host the file themselves and point `FAMODASH_AAGUID_LIST_URL` at it.
- [Changing `FAMODASH_BASE_URL` host later breaks all passkeys, because they're bound to the RP ID] → Document it in the README, and note that magic-link sign-in still works to register new ones.
- [A stolen session cookie can't add a persistent passkey, but it can delete passkeys] → An accepted trade-off from the requirements: deleting only removes convenience, and email sign-in remains.
- [A user signs in via a magic link on a device other than the one they want to add a passkey on] → That's the intended flow: the window belongs to the session where the link was opened.

## Migration Plan

Migration `0004_passkeys.sql` adds tables and a nullable column. Existing sessions get `passkey_enrollment_until = NULL`, so they can't add passkeys until the next magic-link sign-in. Rollback means restoring the DB file.
