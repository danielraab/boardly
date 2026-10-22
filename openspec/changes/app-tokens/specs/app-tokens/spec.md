# Spec Delta

## Purpose

App tokens let external apps such as Karakeep clients (and later Linkwarden clients) connect to Famodash with a bearer API key. Each token is bound to one app and one folder in the user's favourites tree, and it only sees that folder's subtree.

## ADDED Requirements

### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder. A token's secret starts with `famo_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `famo_` and its prefix equals the first 12 characters of the secret

### Requirement: App token status
A token's status SHALL be `revoked` when it has a revocation date, else `expired` when its expiry date has been reached, and `active` otherwise. Only active tokens authenticate requests.

#### Scenario: Expiry reached
- **WHEN** a token's expiry date has been reached and it was never revoked
- **THEN** its status is `expired` and requests with it are rejected

#### Scenario: Revoked after expiry
- **WHEN** a token that has expired is affected by a folder delete that revokes tokens
- **THEN** it is not revoked and its status stays `expired`

### Requirement: Create an app token
`POST /api/app-tokens` with `{"name", "app", "rootId", "expiresAt"}` SHALL create a token for the signed-in user and respond 201 with the token and its `secret`. `expiresAt` is an RFC 3339 time in the future or `null` for no expiry. The secret is returned only in this response and can never be read again. Errors respond 400 with `invalid_name` (name empty or too long after trimming), `invalid_app` (unknown app), `invalid_root` (not a folder of the user's tree) or `invalid_expiry` (not a valid time, or not in the future).

#### Scenario: Create
- **WHEN** the user creates a token named "Phone" for `karakeep` with their folder "Reading" and no expiry
- **THEN** the server responds 201 with the token, status `active`, `expiresAt` `null` and a `secret`

#### Scenario: Root is a link
- **WHEN** `rootId` is the ID of a link node
- **THEN** the server responds 400 with `invalid_root` and no token is created

#### Scenario: Root of another user
- **WHEN** `rootId` is a folder of another user
- **THEN** the server responds 400 with `invalid_root`

#### Scenario: Expiry in the past
- **WHEN** `expiresAt` is yesterday
- **THEN** the server responds 400 with `invalid_expiry`

### Requirement: List app tokens
`GET /api/app-tokens` SHALL return all of the signed-in user's tokens, newest first, each with `id`, `name`, `app`, `prefix`, `rootId` (`null` if the root folder was deleted), `createdAt`, `lastUsedAt`, `expiresAt`, `revokedAt` and `status`. It SHALL never return secrets or their hashes.

#### Scenario: No secret in the list
- **WHEN** the user lists their tokens after creating one
- **THEN** the token is listed with its prefix and without its secret

#### Scenario: Other users' tokens
- **WHEN** two users each have a token
- **THEN** each user's list only contains their own token

### Requirement: Tokens are immutable
A token's name, app, root folder and expiry date SHALL NOT be changeable after creation. The only changes are revoking it, recording its last use and the root becoming `null` when its folder is deleted.

#### Scenario: No edit
- **WHEN** a client sends `PATCH /api/app-tokens/{id}`
- **THEN** the server responds 405 and the token is unchanged

### Requirement: Revoke an app token
`POST /api/app-tokens/{id}/revoke` SHALL set the revocation date of an active token of the signed-in user and respond 200 with the token. Revoking is final. A token that isn't active is rejected with 409 and `token_inactive`. An unknown ID or another user's token responds 404 with `not_found`.

#### Scenario: Revoke
- **WHEN** the user revokes their active token "Phone"
- **THEN** its status becomes `revoked` and the next request with its secret is rejected with 401

#### Scenario: Revoke twice
- **WHEN** the user revokes a token that is already revoked
- **THEN** the server responds 409 with `token_inactive` and the revocation date is unchanged

### Requirement: Delete an app token
`DELETE /api/app-tokens/{id}` SHALL delete a revoked or expired token of the signed-in user and respond 204. An active token is rejected with 409 and `token_active`. An unknown ID or another user's token responds 404 with `not_found`.

#### Scenario: Delete active
- **WHEN** the user deletes an active token
- **THEN** the server responds 409 with `token_active` and the token still exists

#### Scenario: Delete revoked
- **WHEN** the user deletes a revoked token
- **THEN** the server responds 204 and the token is no longer listed

### Requirement: Connected apps tab
The "Connected apps" / "Verbundene Apps" tab of the settings page SHALL list the user's tokens with name, app, root folder path (or "Deleted folder" / "Gelöschter Ordner"), prefix, creation date, last-used date (or "Never used" / "Nie verwendet"), expiry date (or "No expiry" / "Läuft nicht ab") and a status badge "Active" / "Aktiv", "Expired" / "Abgelaufen" or "Revoked" / "Widerrufen". Active tokens have a "Revoke" / "Widerrufen" action, and expired and revoked tokens have a "Delete" / "Löschen" action. Both actions ask for confirmation. With no tokens, the tab explains what connected apps are. A "Connect app" / "App verbinden" action opens the create dialog.

#### Scenario: Empty
- **WHEN** a user without tokens opens the tab
- **THEN** it explains connected apps and shows "Connect app"

#### Scenario: Actions by status
- **WHEN** the list has one active and one revoked token
- **THEN** the active one offers only "Revoke" and the revoked one offers only "Delete"

#### Scenario: Cancel revoke
- **WHEN** the user clicks "Revoke" and cancels the confirmation
- **THEN** the token stays active

### Requirement: Create dialog
The create dialog SHALL ask for a name, the app (only "Karakeep" for now, preselected), the root folder (picked from the favourites tree, folders only, including the top-level folder) and the expiry: "30 days", "90 days" (default), "1 year", "No expiry" or a custom date. A custom date must be in the future. After creating, the dialog SHALL show the secret once, with a copy button, the server address to enter in the app (`<FAMODASH_BASE_URL>/api/karakeep`) and a note that the secret won't be shown again. After the dialog is closed, the secret can't be shown again.

#### Scenario: Show secret once
- **WHEN** the user creates a token and closes the dialog
- **THEN** the list shows the new token, and neither the list nor reopening anything shows its secret

#### Scenario: Copy
- **WHEN** the user clicks the copy button in the secret view
- **THEN** the secret is copied to the clipboard and the button confirms it

#### Scenario: Default expiry
- **WHEN** the user opens the create dialog
- **THEN** "90 days" is selected as the expiry

### Requirement: Bearer authentication for app endpoints
Requests to `/api/karakeep/` SHALL be authenticated only by the header `Authorization: Bearer <secret>`. The session cookie SHALL NOT authenticate these requests and is ignored. A missing, malformed, unknown, revoked or expired token, or a token for another app, SHALL be rejected with 401, error code `unauthenticated` and the same message in every case. A successful request records the token's last use, at most once per minute. The cross-site request check of the web app's API does not apply to these paths.

#### Scenario: Valid token
- **WHEN** a client sends `GET /api/karakeep/api/v1/users/me` with an active Karakeep token
- **THEN** the server responds 200

#### Scenario: Cookie only
- **WHEN** a signed-in browser sends `GET /api/karakeep/api/v1/users/me` with its session cookie and no `Authorization` header
- **THEN** the server responds 401

#### Scenario: Revoked token
- **WHEN** a client uses a revoked token
- **THEN** the server responds 401 with the same body as for an unknown token

#### Scenario: Last used
- **WHEN** a client makes its first request with a new token
- **THEN** the token's last-used date in the list is set to about that time

### Requirement: CORS for app endpoints
Responses on `/api/karakeep/` SHALL allow cross-origin requests from any origin, including browser-extension origins, without credentials: `Access-Control-Allow-Origin: *`, and preflight `OPTIONS` requests are answered with 204, the allowed methods `GET, POST, PUT, PATCH, DELETE` and the allowed headers `Authorization, Content-Type`. The web app's other `/api/` endpoints are unchanged.

#### Scenario: Preflight from an extension
- **WHEN** a request `OPTIONS /api/karakeep/api/v1/users/me` arrives with `Origin: chrome-extension://abc` and `Access-Control-Request-Headers: authorization`
- **THEN** the server responds 204 with `Access-Control-Allow-Origin: *` and allows the `Authorization` header

### Requirement: Karakeep user endpoint
`GET /api/karakeep/api/v1/users/me` SHALL respond 200 with the token owner's `id`, `name` (the display name, or `null`) and `email`. Clients use it to check the connection. Unknown paths under `/api/karakeep/` respond 404 in the API's JSON error format after authentication.

#### Scenario: Who am I
- **WHEN** Ann's Karakeep client calls `GET /api/karakeep/api/v1/users/me`
- **THEN** the response contains Ann's ID, name and email

### Requirement: Token subtree scope
A request authenticated with an app token SHALL only read and change nodes in the subtree of the token's root folder: the root and everything below it. Nodes outside it are treated as if they didn't exist. A node can't be created in, moved to, or moved from a folder outside the subtree, and the root folder itself can't be deleted, moved or renamed through the token. This applies to every endpoint under `/api/karakeep/`.

#### Scenario: Node outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the request is answered as if the link didn't exist

#### Scenario: Whole tree
- **WHEN** a token is rooted at the top-level folder
- **THEN** it can reach every node of the user's tree
