# Spec Delta

## ADDED Requirements

### Requirement: Closed registration
When registration is not allowed (see "Registration switch" in the `app-hosting` capability), a valid sign-in link request for an address without an account SHALL send a "registration closed" email to that address instead of a sign-in email. That email SHALL contain no sign-in link and no one-time code. The response SHALL be the same 202 as for an address with an account, and the request SHALL count against the same rate limits as a sign-in link request. Requests for addresses with an account SHALL get the sign-in email as usual.

#### Scenario: Unknown address with registration closed
- **WHEN** registration is not allowed and a client requests a link for `new@example.org`, which has no account
- **THEN** the server responds 202 and sends a "registration closed" email to `new@example.org` that contains no link and no code

#### Scenario: Known address with registration closed
- **WHEN** registration is not allowed and a client requests a link for `ann@example.org`, which has an account
- **THEN** the server responds 202 and sends the sign-in email with link and code to `ann@example.org`

#### Scenario: Responses are identical
- **WHEN** registration is not allowed and a client requests a link once for an address with an account and once for an address without one
- **THEN** both responses have status 202 and identical bodies and set the login-request cookie the same way

#### Scenario: Rate limited like sign-in requests
- **WHEN** registration is not allowed and a link is requested twice within 60 seconds for an address without an account
- **THEN** the second request gets 429 `rate_limited` with a `Retry-After` header and no email is sent

#### Scenario: Registration allowed
- **WHEN** registration is allowed and a client requests a link for an address without an account
- **THEN** the server sends the sign-in email with link and code

### Requirement: Localized registration-closed email
The "registration closed" email SHALL be written in the requested locale (falling back to `en`). It SHALL say that someone asked to sign in with this address, that no account exists for it, and that registration of new accounts is closed on this instance, and that the recipient can ask the instance's administrator for access or ignore the email if they didn't request it. Wherever the subject and body name the product, they SHALL use the configured app title. It SHALL be sent as plain text with an HTML alternative, both with the same content.

#### Scenario: English email
- **WHEN** registration is not allowed, the server runs without `BOARDLY_APP_TITLE`, and a link is requested with locale `en` for an address without an account
- **THEN** the email subject is "Registration for Boardly is closed" and the body is English

#### Scenario: German email with custom app title
- **WHEN** registration is not allowed, the server runs with `BOARDLY_APP_TITLE="Boardly Staging"`, and a link is requested with locale `de` for an address without an account
- **THEN** the email subject is "Registrierung bei Boardly Staging ist geschlossen", and both parts are German and name "Boardly Staging"

## MODIFIED Requirements

### Requirement: Redeem a sign-in link
`POST /api/auth/magic-link/verify` with `{"token": "...", "confirm": true|false}` SHALL, for a valid unused and unexpired token, mark the token as used, create the account for its email if none exists, start a session, and respond 200 with the user. Without `"confirm": true`, it SHALL redeem the token only when the request carries the login-request cookie of the browser that requested the link. Otherwise it responds 403 with `error.code` `confirmation_required` and leaves the token unused. When registration is not allowed and no account exists for the token's email, it SHALL create no account, start no session and respond 400 with `error.code` `invalid_token`.

#### Scenario: First sign-in creates account
- **WHEN** registration is allowed and a valid token is redeemed for an address with no account
- **THEN** an account is created for that address, a session cookie is set, and the response body is the new user with `name` `null`

#### Scenario: Existing account
- **WHEN** a valid token is redeemed for an address with an account
- **THEN** no new account is created and a session for the existing account is started

#### Scenario: Unconfirmed redemption from another browser
- **WHEN** a valid token is posted without `"confirm": true` and without the requesting browser's login-request cookie
- **THEN** the server responds 403 with `confirmation_required`, sets no session cookie, and the token can still be redeemed

#### Scenario: Link for unknown address after registration was closed
- **WHEN** a link was issued for an address without an account while registration was allowed, and it is redeemed after the server was restarted with registration not allowed
- **THEN** the server responds 400 with `invalid_token`, creates no account and sets no session cookie

### Requirement: Redeem a one-time code
`POST /api/auth/magic-link/code` with `{"email": "...", "code": "..."}` SHALL, for a valid unused and unexpired code that was sent to that address and requested by the browser making this request (identified by the login-request cookie), use up the code and its link, create the account for the address if none exists, start a session and respond 200 with the user. The email is trimmed and lowercased. The code is compared case-insensitively, ignoring spaces and hyphens. When registration is not allowed and no account exists for the address, it SHALL create no account and start no session. In every other case, including a missing login-request cookie or a closed registration for an address without an account, the server SHALL respond 400 with `error.code` `invalid_code`, start no session, and not reveal which case applies.

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

#### Scenario: Code for unknown address after registration was closed
- **WHEN** a code was issued for an address without an account while registration was allowed, and it is posted after the server was restarted with registration not allowed
- **THEN** the server responds 400 with `invalid_code`, creates no account and sets no session cookie
