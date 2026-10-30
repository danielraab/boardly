## MODIFIED Requirements

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Dashboards" / "Dashboards" link to `/dashboards`, a "Favourites" / "Favoriten" link to `/favourites`, a "Notes" / "Notizen" link to `/notes`, an "Attachments" / "Anhänge" link to `/attachments`, a "Settings" / "Einstellungen" link to `/settings`, a divider, a "GitHub" link to `https://github.com/danielraab/boardly`, a divider, and a "Sign out" / "Abmelden" action.

The "GitHub" item SHALL show the GitHub mark and a visible external-link marker. It SHALL open the repository in a new tab without giving the new page access to the opener, and its accessible name SHALL say that it opens in a new tab ("GitHub (opens in new tab)" / "GitHub (öffnet in neuem Tab)"). The signed-out menu doesn't show it.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Dashboards", "Favourites", "Notes", "Attachments", "Settings", "GitHub" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Dashboards", "Favourites", "Notes", "Attachments", "Settings", "GitHub" and "Sign out", with no empty name line

#### Scenario: Dashboards link
- **WHEN** the user activates "Dashboards"
- **THEN** the menu closes and the browser shows `/dashboards`

#### Scenario: Favourites link
- **WHEN** the user activates "Favourites"
- **THEN** the menu closes and the browser shows `/favourites`

#### Scenario: Notes link
- **WHEN** the user activates "Notes"
- **THEN** the menu closes and the browser shows `/notes`

#### Scenario: Attachments link
- **WHEN** the user activates "Attachments"
- **THEN** the menu closes and the browser shows `/attachments`

#### Scenario: Settings link
- **WHEN** the user activates "Settings"
- **THEN** the menu closes and the browser shows `/settings`

#### Scenario: GitHub link
- **WHEN** the user activates "GitHub"
- **THEN** the menu closes, `https://github.com/danielraab/boardly` opens in a new tab and the current page stays as it was

#### Scenario: GitHub item is separated
- **WHEN** a signed-in user opens the user menu
- **THEN** dividers separate "GitHub" from "Settings" above and from "Sign out" below
