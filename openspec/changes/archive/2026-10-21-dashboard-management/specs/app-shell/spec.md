# Spec Delta

## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the product name "Famodash" on the left, linking to `/`, and the language switcher and user avatar on the right. On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the product name, separated from it by a slash.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with "Famodash", the language switcher and the avatar

#### Scenario: Header with switcher
- **WHEN** a user with the dashboards "Home" and "Work" opens `/<user-slug>/work`
- **THEN** the header shows "Famodash / Work", where "Work" opens the dashboard switcher

### Requirement: Not-found page
Any client-side path that matches no route SHALL show a translated not-found message inside the app shell, with a link back to `/`. Because a single path segment is a user address (redirected to `/`) and two segments are a dashboard address, this covers paths with three or more segments, and dashboard addresses that can't be shown (see the `dashboards` capability).

#### Scenario: Unknown path
- **WHEN** a visitor opens `/no/such/page`
- **THEN** the page shows the header and a not-found message with a link to `/`

#### Scenario: Single segment
- **WHEN** a visitor opens `/no-such-page`
- **THEN** the browser shows `/` (it is treated as a user address)

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Dashboards" / "Dashboards" link to `/dashboards`, a "Favourites" / "Favoriten" link to `/favourites`, a "Settings" / "Einstellungen" link to `/settings`, and a "Sign out" / "Abmelden" action.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Dashboards", "Favourites", "Settings" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Dashboards", "Favourites", "Settings" and "Sign out", with no empty name line

#### Scenario: Dashboards link
- **WHEN** the user activates "Dashboards"
- **THEN** the menu closes and the browser shows `/dashboards`

#### Scenario: Favourites link
- **WHEN** the user activates "Favourites"
- **THEN** the menu closes and the browser shows `/favourites`

#### Scenario: Settings link
- **WHEN** the user activates "Settings"
- **THEN** the menu closes and the browser shows `/settings`
