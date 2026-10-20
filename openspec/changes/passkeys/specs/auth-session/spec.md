# Spec Delta

## ADDED Requirements

### Requirement: Session records its sign-in method
Every session SHALL record whether it was created by magic link or by passkey. The method SHALL NOT change for the lifetime of the session.

#### Scenario: Passkey session
- **WHEN** a user signs in with a passkey
- **THEN** the session is recorded as a passkey session and never gets a passkey enrollment window
