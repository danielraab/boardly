# Spec Delta

## MODIFIED Requirements

### Requirement: App token properties
An app token SHALL belong to one user and have a name, an app, a root folder, a note scope, a creation date, a last-used date (empty until first use), an optional expiry date and an optional revocation date. The name is 1 to 100 characters after trimming. The app is `karakeep` or `linkwarden`; other apps are added by later changes. The root folder is any folder of the user's favourites tree, including the top-level folder. The note scope of a `karakeep` token is either the top level (`all`: every note of the user) or a root category (`category`: one category of the user with all its subcategories); a `linkwarden` token has no note access (`none`). A token's secret starts with `bdly_`, is followed by at least 32 random characters, and is never stored in plain text. The first 12 characters of the secret are the token's public prefix, which identifies it in the list.

#### Scenario: Secret format
- **WHEN** a token is created
- **THEN** its secret starts with `bdly_` and its prefix equals the first 12 characters of the secret

#### Scenario: Linkwarden token
- **WHEN** the user creates a token for the app `linkwarden`
- **THEN** the server responds 201 and the token is listed with the app `linkwarden` and the note scope `none`

#### Scenario: Existing Karakeep token
- **WHEN** a Karakeep token was created before tokens had a note scope
- **THEN** it has the note scope `all`

### Requirement: Create an app token
`POST /api/app-tokens` with `{"name", "app", "rootId", "rootCategoryId", "expiresAt"}` SHALL create a token for the signed-in user and respond 201 with the token and its `secret`. `rootCategoryId` is the ID of a category of the user for the note scope `category`, or `null` or missing for the top level; it is only accepted for `karakeep` tokens. `expiresAt` is an RFC 3339 time in the future or `null` for no expiry. The secret is returned only in this response and can never be read again. Errors respond 400 with `invalid_name` (name empty or too long after trimming), `invalid_app` (unknown app), `invalid_root` (not a folder of the user's tree), `invalid_root_category` (not a category of the user, or given for a `linkwarden` token) or `invalid_expiry` (not a valid time, or not in the future).

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

#### Scenario: Expiry in the past
- **WHEN** `expiresAt` is yesterday
- **THEN** the server responds 400 with `invalid_expiry`

### Requirement: List app tokens
`GET /api/app-tokens` SHALL return all of the signed-in user's tokens, newest first, each with `id`, `name`, `app`, `prefix`, `rootId` (`null` if the root folder was deleted), `noteScope` (`all`, `category` or `none`), `rootCategoryId` (the root category's ID for the note scope `category`, `null` if that category was deleted, and `null` for the other scopes), `createdAt`, `lastUsedAt`, `expiresAt`, `revokedAt` and `status`. It SHALL never return secrets or their hashes.

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
The "Connected apps" / "Verbundene Apps" tab of the settings page SHALL list the user's tokens with name, app, root folder path (or "Deleted folder" / "Gelöschter Ordner"), for Karakeep tokens the root category path (or "All notes" / "Alle Notizen" for the top level, or "Deleted category" / "Gelöschte Kategorie"), prefix, creation date, last-used date (or "Never used" / "Nie verwendet"), expiry date (or "No expiry" / "Läuft nicht ab") and a status badge "Active" / "Aktiv", "Expired" / "Abgelaufen" or "Revoked" / "Widerrufen". Each token SHALL show the official logo of its app, in its original colors, in front of the token's details. The logo is decorative: the app name stays in the text, and the logo has no text alternative of its own. The logos SHALL be served by Boardly itself, not loaded from a third-party site. Where an app's logo comes in a light and a dark variant, the variant follows the theme mode (see "Images follow the theme mode" in `theming`). Active tokens have an "Edit" / "Bearbeiten" action, which opens the edit dialog, and a "Revoke" / "Widerrufen" action; expired and revoked tokens have a "Delete" / "Löschen" action. Revoke and delete ask for confirmation. With no tokens, the tab explains what connected apps are. A "Connect app" / "App verbinden" action opens the create dialog.

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

#### Scenario: Logo in a dark theme
- **WHEN** the theme `black` is active and the list has a Karakeep token
- **THEN** the token shows the dark variant of the Karakeep logo

#### Scenario: Logo without a dark variant
- **WHEN** the theme `black` is active and the list has a Linkwarden token
- **THEN** the token shows the Linkwarden logo, which has only one variant

