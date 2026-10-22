## MODIFIED Requirements

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
