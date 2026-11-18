## ADDED Requirements

### Requirement: Security headers
Every response of the web app, meaning the HTML document (also as the fallback for client-side routes), static files and web app manifests, SHALL carry these headers:
- `Content-Security-Policy` with `default-src 'self'`; a `script-src` of `'self'`, the `sha256` hash of every inline script in the served HTML document, the origin of every absolute `src` of a script tag in it, and the sources from `BOARDLY_CSP_SCRIPT_SRC`; `worker-src 'self' blob:`; `object-src 'none'`; `base-uri 'self'`; `form-action 'self'`; `frame-ancestors 'self'`; and directives that allow images, media, connections, frames, fonts and styles (including inline styles) from any source;
- `X-Frame-Options: SAMEORIGIN`;
- `X-Content-Type-Options: nosniff`;
- `Referrer-Policy: strict-origin-when-cross-origin`.

The hashes and origins SHALL be taken from the document as it is served, after the app title, legal links and analytics snippet are inserted. An analytics snippet therefore runs without further configuration. `BOARDLY_CSP_SCRIPT_SRC` is optional: a space-separated list of extra script sources. A value containing `;` or `,` SHALL stop startup with an error that names the variable.

JSON responses under `/api`, including error responses, SHALL carry `X-Content-Type-Options: nosniff` and, unless the response sets its own policy, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`.

#### Scenario: Document headers
- **WHEN** a client requests `GET /` or `GET /settings`
- **THEN** the response has a `Content-Security-Policy` whose `script-src` contains `'self'` and the hash of the theme script, with `object-src 'none'` and `frame-ancestors 'self'`, and the headers `X-Frame-Options: SAMEORIGIN`, `X-Content-Type-Options: nosniff` and `Referrer-Policy: strict-origin-when-cross-origin`

#### Scenario: Static file headers
- **WHEN** a client requests a hashed JavaScript bundle
- **THEN** the response carries the same four headers

#### Scenario: Analytics snippet is allowed
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET='<script defer src="https://stats.example.org/script.js"></script><script>window.a = 1;</script>'` and a client requests `GET /`
- **THEN** the `script-src` contains `https://stats.example.org` and the hash of `window.a = 1;`

#### Scenario: Injected script is blocked
- **WHEN** a script that is neither from Boardly's origin nor inline in the served document tries to run on the page
- **THEN** the browser blocks it according to the policy

#### Scenario: Extra script sources
- **WHEN** the server starts with `BOARDLY_CSP_SCRIPT_SRC="https://cdn.example.org"`
- **THEN** the `script-src` of the document contains `https://cdn.example.org`

#### Scenario: Invalid extra sources
- **WHEN** the server starts with `BOARDLY_CSP_SCRIPT_SRC="https://a.example; img-src *"`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_CSP_SCRIPT_SRC`

#### Scenario: Features keep working
- **WHEN** a signed-in user opens a note with a map, plays a radio station, opens a Spotify embed, opens an offline copy and switches the theme in the production build
- **THEN** all of them work and the browser reports no policy violation

#### Scenario: API responses
- **WHEN** a client requests `GET /api/me` without a session
- **THEN** the 401 JSON response carries `X-Content-Type-Options: nosniff` and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`

#### Scenario: Own policies are kept
- **WHEN** a client requests an attachment at `GET /api/attachments/{id}`
- **THEN** its `Content-Security-Policy` is still `sandbox`
