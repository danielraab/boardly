# Spec Delta

## Purpose

Defines how Famodash runs as a single self-hosted binary. The binary serves the JSON REST API and the web app, is configured from the environment, and keeps its data in a local database.

## ADDED Requirements

### Requirement: Single binary serves API and web app
The server SHALL serve the REST API under `/api/` and the built web app from the same binary and port. It SHALL NOT need any other files at runtime besides its database file.

#### Scenario: Web app is served
- **WHEN** a client requests `GET /`
- **THEN** the server responds with status 200 and the web app's HTML document

#### Scenario: Static assets are served
- **WHEN** a client requests a file that exists in the web app build, such as a hashed JavaScript bundle
- **THEN** the server responds with status 200 and that file's content and content type

### Requirement: Client-side routes fall back to the web app
For a `GET` request outside `/api/` whose path matches no built file, the server SHALL respond with the web app's HTML document and status 200, so client-side routes work on reload.

#### Scenario: Deep link reload
- **WHEN** a client requests `GET /profile` directly
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
The server SHALL read its configuration from environment variables prefixed with `FAMODASH_`. Missing optional settings get documented defaults. An invalid value SHALL stop startup with an error message that names the variable.

#### Scenario: Defaults apply
- **WHEN** the server starts with no `FAMODASH_` variables set
- **THEN** it listens on port 8080 and stores its database at `./data/famodash.db`

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
