# Spec Delta

## MODIFIED Requirements

### Requirement: Global header
Every page SHALL show a header with the app icon and the app title (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default) on the left, together linking to `/`, and on the right, in this order, the "Add link" button (only for signed-in users, see "Add link button"), the theme switcher, the language switcher and the user avatar. The app icon comes directly before the app title, is drawn in the theme's text colour with the rose star as its only other colour, and is hidden from assistive technology so the link's accessible name stays the app title. On dashboard pages, the dashboard switcher (see the `dashboards` capability) appears right after the app title, separated from it by a slash.

#### Scenario: Header on every page
- **WHEN** a visitor opens `/`, `/login` or an unknown path
- **THEN** the page shows the header with the app icon, "Boardly", the theme switcher, the language switcher and the avatar

#### Scenario: Icon before the product name
- **WHEN** the header is rendered
- **THEN** the app icon is shown directly before "Boardly" inside the same link, and the link's accessible name is "Boardly"

#### Scenario: Custom app title
- **WHEN** the app title is "Boardly Staging" and the header is rendered
- **THEN** the header link shows "Boardly Staging" and its accessible name is "Boardly Staging"

#### Scenario: Icon follows the theme
- **WHEN** the visitor switches from the "white" theme to the "black" theme
- **THEN** the icon's outline changes from dark to light together with the header text, and the star stays rose

#### Scenario: Theme switcher placement
- **WHEN** the header is rendered
- **THEN** the theme switcher button comes directly before the language switcher button

#### Scenario: Header with switcher
- **WHEN** a user with the dashboards "Home" and "Work" opens `/<user-slug>/work`
- **THEN** the header shows "Boardly / Work", where "Work" opens the dashboard switcher

### Requirement: Document title
The document title (shown in the browser tab) SHALL be `<page title> - <app title>` on every page that has a page title, where the page title is the translated text of the page's main heading and the app title is the configured one (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default). On dashboard pages, including `/` when it shows the default dashboard, the page title SHALL be the dashboard's title. Pages without a page title of their own (the landing page, the signed-in start page without a default dashboard, and pages while they load) SHALL have the app title as their title. The title SHALL update when the visitor navigates, when the language changes and when the shown dashboard's title changes.

#### Scenario: App page
- **WHEN** a signed-in user opens `/settings` in English
- **THEN** the document title is "Settings - Boardly"

#### Scenario: Dashboard page
- **WHEN** a user opens `/<user-slug>/work` for the dashboard "Work"
- **THEN** the document title is "Work - Boardly"

#### Scenario: Default dashboard on the start page
- **WHEN** a signed-in user whose default dashboard is "Home" opens `/`
- **THEN** the document title is "Home - Boardly"

#### Scenario: Landing page
- **WHEN** a signed-out visitor opens `/`
- **THEN** the document title is "Boardly"

#### Scenario: Navigation
- **WHEN** a signed-in user on `/favourites` follows a link to `/dashboards`
- **THEN** the document title changes from "Favourites - Boardly" to "Dashboards - Boardly"

#### Scenario: Language change
- **WHEN** a user on `/favourites` switches the language from English to German
- **THEN** the document title changes to "Favoriten - Boardly"

#### Scenario: Not-found page
- **WHEN** a visitor opens `/no/such/page` in English
- **THEN** the document title is "Page not found - Boardly"

#### Scenario: Custom app title
- **WHEN** the app title is "Boardly Staging" and a signed-in user opens `/settings` in English
- **THEN** the document title is "Settings - Boardly Staging"

## ADDED Requirements

### Requirement: App title in translated texts
Translated texts that name the product SHALL show the configured app title instead of a fixed "Boardly", in every supported language. This covers the landing page's features heading, the text on the sign-in confirmation page and the message for a reserved user address.

#### Scenario: Default title
- **WHEN** a visitor opens the sign-in confirmation page in English and the app title is "Boardly"
- **THEN** the page reads "Confirm to sign in to Boardly in this browser."

#### Scenario: Custom title in German
- **WHEN** the app title is "Boardly Staging" and a user who chose German enters a reserved user address
- **THEN** the message reads "Diese Adresse ist für Boardly Staging reserviert. Bitte wähle eine andere."
