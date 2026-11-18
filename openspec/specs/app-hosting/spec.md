# app-hosting Specification

## Purpose

Defines how Boardly runs as a single self-hosted binary. The binary serves the JSON REST API and the web app, is configured from the environment, and keeps its data in a local database.

## Requirements

### Requirement: Single binary serves API and web app
The server SHALL serve the REST API under `/api/` and the built web app from the same binary and port. It SHALL NOT need any other files at runtime besides its database file and the files it writes itself to its data directory (`BOARDLY_DATA_DIR`, default `./data`). It SHALL create the data directory and its `images` subdirectory on start if they don't exist.

#### Scenario: Web app is served
- **WHEN** a client requests `GET /`
- **THEN** the server responds with status 200 and the web app's HTML document

#### Scenario: Static assets are served
- **WHEN** a client requests a file that exists in the web app build, such as a hashed JavaScript bundle
- **THEN** the server responds with status 200 and that file's content and content type

#### Scenario: Fresh data directory
- **WHEN** the server starts with `BOARDLY_DATA_DIR` pointing to a directory that doesn't exist yet
- **THEN** it creates the directory and its `images` subdirectory and starts normally

### Requirement: Client-side routes fall back to the web app
For a `GET` request outside `/api/` whose path matches no built file, the server SHALL respond with the web app's HTML document and status 200, so client-side routes work on reload.

#### Scenario: Deep link reload
- **WHEN** a client requests `GET /settings` directly
- **THEN** the server responds with status 200 and the web app's HTML document

### Requirement: API errors use a JSON format
Every API error response SHALL have a JSON body of the form `{"error": {"code": "<machine_code>", "message": "<text>"}}` and a matching HTTP status. This also applies to unknown `/api/` paths, which SHALL NOT fall back to the web app.

#### Scenario: Unknown API path
- **WHEN** a client requests `GET /api/does-not-exist`
- **THEN** the server responds with status 404 and a JSON body whose `error.code` is `not_found`

#### Scenario: Wrong method
- **WHEN** a client sends `DELETE /api/health`
- **THEN** the server responds with status 405 and a JSON body whose `error.code` is `method_not_allowed`

### Requirement: Health endpoint
The server SHALL expose `GET /api/health`. It responds with status 200 and `{"status": "ok"}` when the database is reachable, and with status 503 and an error body when it is not.

#### Scenario: Healthy server
- **WHEN** a client requests `GET /api/health` and the database is reachable
- **THEN** the server responds with status 200 and `{"status": "ok"}`

### Requirement: Configuration from environment
The server SHALL read its configuration from environment variables prefixed with `BOARDLY_`. `BOARDLY_BASE_URL` is required and has no default. Missing optional settings get documented defaults. A missing required setting or an invalid value SHALL stop startup with an error message that names the variable.

#### Scenario: Defaults apply
- **WHEN** the server starts with `BOARDLY_BASE_URL` set and no other `BOARDLY_` variables
- **THEN** it listens on port 8080 and stores its database at `./data/boardly.db`

#### Scenario: Base URL missing
- **WHEN** the server starts with `BOARDLY_BASE_URL` unset or empty
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_BASE_URL`

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_PORT=abc`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_PORT`

### Requirement: Configurable app title
The server SHALL read the instance's app title from `BOARDLY_APP_TITLE`. Leading and trailing whitespace is removed, and an unset or empty value gives the default `Boardly`. The app title replaces the product name as a whole wherever the web app, its manifests and the sign-in email show it. The web app's HTML document SHALL carry the app title as its `<title>` and as the content of a `<meta name="application-name">` tag, escaped for HTML, on every response that serves the document, including the fallback for client-side routes.

#### Scenario: Default title
- **WHEN** the server starts without `BOARDLY_APP_TITLE` and a client requests `GET /`
- **THEN** the HTML document's title is "Boardly" and its `application-name` meta tag has the content "Boardly"

#### Scenario: Custom title
- **WHEN** the server starts with `BOARDLY_APP_TITLE="Boardly Staging"` and a client requests `GET /settings`
- **THEN** the HTML document's title is "Boardly Staging" and its `application-name` meta tag has the content "Boardly Staging"

#### Scenario: Blank value
- **WHEN** the server starts with `BOARDLY_APP_TITLE="   "`
- **THEN** the app title is "Boardly"

#### Scenario: Title with markup characters
- **WHEN** the server starts with `BOARDLY_APP_TITLE="A & B <test>"` and a client requests `GET /`
- **THEN** the HTML document's title reads "A & B <test>" as text, and the document contains no `<test>` element

### Requirement: Analytics snippet
The server SHALL read an optional analytics snippet from `BOARDLY_ANALYTICS_SNIPPET` when it starts. The value is HTML, such as a `<script>` tag of an analytics tool. Leading and trailing whitespace is removed, and an unset or empty value means there is no snippet. The server SHALL insert the snippet unchanged, without escaping, directly before the closing `</head>` tag of the web app's HTML document on every response that serves the document, including the fallback for client-side routes. Without a snippet, the document SHALL be served without any addition. Changing the snippet SHALL need only a restart of the server, not a new build of the web app.

#### Scenario: No snippet by default
- **WHEN** the server starts without `BOARDLY_ANALYTICS_SNIPPET` and a client requests `GET /`
- **THEN** the HTML document contains no content besides that of the built web app before `</head>`

#### Scenario: Snippet on the start page
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET='<script defer src="https://stats.example.org/script.js" data-website-id="abc"></script>'` and a client requests `GET /`
- **THEN** the HTML document contains that script tag unchanged, directly before `</head>`

