# Spec Delta

## MODIFIED Requirements

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
