## MODIFIED Requirements

### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder (except `memos` tokens), a note scope, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep`, `linkwarden` or `memos`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder; a `memos` token has no root folder, because Memos has no bookmarks. The note scope of a `karakeep` or `memos` token is either the top level (`all`: every note of the user) or a root category (`category`: one category of the user with all its subcategories); a `linkwarden` token has no note access (`none`). A token's secret starts with `bdly_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `bdly_` and its prefix equals the first 12 characters of the secret

#### Scenario: Linkwarden token
- **WHEN** the user creates a token for the app `linkwarden`
- **THEN** the server responds 201 and the token is listed with the app `linkwarden` and the note scope `none`

#### Scenario: Memos token
- **WHEN** the user creates a token for the app `memos` without a root category
- **THEN** the server responds 201 and the token is listed with the app `memos`, no root folder and the note scope `all`

#### Scenario: Existing Karakeep token
- **WHEN** a Karakeep token was created before tokens had a note scope
- **THEN** it has the note scope `all`

### Requirement: Create an app token
`POST /api/app-tokens` with `{"name", "app", "rootId", "rootCategoryId", "expiresAt"}` SHALL create a token for the signed-in user and respond 201 with the token and its `secret`. `rootId` is required for `karakeep` and `linkwarden` tokens and SHALL be missing or `null` for `memos` tokens. `rootCategoryId` is the ID of a category of the user for the note scope `category`, or `null` or missing for the top level; it is only accepted for `karakeep` and `memos` tokens. `expiresAt` is an RFC 3339 time in the future or `null` for no expiry. The secret is returned only in this response and can never be read again. Errors respond 400 with `invalid_name` (name empty or too long after trimming), `invalid_app` (unknown app), `invalid_root` (not a folder of the user's tree, or given for a `memos` token), `invalid_root_category` (not a category of the user, or given for a `linkwarden` token) or `invalid_expiry` (not a valid time, or not in the future).

#### Scenario: Create
- **WHEN** the user creates a token named "Phone" for `karakeep` with their folder "Reading" and no expiry
- **THEN** the server responds 201 with the token, status `active`, `expiresAt` `null`, note scope `all` and a `secret`

#### Scenario: Create with a root category
- **WHEN** the user creates a `karakeep` token with their category "Inbox" as `rootCategoryId`
- **THEN** the token has the note scope `category` and `rootCategoryId` the ID of "Inbox"

#### Scenario: Root is a link
- **WHEN** `rootId` is the ID of a link node
- **THEN** the server responds 400 with `invalid_root` and no token is created

#### Scenario: Root of another user
- **WHEN** `rootId` is a folder of another user
- **THEN** the server responds 400 with `invalid_root`

#### Scenario: Root category of another user
- **WHEN** `rootCategoryId` is a category of another user
- **THEN** the server responds 400 with `invalid_root_category` and no token is created

#### Scenario: Root category for Linkwarden
- **WHEN** the user creates a `linkwarden` token with a `rootCategoryId`
- **THEN** the server responds 400 with `invalid_root_category`

#### Scenario: Memos token with a root category
- **WHEN** the user creates a `memos` token with their category "Inbox" as `rootCategoryId`
- **THEN** the token has the note scope `category` and `rootCategoryId` the ID of "Inbox"

#### Scenario: Root folder for Memos
- **WHEN** the user creates a `memos` token with a `rootId`
- **THEN** the server responds 400 with `invalid_root` and no token is created

#### Scenario: Expiry in the past
- **WHEN** `expiresAt` is yesterday
- **THEN** the server responds 400 with `invalid_expiry`

### Requirement: List app tokens
`GET /api/app-tokens` SHALL return all of the signed-in user's tokens, newest first, each with `id`, `name`, `app`, `prefix`, `rootId` (`null` if the root folder was deleted, and always `null` for `memos` tokens), `noteScope` (`all`, `category` or `none`), `rootCategoryId` (the root category's ID for the note scope `category`, `null` if that category was deleted, and `null` for the other scopes), `createdAt`, `lastUsedAt`, `expiresAt`, `revokedAt` and `status`. It SHALL never return secrets or their hashes.

