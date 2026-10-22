# app-hosting Specification

## Purpose

Defines how Famodash runs as a single self-hosted binary. The binary serves the JSON REST API and the web app, is configured from the environment, and keeps its data in a local database.

## Requirements

### Requirement: Single binary serves API and web app
The server SHALL serve the REST API under `/api/` and the built web app from the same binary and port. It SHALL NOT need any other files at runtime besides its database file and the files it writes itself to its data directory (`FAMODASH_DATA_DIR`, default `./data`). It SHALL create the data directory and its `images` subdirectory on start if they don't exist.

#### Scenario: Web app is served
- **WHEN** a client requests `GET /`
- **THEN** the server responds with status 200 and the web app's HTML document

#### Scenario: Static assets are served
- **WHEN** a client requests a file that exists in the web app build, such as a hashed JavaScript bundle
- **THEN** the server responds with status 200 and that file's content and content type

#### Scenario: Fresh data directory
- **WHEN** the server starts with `FAMODASH_DATA_DIR` pointing to a directory that doesn't exist yet
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
The server SHALL read its configuration from environment variables prefixed with `FAMODASH_`. `FAMODASH_BASE_URL` is required and has no default. Missing optional settings get documented defaults. A missing required setting or an invalid value SHALL stop startup with an error message that names the variable.

#### Scenario: Defaults apply
- **WHEN** the server starts with `FAMODASH_BASE_URL` set and no other `FAMODASH_` variables
- **THEN** it listens on port 8080 and stores its database at `./data/famodash.db`

#### Scenario: Base URL missing
- **WHEN** the server starts with `FAMODASH_BASE_URL` unset or empty
- **THEN** it exits with a non-zero status and an error message that names `FAMODASH_BASE_URL`

#### Scenario: Invalid value
- **WHEN** the server starts with `FAMODASH_PORT=abc`
- **THEN** it exits with a non-zero status and an error message that names `FAMODASH_PORT`

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
- **WHEN** the server is running on the configured port and its database is reachable, and `famodash healthcheck` runs
- **THEN** the command exits with status 0

#### Scenario: Server not reachable
- **WHEN** no server listens on the configured port and `famodash healthcheck` runs
- **THEN** the command exits with a non-zero status

#### Scenario: Unhealthy server
- **WHEN** the server answers `/api/health` with status 503 and `famodash healthcheck` runs
- **THEN** the command exits with a non-zero status

#### Scenario: Only the port is needed
- **WHEN** `famodash healthcheck` runs with `FAMODASH_PORT` set and `FAMODASH_BASE_URL` unset
- **THEN** it checks the server on that port and doesn't fail because of the missing base URL

### Requirement: Container image
The repository SHALL contain a container build that produces an image of the whole app from a clean checkout, with no tools needed on the host besides the container builder. The image runs the server as a non-root user, listens on port 8080, keeps its database and caches under `/data`, and reports its health through the health check command.

#### Scenario: Build from a clean checkout
- **WHEN** the image is built from a fresh clone of the repository
- **THEN** the build succeeds, and the image serves both the web app and the API

#### Scenario: Runs with only the base URL
- **WHEN** a container starts from the image with `FAMODASH_BASE_URL` set and an empty volume mounted at `/data`
- **THEN** the server creates its database under `/data`, listens on port 8080, and the container reports healthy

#### Scenario: Refuses to start without base URL
- **WHEN** a container starts from the image without `FAMODASH_BASE_URL`
- **THEN** the container exits with a non-zero status and logs an error naming `FAMODASH_BASE_URL`

#### Scenario: Not running as root
- **WHEN** the container is running
- **THEN** the server process runs with a non-zero user ID

#### Scenario: Data survives a container replacement
- **WHEN** a container is removed and a new one starts from the image with the same volume at `/data`
- **THEN** the new container serves the same accounts and data

#### Scenario: Local secrets stay out of the image
- **WHEN** the image is built from a working copy that contains `.env` or database files
- **THEN** none of these files are in the image or its build stages
