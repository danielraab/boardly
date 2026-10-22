# auth-passkey Specification

## Purpose

Lets users sign in to Boardly with passkeys (WebAuthn discoverable credentials) and manage them: adding a passkey within a short window after an email sign-in, seeing which provider holds it, and deleting it.
## Requirements
### Requirement: Relying party identity
Passkeys SHALL be bound to a relying party ID equal to the host name of the configured base URL, unless one is configured explicitly. Only the base URL's origin is accepted.

#### Scenario: Default RP ID
- **WHEN** the base URL is `https://home.example.org` and no RP ID is configured
- **THEN** new passkeys are created for RP ID `home.example.org` and ceremonies from other origins are rejected

### Requirement: Sign in with a passkey
`/login` SHALL let a visitor sign in with any passkey registered with this Boardly instance, without entering an email address. A successful sign-in starts a session for the passkey's owner and takes them to `/`.

#### Scenario: Button sign-in
- **WHEN** a visitor on `/login` activates "Sign in with passkey" and completes the browser prompt with a registered passkey
- **THEN** they are signed in as that passkey's owner and see `/`

### Requirement: Passkey autofill on the login page
In browsers that support conditional mediation, the email field on `/login` SHALL offer the visitor's passkeys in its autofill suggestions. Picking one signs them in like the button does.

#### Scenario: Autofill sign-in
- **WHEN** a visitor focuses the email field in a supporting browser and picks a passkey from the autofill suggestions
- **THEN** they are signed in without submitting the email form

### Requirement: Failed passkey sign-in
A passkey sign-in that the server can't verify SHALL respond 401 with `error.code` `passkey_invalid`, or `passkey_unknown` when the credential isn't registered (e.g. it was deleted). The login page shows a translated error and keeps the email form usable. A cancelled browser prompt shows no error.

#### Scenario: Deleted passkey used
- **WHEN** a visitor signs in with a passkey that its owner deleted from Boardly
- **THEN** no session is started and the login page says this passkey is no longer valid and suggests the email link

#### Scenario: Prompt cancelled
- **WHEN** the visitor cancels the browser's passkey prompt
- **THEN** the login page shows no error

### Requirement: Passkey enrollment window
A sign-in by magic link SHALL open a passkey enrollment window that ends 5 minutes later. A sign-in by passkey SHALL NOT open one. `GET /api/me` SHALL include `passkeyEnrollmentUntil` (RFC 3339 timestamp, or `null` when no window is open).

#### Scenario: After magic link
- **WHEN** a user signs in by magic link and calls `GET /api/me`
- **THEN** `passkeyEnrollmentUntil` is 5 minutes after the sign-in

#### Scenario: After passkey
- **WHEN** a user signs in with a passkey and calls `GET /api/me`
- **THEN** `passkeyEnrollmentUntil` is `null`

### Requirement: Registration requires an open window
Starting a passkey registration SHALL be refused with 403 and `error.code` `enrollment_window_closed` when the current session's window isn't open. A registration started while the window was open SHALL still be completable while its challenge is valid.

#### Scenario: Window closed
- **WHEN** a user signed in by magic link 6 minutes ago tries to add a passkey
- **THEN** the server responds 403 with `enrollment_window_closed`

#### Scenario: Window closes during the prompt
- **WHEN** a user starts adding a passkey 4m50s after a magic-link sign-in and finishes the browser prompt 30 seconds later
- **THEN** the passkey is saved

### Requirement: Register a passkey
A registered passkey SHALL be a discoverable credential owned by the current user. The browser SHALL be told which passkeys the user already has, so the same authenticator isn't registered twice. The user can give an optional label of up to 64 characters (trimmed, empty means no label).

#### Scenario: Add with label
- **WHEN** a user within the window adds a passkey with label "Work laptop"
- **THEN** the passkey list contains a new entry labelled "Work laptop"

#### Scenario: Add without label
- **WHEN** a user adds a passkey and leaves the label empty
- **THEN** the new entry has no label and is named after its detected provider, or "Passkey" if the provider is unknown

#### Scenario: Label too long
- **WHEN** a client completes registration with a label of 65 characters
- **THEN** the server responds 400 with `error.code` `invalid_label` and saves no passkey

### Requirement: Provider detection
For each passkey, the server SHALL determine the provider from the authenticator's AAGUID using a locally cached copy of the community AAGUID list, and return its name and light and dark icons. An unknown or all-zero AAGUID, or an unavailable list, SHALL yield no provider. Passkeys work the same either way.

#### Scenario: Known provider
- **WHEN** a passkey with the AAGUID of Google Password Manager is listed
- **THEN** its entry includes the provider name "Google Password Manager" and its icons

#### Scenario: Unknown provider
- **WHEN** a passkey with an all-zero AAGUID is listed
- **THEN** its entry has provider `null` and the UI shows a generic key icon

#### Scenario: List refresh
- **WHEN** the server runs and its cached list is older than 24 hours
- **THEN** it downloads the list again and uses the new copy without a restart

#### Scenario: Offline server
- **WHEN** the server can't download the list and has no cached copy
- **THEN** passkeys are listed with provider `null` and sign-in and registration still work

### Requirement: List passkeys
`GET /api/passkeys` SHALL return only the current user's passkeys, newest first, each with id, label, provider, creation time and last-used time.

#### Scenario: Only own passkeys
- **WHEN** two users each have one passkey and one of them lists passkeys
- **THEN** the response contains exactly that user's passkey

### Requirement: Delete a passkey
A user SHALL be able to delete any of their passkeys at any time, regardless of the enrollment window, after confirming in a dialog. Deleting a passkey another user owns, or one that doesn't exist, responds 404. The current session stays valid even if it was created with the deleted passkey.

#### Scenario: Delete outside the window
- **WHEN** a user signed in by passkey an hour ago deletes one of their passkeys and confirms
- **THEN** the passkey is removed from the list and can no longer be used to sign in

#### Scenario: Foreign passkey
- **WHEN** a client calls `DELETE /api/passkeys/<id of another user's passkey>`
- **THEN** the server responds 404 and the passkey is kept

### Requirement: Unsupported browsers
When the browser doesn't support WebAuthn, the web app SHALL hide the passkey sign-in button and disable adding passkeys with a translated explanation. Email sign-in stays available.

#### Scenario: No WebAuthn
- **WHEN** `/login` is opened in a browser without WebAuthn support
- **THEN** no "Sign in with passkey" button is shown and the email form works

