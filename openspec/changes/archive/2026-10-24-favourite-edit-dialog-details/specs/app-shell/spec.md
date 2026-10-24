# Spec Delta

## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the app icon and the product name "Boardly" on the left, together linking to `/`, and on the right, in this order, the "Add link" button (only for signed-in users, see "Add link button"), the theme switcher, the language switcher and the user avatar. The app icon comes directly before the product name, is drawn in the theme's text colour with the rose star as its only other colour, and is hidden from assistive technology so the link's accessible name stays "Boardly". On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the product name, separated from it by a slash.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with the app icon, "Boardly", the theme switcher, the language switcher and the avatar

#### Scenario: Icon before the product name
- **WHEN** the header is rendered
- **THEN** the app icon is shown directly before "Boardly" inside the same link, and the link's accessible name is "Boardly"

#### Scenario: Icon follows the theme
- **WHEN** the visitor switches from the "white" theme to the "black" theme
- **THEN** the icon's outline changes from dark to light together with the header text, and the star stays rose

#### Scenario: Theme switcher placement
- **WHEN** the header is rendered
- **THEN** the theme switcher button comes directly before the language switcher button

#### Scenario: Header with switcher
- **WHEN** a user with the dashboards "Home" and "Work" opens `/<user-slug>/work`
- **THEN** the header shows "Boardly / Work", where "Work" opens the dashboard switcher

## ADDED Requirements

### Requirement: Add link button
When a user is signed in, the header SHALL show an icon button with a plus icon, the same size and corner radius as the other header buttons, directly before the theme switcher. Its accessible name and tooltip SHALL be "Add link" / "Link hinzufügen". It SHALL link to `/share`, the "Save link" page (see the `pwa` capability). Signed-out visitors and visitors whose session is still loading SHALL NOT see it.

#### Scenario: Open the save link page
- **WHEN** a signed-in user activates "Add link" in the header
- **THEN** the browser shows `/share` with an empty form

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/`
- **THEN** the header has no "Add link" button

#### Scenario: Placement
- **WHEN** a signed-in user's header is rendered
- **THEN** the "Add link" button comes directly before the theme switcher
