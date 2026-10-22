# app-tokens Specification

## Purpose
App tokens let external apps such as the Karakeep browser extension and mobile app and the Linkwarden mobile app and browser extension connect to Boardly with a bearer API key. Each token is bound to one app and one folder in the user's favourites tree, and it only sees that folder's subtree.
## Requirements
### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep` or `linkwarden`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder. A token's secret starts with `bdly_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `bdly_` and its prefix equals the first 12 characters of the secret

#### Scenario: Linkwarden token
- **WHEN** the user creates a token for the app `linkwarden`
- **THEN** the server responds 201 and the token is listed with the app `linkwarden`

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
The "Connected apps" / "Verbundene Apps" tab of the settings page SHALL list the user's tokens with name, app, root folder path (or "Deleted folder" / "Gelöschter Ordner"), prefix, creation date, last-used date (or "Never used" / "Nie verwendet"), expiry date (or "No expiry" / "Läuft nicht ab") and a status badge "Active" / "Aktiv", "Expired" / "Abgelaufen" or "Revoked" / "Widerrufen". Each token SHALL show the official logo of its app, in its original colors, in front of the token's details. The logo is decorative: the app name stays in the text, and the logo has no text alternative of its own. The logos SHALL be served by Boardly itself, not loaded from a third-party site. Where an app's logo comes in a light and a dark variant, the variant follows the theme mode (see "Images follow the theme mode" in `theming`). Active tokens have a "Revoke" / "Widerrufen" action, and expired and revoked tokens have a "Delete" / "Löschen" action. Both actions ask for confirmation. With no tokens, the tab explains what connected apps are. A "Connect app" / "App verbinden" action opens the create dialog.

#### Scenario: Empty
- **WHEN** a user without tokens opens the tab
- **THEN** it explains connected apps and shows "Connect app"

#### Scenario: Actions by status
- **WHEN** the list has one active and one revoked token
- **THEN** the active one offers only "Revoke" and the revoked one offers only "Delete"

#### Scenario: Cancel revoke
- **WHEN** the user clicks "Revoke" and cancels the confirmation
- **THEN** the token stays active

#### Scenario: Logo per app
- **WHEN** the list has a Karakeep token and a Linkwarden token
- **THEN** the Karakeep token shows the Karakeep logo and the Linkwarden token shows the Linkwarden logo, and both still show their app name as text

#### Scenario: Logo in a dark theme
- **WHEN** the theme `black` is active and the list has a Karakeep token
- **THEN** the token shows the dark variant of the Karakeep logo

#### Scenario: Logo without a dark variant
- **WHEN** the theme `black` is active and the list has a Linkwarden token
- **THEN** the token shows the Linkwarden logo, which has only one variant

### Requirement: Create dialog
The create dialog SHALL ask for a name, the app ("Karakeep", preselected, or "Linkwarden"), the root folder (picked from the favourites tree, folders only, including the top-level folder) and the expiry: "30 days", "90 days" (default), "1 year", "No expiry" or a custom date. A custom date must be in the future. After creating, the dialog SHALL show the secret once, with a copy button, the server address (also with a copy button) to enter in the app (`<BOARDLY_BASE_URL>/api/karakeep` for Karakeep, `<BOARDLY_BASE_URL>/api/linkwarden` for Linkwarden) and a clearly highlighted warning that the secret won't be shown again. After the dialog is closed, the secret can't be shown again.

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
- **THEN** the dialog shows the server address `<BOARDLY_BASE_URL>/api/linkwarden`

### Requirement: Karakeep tRPC transport
The Karakeep browser extension and mobile app talk tRPC (version 11, `httpBatchLink`, `superjson` transformer) to `<server address>/api/trpc`. Boardly SHALL answer these calls at `/api/karakeep/api/trpc/<procedures>?batch=1`, where `<procedures>` is a comma-separated list of procedure names such as `apiKeys.validate`. Queries are sent with `GET` and their inputs in the `input` query parameter; mutations are sent with `POST` and their inputs in the JSON body. In both cases the inputs are a JSON object keyed by the call's index (`"0"`, `"1"`, …), each value a superjson envelope `{"json": <value>, "meta"?: …}`; a missing key means no input. Dates in an input arrive as ISO 8601 strings in `json`, and Boardly SHALL accept them as such; `meta` is not needed to read an input. The response SHALL be a JSON array with one entry per procedure, in the request's order: `{"result": {"data": {"json": <output>}}}` on success, or `{"error": {"json": {"message", "code", "data": {"code", "httpStatus", "path"}}}}` on failure, where `code` is the JSON-RPC number of the tRPC error code (for example `-32001` for `UNAUTHORIZED`, `-32004` for `NOT_FOUND`). When an output contains dates or maps, the result SHALL also carry superjson's `meta` (`{"json": …, "meta": {"values": {<path>: ["Date"] | ["map"]}}}`), with dates as ISO 8601 strings and maps as arrays of `[key, value]` pairs in `json`, so the clients get real `Date` and `Map` values. The HTTP status SHALL be the common status of all entries, or 207 when they differ. Only the procedures named in this and later changes are implemented; any other procedure answers with a `NOT_FOUND` error entry. A query sent with `POST`, or a mutation sent with `GET`, answers with a `METHOD_NOT_SUPPORTED` error entry. A request without `batch=1`, or whose `input` isn't valid JSON, answers 400 with a single `BAD_REQUEST` error entry.

