# Spec Delta

## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the app icon and the product name "Famodash" on the left, together linking to `/`, and on the right, in this order, the theme switcher, the language switcher and the user avatar. The app icon comes directly before the product name, is drawn in the theme's text colour with the rose star as its only other colour, and is hidden from assistive technology so the link's accessible name stays "Famodash". On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the product name, separated from it by a slash.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with the app icon, "Famodash", the theme switcher, the language switcher and the avatar

#### Scenario: Icon before the product name
- **WHEN** the header is rendered
- **THEN** the app icon is shown directly before "Famodash" inside the same link, and the link's accessible name is "Famodash"

#### Scenario: Icon follows the theme
- **WHEN** the visitor switches from the "white" theme to the "black" theme
- **THEN** the icon's outline changes from dark to light together with the header text, and the star stays rose

#### Scenario: Theme switcher placement
- **WHEN** the header is rendered
- **THEN** the theme switcher button comes directly before the language switcher button

#### Scenario: Header with switcher
- **WHEN** a user with the dashboards "Home" and "Work" opens `/<user-slug>/work`
- **THEN** the header shows "Famodash / Work", where "Work" opens the dashboard switcher

## ADDED Requirements

### Requirement: Favicon
Every page SHALL declare the app icon as its favicon, as an SVG file served by the app at `/favicon.svg`. The favicon SHALL have a transparent background. Its outline SHALL be dark when the browser uses a light colour scheme and light when it uses a dark colour scheme; the star SHALL be rose in both.

#### Scenario: Favicon is declared
- **WHEN** a visitor loads any page
- **THEN** the document head links `/favicon.svg` as an SVG icon

#### Scenario: Favicon is served
- **WHEN** a client requests `GET /favicon.svg`
- **THEN** the server responds with the SVG icon, not the web app's `index.html`

#### Scenario: Dark browser
- **WHEN** the browser uses a dark colour scheme
- **THEN** the favicon's outline is drawn in a light colour and remains visible on the dark tab bar