#### Scenario: No secret in the list
- **WHEN** the user lists their tokens after creating one
- **THEN** the token is listed with its prefix and without its secret

#### Scenario: Other users' tokens
- **WHEN** two users each have a token
- **THEN** each user's list only contains their own token

#### Scenario: Deleted root category
- **WHEN** the root category of a token was deleted
- **THEN** the token is listed with `noteScope` `category`, `rootCategoryId` `null` and status `revoked`

### Requirement: Connected apps tab
The "Connected apps" / "Verbundene Apps" tab of the settings page SHALL list the user's tokens with name, app, for Karakeep and Linkwarden tokens the root folder path (or "Deleted folder" / "Gelöschter Ordner"), for Karakeep and Memos tokens the root category path (or "All notes" / "Alle Notizen" for the top level, or "Deleted category" / "Gelöschte Kategorie"), prefix, creation date, last-used date (or "Never used" / "Nie verwendet"), expiry date (or "No expiry" / "Läuft nicht ab") and a status badge "Active" / "Aktiv", "Expired" / "Abgelaufen" or "Revoked" / "Widerrufen". Each token SHALL show the official logo of its app, in its original colors, in front of the token's details. The logo is decorative: the app name stays in the text, and the logo has no text alternative of its own. The logos SHALL be served by Boardly itself, not loaded from a third-party site. Where an app's logo comes in a light and a dark variant, the variant follows the theme mode (see "Images follow the theme mode" in `theming`). Active tokens have an "Edit" / "Bearbeiten" action, which opens the edit dialog, and a "Revoke" / "Widerrufen" action; expired and revoked tokens have a "Delete" / "Löschen" action. Revoke and delete ask for confirmation. With no tokens, the tab explains what connected apps are. A "Connect app" / "App verbinden" action opens the create dialog.

#### Scenario: Empty
- **WHEN** a user without tokens opens the tab
- **THEN** it explains connected apps and shows "Connect app"

#### Scenario: Actions by status
- **WHEN** the list has one active and one revoked token
- **THEN** the active one offers only "Edit" and "Revoke", and the revoked one offers only "Delete"

#### Scenario: Cancel revoke
- **WHEN** the user clicks "Revoke" and cancels the confirmation
- **THEN** the token stays active

#### Scenario: Root category shown
- **WHEN** the list has a Karakeep token with the root category "Inbox" and one with the top level
- **THEN** the first shows the category "Inbox" and the second "All notes", and a Linkwarden token shows no category

#### Scenario: Logo per app
- **WHEN** the list has a Karakeep token and a Linkwarden token
- **THEN** the Karakeep token shows the Karakeep logo and the Linkwarden token shows the Linkwarden logo, and both still show their app name as text

#### Scenario: Memos token in the list
- **WHEN** the list has a Memos token with the root category "Inbox"
- **THEN** it shows the Memos logo, the app name "Memos", the category "Inbox" and no root folder

#### Scenario: Logo in a dark theme
- **WHEN** the theme `black` is active and the list has a Karakeep token
- **THEN** the token shows the dark variant of the Karakeep logo

#### Scenario: Logo without a dark variant
- **WHEN** the theme `black` is active and the list has a Linkwarden token
- **THEN** the token shows the Linkwarden logo, which has only one variant

