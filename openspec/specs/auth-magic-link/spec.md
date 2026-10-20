# auth-magic-link Specification

## Purpose

Lets anyone sign in to Famodash without a password, by receiving a one-time link at their email address. It creates the account on the first successful sign-in.

## Requirements

### Requirement: Request a sign-in link
`POST /api/auth/magic-link` with `{"email": "...", "locale": "en"|"de"}` SHALL send a sign-in link to that address and respond with status 202. The email is trimmed and lowercased first. An unsupported or missing locale falls back to `en`.

#### Scenario: Valid request
- **WHEN** a client posts `{"email": " Ann@Example.org ", "locale": "de"}`
- **THEN** the server responds 202 and sends a German sign-in email to `ann@example.org`

#### Scenario: Invalid email
- **WHEN** a client posts `{"email": "not-an-email"}`
- **THEN** the server responds 400 with `error.code` `invalid_email` and sends no email

### Requirement: Response does not reveal account existence
The response to a valid sign-in link request SHALL be identical whether or not an account exists for the address.

#### Scenario: Unknown and known address
- **WHEN** a client requests a link once for an address with an account and once for an address without one
- **THEN** both responses have status 202 and identical bodies

### Requirement: Sign-in link properties
Each link SHALL carry a random, unguessable token of at least 256 bits. It SHALL expire 15 minutes after it was issued and SHALL be redeemable only once. The server SHALL store only a hash of the token. The token travels in the URL fragment of `<base URL>/login/verify`.

#### Scenario: Link format
- **WHEN** a sign-in email is sent
- **THEN** it contains a link of the form `<base URL>/login/verify#token=<token>`

#### Scenario: Token not stored in plain text
- **WHEN** the token store is inspected after a link was issued
- **THEN** it contains no copy of the plain token

### Requirement: Localized sign-in email
The sign-in email SHALL be written in the requested locale. It SHALL contain the link, its 15-minute validity, and a note that the recipient can ignore the email if they didn't request it. It SHALL be sent as plain text with an HTML alternative.

#### Scenario: English email
- **WHEN** a link is requested with locale `en`
- **THEN** the email subject and body are English and mention the 15-minute validity

### Requirement: Redeem a sign-in link
`POST /api/auth/magic-link/verify` with `{"token": "...", "confirm": true|false}` SHALL, for a valid unused and unexpired token, mark the token as used, create the account for its email if none exists, start a session, and respond 200 with the user. Without `"confirm": true`, it SHALL redeem the token only when the request carries the login-request cookie of the browser that requested the link. Otherwise it responds 403 with `error.code` `confirmation_required` and leaves the token unused.

#### Scenario: First sign-in creates account
- **WHEN** a valid token is redeemed for an address with no account
- **THEN** an account is created for that address, a session cookie is set, and the response body is the new user with `name` `null`

#### Scenario: Existing account
- **WHEN** a valid token is redeemed for an address with an account
- **THEN** no new account is created and a session for the existing account is started

#### Scenario: Unconfirmed redemption from another browser
- **WHEN** a valid token is posted without `"confirm": true` and without the requesting browser's login-request cookie
- **THEN** the server responds 403 with `confirmation_required`, sets no session cookie, and the token can still be redeemed

### Requirement: Invalid links are rejected
Redeeming a token that is unknown, expired or already used SHALL respond 400 with `error.code` `invalid_token`, without starting a session. The response SHALL NOT reveal which of the three cases applies.

#### Scenario: Reused link
- **WHEN** a token that was already redeemed is redeemed again
- **THEN** the server responds 400 with `invalid_token` and sets no session cookie

#### Scenario: Expired link
- **WHEN** a token is redeemed 16 minutes after it was issued
- **THEN** the server responds 400 with `invalid_token`

### Requirement: Links sign in directly only in the requesting browser
Requesting a sign-in link SHALL set a short-lived HttpOnly login-request cookie in the requesting browser. When the link is opened in that browser, the verify page SHALL sign in right away, without further interaction. Anywhere else (another browser or device, or a mail scanner that pre-opens links and runs their scripts) the token SHALL NOT be used up until the visitor explicitly confirms with "Complete sign-in".

#### Scenario: Same browser
- **WHEN** a visitor requests a link and opens it in the same browser within its validity
- **THEN** they are signed in and taken to `/` without having to confirm

#### Scenario: Other browser or device
- **WHEN** the link is opened in a browser without the login-request cookie
- **THEN** the page shows a "Complete sign-in" button and the token stays unused until it is activated

#### Scenario: Mail scanner
- **WHEN** a client without the login-request cookie opens the link and runs the page's scripts
- **THEN** no session is started and the visitor can still use the link afterwards

#### Scenario: Confirmation
- **WHEN** the visitor activates "Complete sign-in" and redemption succeeds
- **THEN** the visitor is signed in and taken to `/`

#### Scenario: Failed confirmation
- **WHEN** redemption fails with `invalid_token`
- **THEN** the page says the link is invalid or expired and offers a way back to `/login`

### Requirement: Rate limits on link requests
The server SHALL limit sign-in link requests to one per 60 seconds and five per hour for each email address, and to 20 per hour for each client IP. Rejected requests get status 429 with `error.code` `rate_limited` and a `Retry-After` header.

#### Scenario: Immediate repeat
- **WHEN** a link is requested for the same address twice within 60 seconds
- **THEN** the second request gets 429 `rate_limited` with a `Retry-After` header and no email is sent

### Requirement: Login page email form
`/login` SHALL offer an email field and a submit button. After submission, it shows a translated confirmation with the address and a resend action that is disabled for 60 seconds. On 429 it shows how long to wait.

#### Scenario: Submit
- **WHEN** a visitor enters their email on `/login` and submits
- **THEN** the page shows "Check your inbox" with that address, and the resend action is disabled for 60 seconds

#### Scenario: Already signed in
- **WHEN** a signed-in user opens `/login`
- **THEN** they are redirected to `/`

### Requirement: Development mail fallback
When no SMTP host is configured, the server SHALL write each sign-in link to its log instead of sending email, and SHALL log a warning at startup that mail delivery is disabled.

#### Scenario: No SMTP configured
- **WHEN** the server runs without `FAMODASH_SMTP_HOST` and a link is requested
- **THEN** the full link appears in the server log and the request still responds 202
