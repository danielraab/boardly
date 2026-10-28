# Spec Delta

## MODIFIED Requirements

### Requirement: Content width
The header and the page content SHALL share one horizontally centred container that is as wide as the viewport minus the side padding, up to at most 1536 px. On wider viewports the container stays 1536 px wide and the remaining space is split evenly to its left and right. The header always uses the whole container. Each page limits its content to one of the page widths (see "Page widths") inside the container. On pages with the full width, the left edge of the header content and the left edge of the page content line up at every viewport width.

#### Scenario: Wide viewport
- **WHEN** a dashboard is viewed 1920 px wide
- **THEN** the header and the page content are each 1536 px wide and centred, with 192 px of empty space on each side

#### Scenario: Viewport narrower than the cap
- **WHEN** a dashboard is viewed 1280 px wide
- **THEN** the header and the page content span the whole viewport except for the side padding

#### Scenario: Header and content aligned
- **WHEN** a dashboard is viewed at any width
- **THEN** the app icon in the header and the dashboard title start at the same horizontal position

#### Scenario: Header on a narrower page
- **WHEN** the settings page is viewed 1920 px wide
- **THEN** the header is 1536 px wide and the page content is narrower and centred below it

## ADDED Requirements

### Requirement: Page widths
Every page inside the app shell SHALL use exactly one of these page widths for its content. Each is a horizontally centred column inside the container; on a viewport too narrow for it, the column takes the container's whole width minus the side padding.

| Width  | At most                        | Pages |
|--------|--------------------------------|-------|
| full   | the container's width (1536 px) | a dashboard (own, default dashboard on the home page, public), the notes overview |
| wide   | 1152 px                        | favourites, the dashboards list, the landing page |
| narrow | 768 px                         | settings, the full-page note editor, the not-found page, the home page when no default dashboard is shown |
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