### Requirement: Create dialog
The create dialog SHALL ask for a name, the app ("Karakeep", preselected, "Linkwarden" or "Memos", with the hint "for apps like MoeMemos" / "für Apps wie MoeMemos"), for Karakeep and Linkwarden the root folder (picked from the favourites tree, folders only, including the top-level folder), for Karakeep and Memos the root category (picked from the user's categories as a tree, or "All notes" / "Alle Notizen", preselected, with a hint that text bookmarks from Karakeep, or new memos from Memos, are saved there as notes) and the expiry: "30 days", "90 days" (default), "1 year", "No expiry" or a custom date. A custom date must be in the future. After creating, the dialog SHALL show the secret once, with a copy button, the server address (also with a copy button) to enter in the app (`<BOARDLY_BASE_URL>/api/karakeep` for Karakeep, `<BOARDLY_BASE_URL>/api/linkwarden` for Linkwarden, `<BOARDLY_BASE_URL>/api/memos` for Memos) and a clearly highlighted warning that the secret won't be shown again. After the dialog is closed, the secret can't be shown again.

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

#### Scenario: Memos without a folder
- **WHEN** the user switches the app to "Memos"
- **THEN** the root folder is no longer shown and the root category is, the token is created without `rootId`, and after creating the dialog shows the server address `<BOARDLY_BASE_URL>/api/memos`

#### Scenario: Root category only for Karakeep
- **WHEN** the user switches the app to "Linkwarden"
- **THEN** the root category is no longer shown, the root folder is, and the token is created without `rootCategoryId`

### Requirement: Bearer authentication for app endpoints
Requests to `/api/karakeep/`, `/api/linkwarden/` and `/api/memos/` SHALL be authenticated only by the header `Authorization: Bearer <secret>`. The session cookie SHALL NOT authenticate these requests and is ignored. The cross-site request check of the web app's API does not apply to these paths. Procedures and endpoints marked as public in this spec or in the app's spec work without a token; every other procedure or endpoint requires an active token for the app of its path: `karakeep` under `/api/karakeep/`, `linkwarden` under `/api/linkwarden/`, `memos` under `/api/memos/`. A missing, malformed, unknown, revoked or expired token, or a token for another app, SHALL fail such a procedure with an `UNAUTHORIZED` error entry (HTTP status 401), such a Linkwarden endpoint with 401 and `{"response": "You must be logged in."}`, and such a Memos endpoint with 401 and the Memos error `{"code": 16, "message": "Unauthenticated.", "details": []}`, with the same message in every case. A request authenticated by a token records the token's last use, at most once per minute.

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

#### Scenario: Memos endpoint with a Karakeep token
- **WHEN** MoeMemos calls `GET /api/memos/api/v1/auth/me` with an active Karakeep token
- **THEN** the server responds 401 with the Memos error code `16`

#### Scenario: Token for another app
- **WHEN** a client calls `GET /api/linkwarden/api/v1/users/me` with an active Karakeep token
- **THEN** the server responds 401 with `{"response": "You must be logged in."}`

### Requirement: CORS for app endpoints
Responses on `/api/karakeep/`, `/api/linkwarden/` and `/api/memos/` SHALL allow cross-origin requests from any origin, including browser-extension origins, without credentials: `Access-Control-Allow-Origin: *`. Preflight `OPTIONS` requests SHALL be answered with 204 before authentication, with the allowed methods `GET, POST, PUT, PATCH, DELETE` and the request headers the preflight asks for (`Access-Control-Request-Headers`), because the extensions can add user-configured headers. The web app's other `/api/` endpoints are unchanged.

#### Scenario: Preflight from an extension
- **WHEN** a request `OPTIONS /api/karakeep/api/trpc/users.whoami?batch=1` arrives with `Origin: chrome-extension://abc` and `Access-Control-Request-Headers: authorization,content-type`
- **THEN** the server responds 204 with `Access-Control-Allow-Origin: *` and allows the headers `authorization,content-type`

#### Scenario: Preflight for an update
- **WHEN** a request `OPTIONS /api/linkwarden/api/v1/links/7` arrives with `Access-Control-Request-Method: PUT`
- **THEN** the server responds 204 and allows the methods `GET, POST, PUT, PATCH, DELETE`

### Requirement: Token subtree scope
A request authenticated with an app token SHALL only read and change nodes in the subtree of the token's root folder: the root and everything below it, and notes in the token's note scope: every note of the user for `all`, the notes in the root category and in all its subcategories at every depth for `category`, and none for `none`. Nodes and notes outside it are treated as if they didn't exist. A node can't be created in, moved to, or moved from a folder outside the subtree, and the root folder itself can't be deleted, moved or renamed through the token. Notes created through the token go into the root category, or are uncategorized for `all`. A `memos` token reaches no nodes at all. This applies to every procedure and endpoint under `/api/karakeep/`, `/api/linkwarden/` and `/api/memos/`.

#### Scenario: Node outside the subtree
- **WHEN** a token rooted at "Reading" asks for a link in "Work"
- **THEN** the request is answered as if the link didn't exist

#### Scenario: Whole tree
- **WHEN** a token is rooted at the top-level folder
- **THEN** it can reach every node of the user's tree

#### Scenario: Note outside the root category
- **WHEN** a token with the root category "Inbox" asks for a note in "Work"
- **THEN** the request is answered as if the note didn't exist

#### Scenario: Note in a subcategory
- **WHEN** a token has the root category "Home" and a note is in "Home/Garden"
- **THEN** the token can read and change that note

#### Scenario: Uncategorized notes
- **WHEN** a token has the root category "Inbox" and the user has an uncategorized note
- **THEN** the token can't see that note, while a token with the note scope `all` can

### Requirement: Edit an app token
`PATCH /api/app-tokens/{id}` with `rootId` and/or `rootCategoryId` SHALL change the root folder of an active `karakeep` or `linkwarden` token and the note scope of an active `karakeep` or `memos` token of the signed-in user and respond 200 with the token. `rootCategoryId` `null` sets the note scope `all`, and a category ID sets `category`. The same validation as on create applies (`invalid_root`, also for any `rootId` other than `null` on a `memos` token, and `invalid_root_category`). Any other field, including `name`, `app`, `expiresAt` and `secret`, responds 400 with `invalid_request` and changes nothing. A token that isn't active is rejected with 409 and `token_inactive`. An unknown ID or another user's token responds 404 with `not_found`. The token's secret, prefix, dates and status stay the same, and the next request with it uses the new scope.

#### Scenario: Move to another folder
- **WHEN** the user changes the root folder of their active token "Phone" from "Reading" to "Work"
- **THEN** the server responds 200, and the next `bookmarks.getBookmarks` with that token returns the links of "Work"

#### Scenario: Change the root category
- **WHEN** the user changes the root category of a Karakeep token from the top level to "Inbox"
- **THEN** the token has the note scope `category`, and new text bookmarks from the app go into "Inbox"

#### Scenario: Change the category of a Memos token
- **WHEN** the user changes the root category of a Memos token from "Inbox" to the top level
- **THEN** the token has the note scope `all`, and the next memo list with it contains every note of the user

#### Scenario: Revoked token
- **WHEN** the user edits a revoked token
- **THEN** the server responds 409 with `token_inactive`, and the token is unchanged

#### Scenario: Rename
- **WHEN** a client sends `PATCH /api/app-tokens/{id}` with a `name`
- **THEN** the server responds 400 with `invalid_request`, and the token is unchanged

### Requirement: Edit dialog
The edit dialog SHALL show the token's name and app as text, and ask, for Karakeep and Linkwarden tokens, for the root folder (picked like in the create dialog, the current one preselected) and, for Karakeep and Memos tokens, for the root category (picked like in the create dialog, the current one preselected). Saving sends only the changed values and closes the dialog; the list then shows the new folder and category. A failed save shows a translated error and keeps the dialog open. All text SHALL be translated (English and German).

#### Scenario: Change the folder
- **WHEN** the user opens "Edit" on the token "Phone", picks the folder "Work" and saves
- **THEN** the dialog closes and the list shows "Work" as the root folder of "Phone"

#### Scenario: Linkwarden token
- **WHEN** the user opens "Edit" on a Linkwarden token
- **THEN** the dialog shows only the root folder choice

#### Scenario: Memos token
- **WHEN** the user opens "Edit" on a Memos token
- **THEN** the dialog shows only the root category choice
