## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the product name "Famodash" on the left, linking to `/`, and on the right, in this order, the theme switcher, the language switcher and the user avatar.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with "Famodash", the theme switcher, the language switcher and the avatar

#### Scenario: Theme switcher placement
- **WHEN** the header is rendered
- **THEN** the theme switcher button comes directly before the language switcher button