#### Scenario: Snippet on a deep link
- **WHEN** the server starts with an analytics snippet and a client requests `GET /settings`
- **THEN** the HTML document contains the snippet directly before `</head>`

#### Scenario: Snippet is not escaped
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET='<script>window.a = 1 && 2;</script>'` and a client requests `GET /`
- **THEN** the HTML document contains `<script>window.a = 1 && 2;</script>` as written, with no `&amp;` or `&lt;`

#### Scenario: Blank value
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET="   "`
- **THEN** there is no snippet, and the HTML document is served without any addition

#### Scenario: Other files are unchanged
- **WHEN** the server starts with an analytics snippet and a client requests a hashed JavaScript bundle or a web app manifest
- **THEN** the response doesn't contain the snippet

### Requirement: Database schema is migrated on startup
On startup, the server SHALL apply every pending database migration before it accepts requests. Each migration SHALL be applied exactly once, even across restarts.

#### Scenario: Fresh start
- **WHEN** the server starts and no database file exists
- **THEN** it creates the database, applies all migrations and then serves requests

#### Scenario: Restart
- **WHEN** the server restarts against a database that already has every migration applied
- **THEN** it applies no migration again and starts normally

### Requirement: Health check command
The binary SHALL provide a `healthcheck` subcommand. It requests `/api/health` from the server on the local host at the configured port. It exits with status 0 if the response is 200 and with a non-zero status otherwise, including when the server can't be reached or doesn't answer within a few seconds. It SHALL NOT need any setting other than the port.

#### Scenario: Healthy server
- **WHEN** the server is running on the configured port and its database is reachable, and `boardly healthcheck` runs
- **THEN** the command exits with status 0

#### Scenario: Server not reachable
- **WHEN** no server listens on the configured port and `boardly healthcheck` runs
- **THEN** the command exits with a non-zero status

#### Scenario: Unhealthy server
- **WHEN** the server answers `/api/health` with status 503 and `boardly healthcheck` runs
- **THEN** the command exits with a non-zero status

#### Scenario: Only the port is needed
- **WHEN** `boardly healthcheck` runs with `BOARDLY_PORT` set and `BOARDLY_BASE_URL` unset
- **THEN** it checks the server on that port and doesn't fail because of the missing base URL

### Requirement: Request logging
The server SHALL write one log line per HTTP request that it has finished answering, with the method, path, status and duration, according to `BOARDLY_LOG_REQUESTS`:
- `all` (the default when the variable is unset or empty): every request is logged
- `errors`: only requests answered with status 500 or higher are logged
- `none`: no request is logged

Any other value SHALL stop startup with an error message that names `BOARDLY_LOG_REQUESTS`. In every mode, a `GET /api/health` request answered with status 200 SHALL NOT be logged; a health request with any other status follows the rules of the mode. Requests for the web app's files SHALL be treated like any other request. The setting SHALL NOT affect any other log output.

#### Scenario: All requests by default
- **WHEN** the server starts without `BOARDLY_LOG_REQUESTS` and a client requests `GET /api/dashboards`, `GET /` and a hashed JavaScript bundle
- **THEN** the log contains one request line for each of them

#### Scenario: Errors only
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=errors` and answers one request with status 200, one with 404 and one with 500
- **THEN** the log contains a request line only for the request answered with 500

#### Scenario: No request logging
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=none` and answers one request with status 200 and one with 500
- **THEN** the log contains no request line

#### Scenario: Successful health check is not logged
- **WHEN** the server runs with `BOARDLY_LOG_REQUESTS=all` and a client requests `GET /api/health` while the database is reachable
- **THEN** the log contains no request line for it

