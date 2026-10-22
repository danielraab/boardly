## MODIFIED Requirements

### Requirement: Sign in with a passkey
`/login` SHALL let a visitor sign in with any passkey registered with this Boardly instance, without entering an email address. A successful sign-in starts a session for the passkey's owner and takes them to `/`.

#### Scenario: Button sign-in
- **WHEN** a visitor on `/login` activates "Sign in with passkey" and completes the browser prompt with a registered passkey
- **THEN** they are signed in as that passkey's owner and see `/`

### Requirement: Failed passkey sign-in
A passkey sign-in that the server can't verify SHALL respond 401 with `error.code` `passkey_invalid`, or `passkey_unknown` when the credential isn't registered (e.g. it was deleted). The login page shows a translated error and keeps the email form usable. A cancelled browser prompt shows no error.

#### Scenario: Deleted passkey used
- **WHEN** a visitor signs in with a passkey that its owner deleted from Boardly
- **THEN** no session is started and the login page says this passkey is no longer valid and suggests the email link

#### Scenario: Prompt cancelled
- **WHEN** the visitor cancels the browser's passkey prompt
- **THEN** the login page shows no error
