## MODIFIED Requirements

### Requirement: Session cookie
A successful sign-in SHALL set a cookie `boardly_session` that holds a random session identifier of at least 256 bits, with `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` when the base URL uses HTTPS. The server SHALL store only a hash of the identifier.

#### Scenario: Cookie attributes
- **WHEN** a sign-in succeeds on a server whose base URL starts with `https://`
- **THEN** the response sets `boardly_session` with `HttpOnly`, `Secure`, `SameSite=Lax` and `Path=/`
