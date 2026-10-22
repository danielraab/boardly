## MODIFIED Requirements

### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep` or `linkwarden`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder. A token's secret starts with `bdly_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `bdly_` and its prefix equals the first 12 characters of the secret

#### Scenario: Linkwarden token
- **WHEN** the user creates a token for the app `linkwarden`
- **THEN** the server responds 201 and the token is listed with the app `linkwarden`

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
