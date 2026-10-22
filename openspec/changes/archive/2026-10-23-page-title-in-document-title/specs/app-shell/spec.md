# Spec Delta

## ADDED Requirements

### Requirement: Document title
The document title (shown in the browser tab) SHALL be `<page title> - Boardly` on every page that has a page title, where the page title is the translated text of the page's main heading. On dashboard pages, including `/` when it shows the default dashboard, the page title SHALL be the dashboard's title. Pages without a page title of their own (the landing page, the signed-in start page without a default dashboard, and pages while they load) SHALL have the title "Boardly". The title SHALL update when the visitor navigates, when the language changes and when the shown dashboard's title changes.

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
