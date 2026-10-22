# Proposal

## Why

A sign-in link opens in the default browser. When someone signs in from the installed app (iOS keeps a separate cookie store for it), from another browser or on a device without mail access, the link signs in the wrong place or not at all. A short one-time code in the same email can be typed into the login page that requested it, wherever that is.

## What Changes

- Every sign-in email contains, next to the existing "Sign in" button and link, a 6-character one-time code, introduced with "or use this one-time code".
- After requesting a sign-in email, the login page shows a field for that code next to the "Check your inbox" message. Entering the correct code signs in on the spot.
- A code belongs to its sign-in link: using either one uses up both. It is valid for the same 15 minutes.
- A code only works in the browser that requested it (the existing login-request cookie) and for the email address it was sent to. After 5 wrong attempts, that browser's codes for the address stop working, while the links in the emails keep working.
- New endpoint `POST /api/auth/magic-link/code`.
- The development mail fallback also logs the code.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `auth-magic-link`: sign-in email content, code properties and redemption, attempt limit, login page code entry, development log output.

## Impact

- Backend: migration adding `code_hash` and `code_attempts` to `magic_link_tokens`, code generation in `RequestLink`, a new `RedeemCode` in `internal/auth`, a new handler in `internal/httpapi`, email templates (en, de, text and HTML), dev mail fallback.
- Frontend: code field and submission on the login page's "Check your inbox" view, new en/de strings.
- API: new endpoint. Existing endpoints and links are unchanged. Links issued before the migration have no code.
- Works with the `redirect` parameter from `pwa-share-and-shortcuts`, if that is in place: a code sign-in on the login page is a sign-in "on the login page itself".