#### Scenario: Failing health check is logged
- **WHEN** the server runs with `BOARDLY_LOG_REQUESTS=errors` and a client requests `GET /api/health` while the database is not reachable
- **THEN** the log contains a request line with status 503

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=verbose`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_LOG_REQUESTS`

#### Scenario: Other log output is unchanged
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=none`
- **THEN** startup messages and errors logged while handling requests still appear in the log

### Requirement: Container image
The repository SHALL contain a container build that produces an image of the whole app from a clean checkout, with no tools needed on the host besides the container builder. The image runs the server as a non-root user, listens on port 8080, keeps its database and caches under `/data`, and reports its health through the health check command.

#### Scenario: Build from a clean checkout
- **WHEN** the image is built from a fresh clone of the repository
- **THEN** the build succeeds, and the image serves both the web app and the API

#### Scenario: Runs with only the base URL
- **WHEN** a container starts from the image with `BOARDLY_BASE_URL` set and an empty volume mounted at `/data`
- **THEN** the server creates its database under `/data`, listens on port 8080, and the container reports healthy

#### Scenario: Refuses to start without base URL
- **WHEN** a container starts from the image without `BOARDLY_BASE_URL`
- **THEN** the container exits with a non-zero status and logs an error naming `BOARDLY_BASE_URL`

#### Scenario: Not running as root
- **WHEN** the container is running
- **THEN** the server process runs with a non-zero user ID

#### Scenario: Data survives a container replacement
- **WHEN** a container is removed and a new one starts from the image with the same volume at `/data`
- **THEN** the new container serves the same accounts and data

#### Scenario: Local secrets stay out of the image
- **WHEN** the image is built from a working copy that contains `.env` or database files
- **THEN** none of these files are in the image or its build stages

### Requirement: Registration switch
The server SHALL read from `BOARDLY_ALLOW_REGISTRATION` whether new accounts may be created. The value is `true` or `false`, and an unset or empty value gives the default `true`. Any other value SHALL stop startup with an error message that names `BOARDLY_ALLOW_REGISTRATION`. How sign-in behaves when registration is not allowed is described in the `auth-magic-link` capability.

#### Scenario: Default allows registration
- **WHEN** the server starts without `BOARDLY_ALLOW_REGISTRATION`
- **THEN** registration is allowed

#### Scenario: Registration disabled
- **WHEN** the server starts with `BOARDLY_ALLOW_REGISTRATION=false`
- **THEN** registration is not allowed

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_ALLOW_REGISTRATION=maybe`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_ALLOW_REGISTRATION`

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

### Requirement: Client IP behind reverse proxies
The server SHALL use the connection's remote address as the client IP for rate limits, unless `BOARDLY_TRUST_PROXY_HEADERS` is `true`. In that case it SHALL read `BOARDLY_TRUSTED_PROXY_COUNT`, an integer from 1 to 10 (default `1`) that gives the number of reverse proxies in front of the server. Other values stop startup with an error that names the variable. The server SHALL join all `X-Forwarded-For` header lines into one comma-separated list. It SHALL take the entry at position `count` from the right as the client IP, or the leftmost entry when the list has fewer entries. When there is no `X-Forwarded-For` header, or the chosen entry is not a valid IP address, it SHALL use the connection's remote address. Entries that a client adds itself, to the left of the proxies' entries, SHALL never be used.

#### Scenario: Spoofed entry behind one proxy
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true`, `BOARDLY_TRUSTED_PROXY_COUNT` is unset and a request arrives with `X-Forwarded-For: 1.2.3.4, 203.0.113.7`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Two proxies
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true`, `BOARDLY_TRUSTED_PROXY_COUNT=2` and a request arrives with `X-Forwarded-For: 1.2.3.4, 203.0.113.7, 198.51.100.2`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Several header lines
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true` and a request has two `X-Forwarded-For` lines, `1.2.3.4` and `203.0.113.7`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Invalid entry
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true` and the chosen entry is `unknown`
- **THEN** the client IP is the connection's remote address

#### Scenario: Headers not trusted
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS` is unset and a request arrives with `X-Forwarded-For: 203.0.113.7`
- **THEN** the client IP is the connection's remote address

#### Scenario: Invalid count
- **WHEN** the server starts with `BOARDLY_TRUSTED_PROXY_COUNT=0`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_TRUSTED_PROXY_COUNT`

#### Scenario: Faked addresses don't escape the link limit
- **WHEN** one client behind a single trusted proxy requests sign-in links for different addresses in quick succession, each request with a different made-up leftmost `X-Forwarded-For` entry
- **THEN** the requests count against that client's one per-IP limit, and the first request over it gets 429 `rate_limited`
