# Spec Delta

## MODIFIED Requirements

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
