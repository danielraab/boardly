# Spec Delta

## ADDED Requirements

### Requirement: Global footer
Every page inside the app shell SHALL end with a slim footer below the page content, in the same container as the header. It is a `contentinfo` landmark, in small muted text. On pages shorter than the viewport, it sits at the bottom of the viewport. On longer pages, it comes after the content.

- On the left, "Imprint" / "Impressum" and "Privacy" / "Datenschutz" link to the configured legal pages (see the `legal-pages` capability), separated by a dot. A page that isn't configured has no link. A link to a page served from files opens it in the same tab. A link to an external URL opens it in a new tab without giving the new page access to the opener, shows a visible external-link marker, and has an accessible name that says it opens in a new tab.
- On the right, "Open source" / "Open Source" is followed by a "GitHub" link to `https://github.com/danielraab/boardly`. It shows the GitHub mark, hidden from assistive technology, and a visible external-link marker. It opens in a new tab without giving the new page access to the opener. Its accessible name is "GitHub (opens in new tab)" / "GitHub (öffnet in neuem Tab)", and it points to the repository even when the app title is customised.

At narrow widths, the two groups SHALL wrap onto separate lines instead of scrolling horizontally.

#### Scenario: Footer on every page
- **WHEN** a visitor opens `/`, `/login`, a dashboard, `/settings` or an unknown path
- **THEN** the page ends with the footer, and the footer shows "Open source · GitHub"

#### Scenario: Both legal pages from files
- **WHEN** the imprint and the privacy policy are served from files and a visitor views any page in German
- **THEN** the footer shows "Impressum · Datenschutz", linking to `/imprint` and `/privacy` in the same tab

#### Scenario: External privacy policy
- **WHEN** the privacy policy is `https://example.at/datenschutz`
- **THEN** the footer's "Privacy" link points there, opens in a new tab, shows the external-link marker and is named "Privacy (opens in new tab)"

#### Scenario: Nothing configured
- **WHEN** no legal page is configured
- **THEN** the footer shows only "Open source · GitHub"

#### Scenario: Short page
- **WHEN** the login page is viewed in a viewport taller than its content
- **THEN** the footer is at the bottom of the viewport

#### Scenario: Narrow viewport
- **WHEN** a page with both legal links is shown at 320 px width
- **THEN** the legal links and "Open source · GitHub" are on separate lines and nothing scrolls horizontally

## MODIFIED Requirements

### Requirement: Document title
The document title (shown in the browser tab) SHALL be `<page title> - <app title>` on every page that has a page title, where the page title is the translated text of the page's main heading and the app title is the configured one (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default). On dashboard pages, including `/` when it shows the default dashboard, the page title SHALL be the dashboard's title. On the legal pages, the page title SHALL be the page's translated name, "Imprint" / "Impressum" or "Privacy" / "Datenschutz", whatever headings the operator's content has. Pages without a page title of their own (the landing page, the signed-in start page without a default dashboard, and pages while they load) SHALL have the app title as their title. The title SHALL update when the visitor navigates, when the language changes and when the shown dashboard's title changes.

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

#### Scenario: Legal page
- **WHEN** a visitor opens `/privacy` in English and the content's first heading is "Data protection notice"
- **THEN** the document title is "Privacy - Boardly"

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

### Requirement: Page widths
Every page inside the app shell SHALL use exactly one of these page widths for its content. Each is a horizontally centred column inside the container; on a viewport too narrow for it, the column takes the container's whole width minus the side padding.

| Width  | At most                        | Pages |
|--------|--------------------------------|-------|
| full   | the container's width (1536 px) | a dashboard (own, default dashboard on the home page, public), the notes overview |
| wide   | 1152 px                        | favourites, the dashboards list, the media page, the landing page |
| narrow | 768 px                         | settings, the full-page note editor, the not-found page, the home page when no default dashboard is shown, the imprint and the privacy policy |
| focus  | 448 px                         | login, login verification, "Save link" |

Choosing a page width SHALL only set the page's outer width and horizontal position; the layout inside the page stays as it is.

#### Scenario: Settings on a wide screen
- **WHEN** the settings page is viewed 1920 px wide
- **THEN** its title, tabs and cards are at most 768 px wide and centred in the viewport

#### Scenario: Settings on a phone
- **WHEN** the settings page is viewed 375 px wide
- **THEN** its content spans the viewport minus the side padding, as before

#### Scenario: Favourites on a wide screen
- **WHEN** the favourites page is viewed 1920 px wide
- **THEN** its content is at most 1152 px wide and centred in the viewport

#### Scenario: Landing page on a wide screen
- **WHEN** a signed-out visitor views the landing page 1920 px wide
- **THEN** its content is at most 1152 px wide and centred in the viewport

#### Scenario: Notes overview on a wide screen
- **WHEN** the notes overview is viewed 1920 px wide
- **THEN** its content is 1536 px wide, like a dashboard

#### Scenario: Note editor on a wide screen
- **WHEN** a note is opened on its own page 1920 px wide
- **THEN** the editor is at most 768 px wide and centred in the viewport

#### Scenario: Login on a wide screen
- **WHEN** the login page is viewed 1920 px wide
- **THEN** its content is at most 448 px wide and centred in the viewport

#### Scenario: Media page on a wide screen
- **WHEN** the media page is viewed 1920 px wide
- **THEN** its content is at most 1152 px wide and centred in the viewport

#### Scenario: Legal page on a wide screen
- **WHEN** the imprint is viewed 1920 px wide
- **THEN** its content is at most 768 px wide and centred in the viewport
