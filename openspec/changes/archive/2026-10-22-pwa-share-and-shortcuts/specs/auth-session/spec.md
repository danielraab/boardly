# Spec Delta

## ADDED Requirements

### Requirement: Return to the requested page after sign-in
The login page SHALL accept a `redirect` query parameter. After a sign-in completed on the login page itself, it SHALL navigate to `redirect` if that is a local path: it starts with a single `/`, doesn't start with `//` or `/\`, and isn't a `/login` path. Otherwise, and when `redirect` is missing, it SHALL navigate to `/`. A signed-in user who opens the login page with a valid `redirect` SHALL be sent there instead of `/`.

#### Scenario: Local redirect
- **WHEN** a visitor opens `/login?redirect=%2Fshare%3Furl%3Dhttps%253A%252F%252Fexample.org` and signs in with a passkey
- **THEN** the browser shows `/share?url=https%3A%2F%2Fexample.org`

#### Scenario: Foreign redirect
- **WHEN** a visitor opens `/login?redirect=https%3A%2F%2Fevil.example` or `/login?redirect=%2F%2Fevil.example` and signs in
- **THEN** the browser shows `/`
