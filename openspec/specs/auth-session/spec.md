# auth-session Specification

## Purpose

Defines how a signed-in user stays signed in. It covers the session cookie, its lifetime, the current-user lookup, signing out, protection against cross-site requests, and how the web app protects pages that need a user.

## Requirements

### Requirement: Session cookie
A successful sign-in SHALL set a cookie `famodash_session` that holds a random session identifier of at least 256 bits, with `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` when the base URL uses HTTPS. The server SHALL store only a hash of the identifier.

#### Scenario: Cookie attributes
- **WHEN** a sign-in succeeds on a server whose base URL starts with `https://`
- **THEN** the response sets `famodash_session` with `HttpOnly`, `Secure`, `SameSite=Lax` and `Path=/`

### Requirement: Session lifetime
A session SHALL expire after 30 days without use. Using it extends the expiry, at most once per hour. Requests with an expired or unknown session are treated as unauthenticated.

#### Scenario: Inactive session
- **WHEN** a request carries a session that was last used 31 days ago
- **THEN** the request is treated as unauthenticated

### Requirement: Current user
`GET /api/me` SHALL respond 200 with `{"id", "email", "name"}` for an authenticated request, and 401 with `error.code` `unauthenticated` otherwise. `name` is `null` when not set.

#### Scenario: Signed in
- **WHEN** a signed-in user requests `GET /api/me`
- **THEN** the response is 200 with their id, email and name

#### Scenario: Signed out
- **WHEN** a request without a session cookie calls `GET /api/me`
- **THEN** the response is 401 with `unauthenticated`

### Requirement: Sign out
`POST /api/auth/logout` SHALL delete the current session on the server, clear the cookie and respond 204. It SHALL also respond 204 when there is no valid session.

#### Scenario: Logout
- **WHEN** a signed-in user calls `POST /api/auth/logout` and then `GET /api/me` with the old cookie
- **THEN** the logout responds 204 and the next request responds 401

### Requirement: Cross-site request protection
The server SHALL reject state-changing API requests (POST, PUT, PATCH, DELETE) that don't have a JSON body content type, or whose `Origin` header is present and doesn't match the base URL's origin. It responds 403 with `error.code` `forbidden_origin`.

#### Scenario: Foreign origin
- **WHEN** a request to `POST /api/auth/logout` carries `Origin: https://evil.example`
- **THEN** the server responds 403 with `forbidden_origin` and the session stays valid

### Requirement: Protected pages in the web app
Pages that require a user SHALL redirect signed-out visitors to `/login`. While the session state is loading, the web app SHALL show neither protected content nor the signed-out landing page.

#### Scenario: Signed-out visitor opens profile
- **WHEN** a signed-out visitor opens `/profile`
- **THEN** the browser shows `/login`

### Requirement: Sign out in the web app
Signing out in the web app SHALL call the logout endpoint, discard all cached user data and take the visitor to `/`.

#### Scenario: Sign out from menu
- **WHEN** a signed-in user activates "Sign out"
- **THEN** they see the landing page at `/` and the user menu shows "Sign in"

### Requirement: Expired session in the web app
When an API request made by the web app returns 401, the web app SHALL switch to the signed-out state.

#### Scenario: Session expired while open
- **WHEN** the user's session was deleted on the server and the web app makes its next API request
- **THEN** the user menu shows "Sign in" and protected pages redirect to `/login`
