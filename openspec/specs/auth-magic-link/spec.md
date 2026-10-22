# auth-magic-link Specification

## Purpose

Lets anyone sign in to Boardly without a password, by receiving a one-time link at their email address. It creates the account on the first successful sign-in.

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
The sign-in email SHALL be written in the requested locale. It SHALL contain the sign-in button and link, then, introduced as an alternative ("or use this one-time code"), the one-time code with a note that it is entered on the sign-in page where the email was requested. It SHALL also contain the 15-minute validity of link and code and a note that the recipient can ignore the email if they didn't request it. It SHALL be sent as plain text with an HTML alternative, both with the same content.

#### Scenario: English email
- **WHEN** a link is requested with locale `en`
- **THEN** the email subject and body are English and mention the 15-minute validity

#### Scenario: Both ways to sign in
- **WHEN** a sign-in email is sent
- **THEN** its HTML and plain-text parts both contain the link, followed by the alternative of the one-time code

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
`/login` SHALL offer an email field and a submit button. After submission, it shows a translated confirmation with the address, a field to enter the one-time code from the email with a button to sign in with it, and a resend action that is disabled for 60 seconds. On 429 it shows how long to wait. The code field SHALL consist of 6 single-character slots in two groups of three with a hyphen between them, accept only letters and digits, show letters in upper case, and accept a pasted code with or without the hyphen. A correct code SHALL sign the visitor in on this page. A rejected code SHALL show a translated message that the code is wrong or expired, and keep the field for another try.

#### Scenario: Submit
- **WHEN** a visitor enters their email on `/login` and submits
- **THEN** the page shows "Check your inbox" with that address, a field for the one-time code, and the resend action is disabled for 60 seconds

#### Scenario: Paste the code
- **WHEN** the visitor pastes `k7q-4mz` into the first slot
- **THEN** the slots show `K`, `7`, `Q`, `4`, `M`, `Z`

#### Scenario: Sign in with the code
- **WHEN** the visitor enters the code from the email and submits it
- **THEN** they are signed in and taken to `/`

#### Scenario: Wrong code
- **WHEN** the visitor submits a wrong code
- **THEN** the page shows that the code is wrong or expired and the visitor stays signed out

#### Scenario: Already signed in
- **WHEN** a signed-in user opens `/login`
- **THEN** they are redirected to `/`

### Requirement: Development mail fallback
When no SMTP host is configured, the server SHALL write each sign-in link and its one-time code to its log instead of sending email, and SHALL log a warning at startup that mail delivery is disabled.

#### Scenario: No SMTP configured
- **WHEN** the server runs without `FAMODASH_SMTP_HOST` and a link is requested
- **THEN** the full link and the one-time code appear in the server log and the request still responds 202

### Requirement: One-time code
Each sign-in email SHALL carry, besides its link, a one-time code of 6 characters from the set `ABCDEFGHJKMNPQRSTUVWXYZ23456789`, chosen at random. Wherever the code is shown, it SHALL be written as two groups of three characters separated by a hyphen, such as `K7Q-4MZ`. The code belongs to the link: it SHALL expire with it, and redeeming either the link or the code SHALL use up both. The server SHALL store only a hash of the code.

#### Scenario: Code format
- **WHEN** a sign-in email is sent
- **THEN** it contains a code of 6 characters from the allowed set, written like `K7Q-4MZ`

#### Scenario: Link used first
- **WHEN** the link of an email was redeemed and its code is entered afterwards
- **THEN** the code is rejected with `invalid_code`

#### Scenario: Code used first
- **WHEN** the code of an email was redeemed and its link is opened afterwards
- **THEN** the link is rejected as invalid

#### Scenario: Code not stored in plain text
- **WHEN** the token store is inspected after a code was issued
- **THEN** it contains no copy of the plain code

### Requirement: Redeem a one-time code
`POST /api/auth/magic-link/code` with `{"email": "...", "code": "..."}` SHALL, for a valid unused and unexpired code that was sent to that address and requested by the browser making this request (identified by the login-request cookie), use up the code and its link, create the account for the address if none exists, start a session and respond 200 with the user. The email is trimmed and lowercased. The code is compared case-insensitively, ignoring spaces and hyphens. In every other case, including a missing login-request cookie, the server SHALL respond 400 with `error.code` `invalid_code`, start no session, and not reveal which case applies.

#### Scenario: Correct code
- **WHEN** a browser requests a sign-in email for `ann@example.org` and posts its code in lower case without the hyphen
- **THEN** the server responds 200 with the user and sets a session cookie

#### Scenario: Other browser
- **WHEN** the correct code is posted from a browser without the requesting browser's login-request cookie
- **THEN** the server responds 400 with `invalid_code` and the code can still be used in the requesting browser

#### Scenario: Wrong address
- **WHEN** the correct code is posted with a different email address
- **THEN** the server responds 400 with `invalid_code`

#### Scenario: Expired code
- **WHEN** a code is posted 16 minutes after its email was requested
- **THEN** the server responds 400 with `invalid_code`

### Requirement: Limited code attempts
A wrong code SHALL count as a failed attempt against every unused, unexpired code sent to that address for the requesting browser. A code with 5 failed attempts SHALL no longer be accepted, even when entered correctly. Its link SHALL stay usable.

#### Scenario: Locked after five wrong codes
- **WHEN** a browser posts 5 wrong codes for its address and then the correct one
- **THEN** the correct code is rejected with `invalid_code`

#### Scenario: Link still works
- **WHEN** the code of an email is locked and the email's link is opened in the requesting browser
- **THEN** the visitor is signed in
