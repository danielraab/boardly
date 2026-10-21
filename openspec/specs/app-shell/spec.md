# app-shell Specification

## Purpose

Defines the global frame around every Famodash page: the header with the product name, the theme switcher, the language switcher and the user avatar with its menu.
## Requirements

### Requirement: Global header
Every page SHALL show a header with the product name "Famodash" on the left, linking to `/`, and on the right, in this order, the theme switcher, the language switcher and the user avatar. On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the product name, separated from it by a slash.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with "Famodash", the theme switcher, the language switcher and the avatar

#### Scenario: Theme switcher placement
- **WHEN** the header is rendered
- **THEN** the theme switcher button comes directly before the language switcher button

#### Scenario: Header with switcher
- **WHEN** a user with the dashboards "Home" and "Work" opens `/<user-slug>/work`
- **THEN** the header shows "Famodash / Work", where "Work" opens the dashboard switcher

### Requirement: Avatar shows a generic person icon
The avatar SHALL be a round button that shows a generic person icon. It SHALL have an accessible name that describes it as the user menu.

#### Scenario: Avatar rendering
- **WHEN** the header is rendered
- **THEN** the avatar button shows a person icon and has an accessible name such as "User menu" / "Benutzermenü"

### Requirement: User menu opens from the avatar
Activating the avatar SHALL open a menu anchored below it at the top right. The menu SHALL close on Escape, on a click outside it, and after one of its items is activated. It SHALL be fully usable with the keyboard.

#### Scenario: Open and close with the keyboard
- **WHEN** a visitor focuses the avatar, presses Enter and then presses Escape
- **THEN** the menu opens, then closes, and focus returns to the avatar

#### Scenario: Close on outside click
- **WHEN** the menu is open and the visitor clicks outside it
- **THEN** the menu closes

### Requirement: Signed-out menu content
When no user is signed in, the user menu SHALL contain a single "Sign in" / "Anmelden" action that goes to `/login`.

#### Scenario: Sign-in action
- **WHEN** a signed-out visitor opens the user menu and activates "Sign in"
- **THEN** the menu closes and the browser shows `/login`

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

### Requirement: Menu while session is loading
While the web app doesn't know yet whether a user is signed in, the avatar SHALL be shown but its menu SHALL contain neither the signed-out nor the signed-in actions.

#### Scenario: Slow session lookup
- **WHEN** the current-user request hasn't completed yet and the visitor opens the user menu
- **THEN** the menu shows a loading indicator and no "Sign in" or "Sign out" action
