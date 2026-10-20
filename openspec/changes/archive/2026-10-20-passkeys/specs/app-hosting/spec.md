# Spec Delta

## MODIFIED Requirements

### Requirement: Client-side routes fall back to the web app
For a `GET` request outside `/api/` whose path matches no built file, the server SHALL respond with the web app's HTML document and status 200, so client-side routes work on reload.

#### Scenario: Deep link reload
- **WHEN** a client requests `GET /settings` directly
- **THEN** the server responds with status 200 and the web app's HTML document
