# Spec Delta

## MODIFIED Requirements

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Favourites" / "Favoriten" link to `/favourites`, a "Settings" / "Einstellungen" link to `/settings`, and a "Sign out" / "Abmelden" action.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Favourites", "Settings" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Favourites", "Settings" and "Sign out", with no empty name line

#### Scenario: Favourites link
- **WHEN** the user activates "Favourites"
- **THEN** the menu closes and the browser shows `/favourites`

#### Scenario: Settings link
- **WHEN** the user activates "Settings"
- **THEN** the menu closes and the browser shows `/settings`
