## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the app icon and the product name "Boardly" on the left, together linking to `/`, and on the right, in this order, the theme switcher, the language switcher and the user avatar. The app icon comes directly before the product name, is drawn in the theme's text colour with the rose star as its only other colour, and is hidden from assistive technology so the link's accessible name stays "Boardly". On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the product name, separated from it by a slash.

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
