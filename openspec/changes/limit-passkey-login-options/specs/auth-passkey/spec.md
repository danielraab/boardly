## ADDED Requirements

### Requirement: Rate limit on passkey sign-in options
The server SHALL limit `POST /api/auth/passkey/login/options` to 120 requests per hour with a burst of 30 for each client IP, found as described in "Client IP behind reverse proxies" in `app-hosting`. A request over the limit SHALL get status 429 with `error.code` `rate_limited` and a `Retry-After` header. It SHALL NOT store a challenge or set the challenge cookie. This limit is separate from the limit on sign-in link requests.

On the login page, a 429 after "Sign in with passkey" SHALL show the translated "too many requests" message with the seconds from `Retry-After`. A 429 during passkey autofill SHALL show no message and SHALL NOT request new options until the page is opened again or the visitor uses the button.

#### Scenario: Burst exhausted
- **WHEN** one client IP requests passkey sign-in options 31 times within a minute
- **THEN** the 31st request gets 429 `rate_limited` with a `Retry-After` header, and no challenge is stored for it

#### Scenario: Email links unaffected
- **WHEN** a client IP has used up its passkey option limit and requests a sign-in link for an address
- **THEN** the link request is not rejected because of the passkey limit

#### Scenario: Button over the limit
- **WHEN** a visitor activates "Sign in with passkey" and the server answers the options request with 429 and `Retry-After: 40`
- **THEN** the login page says there were too many requests and to try again in 40 s, and the email form stays usable

#### Scenario: Autofill over the limit
- **WHEN** the login page's autofill requests passkey options and gets 429
- **THEN** no message is shown and the page makes no further options request on its own