#### Scenario: Batched call
- **WHEN** a client sends `GET /api/karakeep/api/trpc/users.whoami,config.clientConfig?batch=1&input={}` with a valid token
- **THEN** the server responds 200 with an array of two `result` entries, the user first and the configuration second

#### Scenario: Unknown procedure
- **WHEN** a client calls `bookmarks.summarizeBookmark`
- **THEN** its entry is an error with `data.code` `NOT_FOUND`, `code` `-32004` and `data.path` `bookmarks.summarizeBookmark`

#### Scenario: Mixed results
- **WHEN** one call of a batch succeeds and another fails with `NOT_FOUND`
- **THEN** the server responds 207 and each entry carries its own result or error

#### Scenario: Date in an output
- **WHEN** a client calls `bookmarks.getBookmark` for a link created on 1 March 2026 at 12:00 UTC
- **THEN** the result's `json.createdAt` is `"2026-03-01T12:00:00.000Z"` and `meta.values` maps `createdAt` to `["Date"]`

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

### Requirement: Karakeep login procedures
Boardly SHALL implement the procedures the extension and the mobile app need to log in:

- `apiKeys.validate` (public mutation, input `{"apiKey"}`): succeeds with `{"success": true}` when the API key is an active Karakeep token, and fails with `UNAUTHORIZED` otherwise. A successful call records the token's last use.
- `apiKeys.exchange` (public mutation, email and password login): always fails with `UNAUTHORIZED` and the message "Boardly has no passwords. Log in with an API key from Settings → Connected apps.", because Boardly has no passwords.
- `users.whoami` (query): the token owner's `id`, `name` (the display name, or `null`), `email`, `image` `null` and `localUser` `true`.
- `config.clientConfig` (public query): a fixed configuration that tells the clients that signups and password login are disabled and that AI features, chat, semantic search, payments, video download and Turnstile are not available, with `publicUrl` and `publicApiUrl` built from `BOARDLY_BASE_URL` and `disableNewReleaseCheck` `true`.

#### Scenario: Log in with an API key
- **WHEN** the extension calls `apiKeys.validate` with an active Karakeep token as `apiKey` and without an `Authorization` header
- **THEN** the call succeeds with `{"success": true}`

#### Scenario: Log in with a revoked key
- **WHEN** `apiKeys.validate` is called with a revoked token
- **THEN** the call fails with `UNAUTHORIZED`

#### Scenario: Log in with a password
- **WHEN** a client calls `apiKeys.exchange` with any email and password
- **THEN** the call fails with `UNAUTHORIZED` and the message explains to use an API key

#### Scenario: Who am I
- **WHEN** Ann's client calls `users.whoami`
- **THEN** the result contains Ann's ID, name and email

### Requirement: Karakeep status endpoints
`GET /api/karakeep/api/health` SHALL respond 200 with `{"status": "ok", "message": …}` and `GET /api/karakeep/api/version` SHALL respond 200 with `{"version": "Boardly"}`. Both are public. The mobile app uses them for its connection indicator and its settings page. Any other path under `/api/karakeep/` outside `api/trpc` responds 404 in the API's JSON error format.

#### Scenario: Health
- **WHEN** the mobile app calls `GET /api/karakeep/api/health` without a token
- **THEN** the server responds 200 with `status` `ok`

#### Scenario: Former REST path
- **WHEN** a client calls `GET /api/karakeep/api/v1/users/me`
- **THEN** the server responds 404

### Requirement: Token subtree scope
A request authenticated with an app token SHALL only read and change nodes in the subtree of the token's root folder: the root and everything below it. Nodes outside it are treated as if they didn't exist. A node can't be created in, moved to, or moved from a folder outside the subtree, and the root folder itself can't be deleted, moved or renamed through the token. This applies to every procedure and endpoint under `/api/karakeep/` and `/api/linkwarden/`.

#### Scenario: Node outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the request is answered as if the link didn't exist

#### Scenario: Whole tree
- **WHEN** a token is rooted at the top-level folder
- **THEN** it can reach every node of the user's tree