### Requirement: Create dialog
The create dialog SHALL ask for a name, the app ("Karakeep", preselected, or "Linkwarden"), the root folder (picked from the favourites tree, folders only, including the top-level folder), for Karakeep only the root category (picked from the user's categories as a tree, or "All notes" / "Alle Notizen", preselected, with a hint that text bookmarks from the app are saved there as notes) and the expiry: "30 days", "90 days" (default), "1 year", "No expiry" or a custom date. A custom date must be in the future. After creating, the dialog SHALL show the secret once, with a copy button, the server address (also with a copy button) to enter in the app (`<BOARDLY_BASE_URL>/api/karakeep` for Karakeep, `<BOARDLY_BASE_URL>/api/linkwarden` for Linkwarden) and a clearly highlighted warning that the secret won't be shown again. After the dialog is closed, the secret can't be shown again.

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

#### Scenario: Root category only for Karakeep
- **WHEN** the user switches the app to "Linkwarden"
- **THEN** the root category is no longer shown, and the token is created without `rootCategoryId`

### Requirement: Token subtree scope
A request authenticated with an app token SHALL only read and change nodes in the subtree of the token's root folder: the root and everything below it, and notes in the token's note scope: every note of the user for `all`, the notes in the root category and in all its subcategories at every depth for `category`, and none for `none`. Nodes and notes outside it are treated as if they didn't exist. A node can't be created in, moved to, or moved from a folder outside the subtree, and the root folder itself can't be deleted, moved or renamed through the token. Notes created through the token go into the root category, or are uncategorized for `all`. This applies to every procedure and endpoint under `/api/karakeep/` and `/api/linkwarden/`.

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

## REMOVED Requirements

### Requirement: Tokens are immutable
**Reason**: The root folder and the root category of a connected app can be changed after creation (see "Edit an app token").
**Migration**: Clients use `PATCH /api/app-tokens/{id}`; name, app, expiry and secret still can't be changed.

## ADDED Requirements

### Requirement: Edit an app token
`PATCH /api/app-tokens/{id}` with `rootId` and/or `rootCategoryId` SHALL change the root folder and, for `karakeep` tokens, the note scope of an active token of the signed-in user and respond 200 with the token. `rootCategoryId` `null` sets the note scope `all`, and a category ID sets `category`. The same validation as on create applies (`invalid_root`, `invalid_root_category`). Any other field, including `name`, `app`, `expiresAt` and `secret`, responds 400 with `invalid_request` and changes nothing. A token that isn't active is rejected with 409 and `token_inactive`. An unknown ID or another user's token responds 404 with `not_found`. The token's secret, prefix, dates and status stay the same, and the next request with it uses the new scope.

#### Scenario: Move to another folder
- **WHEN** the user changes the root folder of their active token "Phone" from "Reading" to "Work"
- **THEN** the server responds 200, and the next `bookmarks.getBookmarks` with that token returns the links of "Work"

#### Scenario: Change the root category
- **WHEN** the user changes the root category of a Karakeep token from the top level to "Inbox"
- **THEN** the token has the note scope `category`, and new text bookmarks from the app go into "Inbox"

#### Scenario: Revoked token
- **WHEN** the user edits a revoked token
- **THEN** the server responds 409 with `token_inactive`, and the token is unchanged

#### Scenario: Rename
- **WHEN** a client sends `PATCH /api/app-tokens/{id}` with a `name`
- **THEN** the server responds 400 with `invalid_request`, and the token is unchanged

### Requirement: Edit dialog
The edit dialog SHALL show the token's name and app as text, and ask for the root folder (picked like in the create dialog, the current one preselected) and, for Karakeep tokens, the root category (picked like in the create dialog, the current one preselected). Saving sends only the changed values and closes the dialog; the list then shows the new folder and category. A failed save shows a translated error and keeps the dialog open. All text SHALL be translated (English and German).

#### Scenario: Change the folder
- **WHEN** the user opens "Edit" on the token "Phone", picks the folder "Work" and saves
- **THEN** the dialog closes and the list shows "Work" as the root folder of "Phone"

#### Scenario: Linkwarden token
- **WHEN** the user opens "Edit" on a Linkwarden token
- **THEN** the dialog shows only the root folder choice
