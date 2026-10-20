# Spec Delta

## ADDED Requirements

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Profile" / "Profil" link to `/profile`, and a "Sign out" / "Abmelden" action.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Profile" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Profile" and "Sign out", with no empty name line

#### Scenario: Profile link
- **WHEN** the user activates "Profile"
- **THEN** the menu closes and the browser shows `/profile`

### Requirement: Menu while session is loading
While the web app doesn't know yet whether a user is signed in, the avatar SHALL be shown but its menu SHALL contain neither the signed-out nor the signed-in actions.

#### Scenario: Slow session lookup
- **WHEN** the current-user request hasn't completed yet and the visitor opens the user menu
- **THEN** the menu shows a loading indicator and no "Sign in" or "Sign out" action
