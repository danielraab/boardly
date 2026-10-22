## MODIFIED Requirements

### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep` or `linkwarden`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder. A token's secret starts with `famo_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `famo_` and its prefix equals the first 12 characters of the secret

#### Scenario: Linkwarden token
- **WHEN** the user creates a token for the app `linkwarden`
- **THEN** the server responds 201 and the token is listed with the app `linkwarden`

### Requirement: Create dialog
The create dialog SHALL ask for a name, the app ("Karakeep", preselected, or "Linkwarden"), the root folder (picked from the favourites tree, folders only, including the top-level folder) and the expiry: "30 days", "90 days" (default), "1 year", "No expiry" or a custom date. A custom date must be in the future. After creating, the dialog SHALL show the secret once, with a copy button, the server address (also with a copy button) to enter in the app (`<FAMODASH_BASE_URL>/api/karakeep` for Karakeep, `<FAMODASH_BASE_URL>/api/linkwarden` for Linkwarden) and a clearly highlighted warning that the secret won't be shown again. After the dialog is closed, the secret can't be shown again.

#### Scenario: Show secret once
- **WHEN** the user creates a token and closes the dialog
- **THEN** the list shows the new token, and neither the list nor reopening anything shows its secret

#### Scenario: Copy
- **WHEN** the user clicks the copy button in the secret view
- **THEN** the secret is copied to the clipboard and the button shows a check mark, which turns back into the copy icon after 3 seconds

#### Scenario: Default expiry
- **WHEN** the user opens the create dialog
- **THEN** "90 days" is selected as the expiry

#### Scenario: Server address for Linkwarden
- **WHEN** the user creates a token for "Linkwarden"
- **THEN** the dialog shows the server address `<FAMODASH_BASE_URL>/api/linkwarden`

### Requirement: Bearer authentication for app endpoints
Requests to `/api/karakeep/` and `/api/linkwarden/` SHALL be authenticated only by the header `Authorization: Bearer <secret>`. The session cookie SHALL NOT authenticate these requests and is ignored. The cross-site request check of the web app's API does not apply to these paths. Procedures and endpoints marked as public in this spec or in the app's spec work without a token; every other procedure or endpoint requires an active token for the app of its path: `karakeep` under `/api/karakeep/`, `linkwarden` under `/api/linkwarden/`. A missing, malformed, unknown, revoked or expired token, or a token for another app, SHALL fail such a procedure with an `UNAUTHORIZED` error entry (HTTP status 401), and such a Linkwarden endpoint with 401 and `{"response": "You must be logged in."}`, with the same message in every case. A request authenticated by a token records the token's last use, at most once per minute.

#### Scenario: Valid token
- **WHEN** a client calls `users.whoami` with an active Karakeep token
- **THEN** the call succeeds

#### Scenario: Cookie only
- **WHEN** a signed-in browser calls `users.whoami` with its session cookie and no `Authorization` header
- **THEN** the call fails with `UNAUTHORIZED` and HTTP status 401

#### Scenario: Revoked token
- **WHEN** a client calls `users.whoami` with a revoked token
- **THEN** the call fails with the same error entry as for an unknown token

#### Scenario: Public procedure without a token
- **WHEN** a client calls `config.clientConfig` without an `Authorization` header
- **THEN** the call succeeds

#### Scenario: Last used
- **WHEN** a client makes its first request with a new token
- **THEN** the token's last-used date in the list is set to about that time

#### Scenario: Linkwarden endpoint with a Linkwarden token
- **WHEN** the Linkwarden app calls `GET /api/linkwarden/api/v1/users/me` with an active Linkwarden token
- **THEN** the call succeeds

#### Scenario: Token for another app
- **WHEN** a client calls `GET /api/linkwarden/api/v1/users/me` with an active Karakeep token
- **THEN** the server responds 401 with `{"response": "You must be logged in."}`

### Requirement: CORS for app endpoints
Responses on `/api/karakeep/` and `/api/linkwarden/` SHALL allow cross-origin requests from any origin, including browser-extension origins, without credentials: `Access-Control-Allow-Origin: *`. Preflight `OPTIONS` requests SHALL be answered with 204 before authentication, with the allowed methods `GET, POST, PUT, DELETE` and the request headers the preflight asks for (`Access-Control-Request-Headers`), because the extensions can add user-configured headers. The web app's other `/api/` endpoints are unchanged.

#### Scenario: Preflight from an extension
- **WHEN** a request `OPTIONS /api/karakeep/api/trpc/users.whoami?batch=1` arrives with `Origin: chrome-extension://abc` and `Access-Control-Request-Headers: authorization,content-type`
- **THEN** the server responds 204 with `Access-Control-Allow-Origin: *` and allows the headers `authorization,content-type`

#### Scenario: Preflight for an update
- **WHEN** a request `OPTIONS /api/linkwarden/api/v1/links/7` arrives with `Access-Control-Request-Method: PUT`
- **THEN** the server responds 204 and allows the methods `GET, POST, PUT, DELETE`

### Requirement: Token subtree scope
A request authenticated with an app token SHALL only read and change nodes in the subtree of the token's root folder: the root and everything below it. Nodes outside it are treated as if they didn't exist. A node can't be created in, moved to, or moved from a folder outside the subtree, and the root folder itself can't be deleted, moved or renamed through the token. This applies to every procedure and endpoint under `/api/karakeep/` and `/api/linkwarden/`.

#### Scenario: Node outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the request is answered as if the link didn't exist

#### Scenario: Whole tree
- **WHEN** a token is rooted at the top-level folder
- **THEN** it can reach every node of the user's tree
