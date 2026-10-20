# Spec Delta

## ADDED Requirements

### Requirement: Session records its sign-in method
Every session SHALL record whether it was created by magic link or by passkey. The method SHALL NOT change for the lifetime of the session.

#### Scenario: Passkey session
- **WHEN** a user signs in with a passkey
- **THEN** the session is recorded as a passkey session and never gets a passkey enrollment window

## MODIFIED Requirements

### Requirement: Protected pages in the web app
Pages that require a user SHALL redirect signed-out visitors to `/login`. While the session state is loading, the web app SHALL show neither protected content nor the signed-out landing page.

#### Scenario: Signed-out visitor opens settings
- **WHEN** a signed-out visitor opens `/settings`
- **THEN** the browser shows `/login`
