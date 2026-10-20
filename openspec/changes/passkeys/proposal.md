# Proposal

## Why

Magic links work, but every sign-in costs a round trip through the mailbox. Passkeys give a one-tap, phishing-resistant sign-in, which the README promises as the second login method. Adding a passkey is a sensitive action: anyone holding an open session could otherwise add their own. So it's only allowed shortly after the user proved control of their email address.

## What Changes

- Add passkey sign-in on `/login`. It works through the browser's autofill on the email field (conditional UI) and through an explicit "Sign in with passkey" button. No email address needs to be typed.
- Add passkey registration. It's allowed only within 5 minutes after a sign-in by magic link. A passkey sign-in never opens that window.
- Each passkey can get an optional label when it's added.
- Detect the passkey's provider (e.g. "iCloud Keychain", "Google Password Manager", "1Password") from its AAGUID, using the community-maintained [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids) list, and show its name and icon.
- Users can delete a passkey at any time, without the 5-minute window.
- `/profile` gets tabs below the profile header. The only tab for now is "Passkeys": a list of passkeys, the remaining time for adding one, and a way to reopen the window by sending a new sign-in link.

## Capabilities

### New Capabilities

- `auth-passkey`: WebAuthn passkey sign-in, registration with the time-limited enrollment window, listing with provider detection, and deletion.

### Modified Capabilities

- `user-profile`: adds the tabbed area on `/profile` with the "Passkeys" tab. The existing header and name requirements stay unchanged.
- `auth-session`: adds that a session records how it was created and that sessions created by passkey can't add passkeys. The existing requirements stay unchanged.

## Impact

- **Backend**: migration `0003` (`passkeys` and `webauthn_challenges` tables, `sessions.passkey_enrollment_until`). New endpoints under `/api/auth/passkey/*` and `/api/passkeys`. `/api/me` gains a `passkeyEnrollmentUntil` field. New config `FAMODASH_WEBAUTHN_RP_ID` and `FAMODASH_AAGUID_LIST_URL`. New dependency `github.com/go-webauthn/webauthn`. The server downloads the AAGUID list (about 7 MB, no license declared upstream, so it's not redistributed in the repo or binary) and caches it next to the database. That needs outbound HTTPS. Without it, the provider stays unknown.
- **Frontend**: new dependency `@simplewebauthn/browser`. `/login` and `/profile` change. New dialogs for adding and deleting a passkey.
- **Depends on**: `magic-link-auth` (archived first).
