## MODIFIED Requirements

### Requirement: Theme choice is remembered in the browser
An explicit choice made in the theme menu SHALL be stored in the browser's localStorage under `boardly.theme` and SHALL apply to later visits in the same browser. Choosing System SHALL store `system`. An unknown stored value SHALL be treated as `system`. If localStorage is unavailable, the choice SHALL still apply for the current page view.

#### Scenario: Choice survives a reload
- **WHEN** a visitor chooses Fjord and reloads the page
- **THEN** the Fjord theme is shown

#### Scenario: Invalid stored value
- **WHEN** localStorage contains `boardly.theme` = `neon`
- **THEN** the web app behaves as if the choice were `system`
