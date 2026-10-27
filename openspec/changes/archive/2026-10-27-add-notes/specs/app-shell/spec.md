# Spec Delta

## MODIFIED Requirements

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Dashboards" / "Dashboards" link to `/dashboards`, a "Favourites" / "Favoriten" link to `/favourites`, a "Notes" / "Notizen" link to `/notes`, a "Settings" / "Einstellungen" link to `/settings`, and a "Sign out" / "Abmelden" action.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Dashboards", "Favourites", "Notes", "Settings" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Dashboards", "Favourites", "Notes", "Settings" and "Sign out", with no empty name line

#### Scenario: Dashboards link
- **WHEN** the user activates "Dashboards"
- **THEN** the menu closes and the browser shows `/dashboards`

#### Scenario: Favourites link
- **WHEN** the user activates "Favourites"
- **THEN** the menu closes and the browser shows `/favourites`

#### Scenario: Notes link
- **WHEN** the user activates "Notes"
- **THEN** the menu closes and the browser shows `/notes`

#### Scenario: Settings link
- **WHEN** the user activates "Settings"
- **THEN** the menu closes and the browser shows `/settings`
