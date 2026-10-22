# Spec Delta

## ADDED Requirements

### Requirement: One-time code
Each sign-in email SHALL carry, besides its link, a one-time code of 6 characters from the set `ABCDEFGHJKMNPQRSTUVWXYZ23456789`, chosen at random. The code belongs to the link: it SHALL expire with it, and redeeming either the link or the code SHALL use up both. The server SHALL store only a hash of the code.

#### Scenario: Code format
- **WHEN** a sign-in email is sent
- **THEN** it contains a code of 6 characters from the allowed set

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
- **WHEN** a browser requests a sign-in email for `ann@example.org` and posts its code in lower case with a space in the middle
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

## MODIFIED Requirements

### Requirement: Localized sign-in email
The sign-in email SHALL be written in the requested locale. It SHALL contain the sign-in button and link, then, introduced as an alternative ("or use this one-time code"), the one-time code with a note that it is entered on the sign-in page where the email was requested. It SHALL also contain the 15-minute validity of link and code and a note that the recipient can ignore the email if they didn't request it. It SHALL be sent as plain text with an HTML alternative, both with the same content.

#### Scenario: English email
- **WHEN** a link is requested with locale `en`
- **THEN** the email subject and body are English and mention the 15-minute validity

#### Scenario: Both ways to sign in
- **WHEN** a sign-in email is sent
- **THEN** its HTML and plain-text parts both contain the link, followed by the alternative of the one-time code

### Requirement: Login page email form
`/login` SHALL offer an email field and a submit button. After submission, it shows a translated confirmation with the address, a field to enter the one-time code from the email with a button to sign in with it, and a resend action that is disabled for 60 seconds. On 429 it shows how long to wait. A correct code SHALL sign the visitor in on this page. A rejected code SHALL show a translated message that the code is wrong or expired, and keep the field for another try.

#### Scenario: Submit
- **WHEN** a visitor enters their email on `/login` and submits
- **THEN** the page shows "Check your inbox" with that address, a field for the one-time code, and the resend action is disabled for 60 seconds

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
