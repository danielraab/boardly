# Spec Delta

## MODIFIED Requirements

### Requirement: Signed-in menu content
When a user is signed in, the user menu SHALL show, in this order: the display name (if set), the email address, a "Dashboards" / "Dashboards" link to `/dashboards`, a "Favourites" / "Favoriten" link to `/favourites`, a "Notes" / "Notizen" link to `/notes`, a "Media" / "Medien" link to `/media`, an "Attachments" / "Anhänge" link to `/attachments`, a "Settings" / "Einstellungen" link to `/settings`, a divider, a "GitHub" link to `https://github.com/danielraab/boardly`, a divider, and a "Sign out" / "Abmelden" action.

The "GitHub" item SHALL show the GitHub mark and a visible external-link marker. It SHALL open the repository in a new tab without giving the new page access to the opener, and its accessible name SHALL say that it opens in a new tab ("GitHub (opens in new tab)" / "GitHub (öffnet in neuem Tab)"). The signed-out menu doesn't show it.

#### Scenario: Menu with name
- **WHEN** a signed-in user named "Ann" with email `ann@example.org` opens the user menu
- **THEN** it shows "Ann", `ann@example.org`, "Dashboards", "Favourites", "Notes", "Media", "Attachments", "Settings", "GitHub" and "Sign out" in this order

#### Scenario: Menu without name
- **WHEN** a signed-in user without a display name opens the user menu
- **THEN** it shows the email address followed by "Dashboards", "Favourites", "Notes", "Media", "Attachments", "Settings", "GitHub" and "Sign out", with no empty name line

#### Scenario: Dashboards link
- **WHEN** the user activates "Dashboards"
- **THEN** the menu closes and the browser shows `/dashboards`

#### Scenario: Favourites link
- **WHEN** the user activates "Favourites"
- **THEN** the menu closes and the browser shows `/favourites`

#### Scenario: Notes link
- **WHEN** the user activates "Notes"
- **THEN** the menu closes and the browser shows `/notes`

#### Scenario: Media link
- **WHEN** the user activates "Media"
- **THEN** the menu closes and the browser shows `/media`

#### Scenario: Attachments link
- **WHEN** the user activates "Attachments"
- **THEN** the menu closes and the browser shows `/attachments`

#### Scenario: Settings link
- **WHEN** the user activates "Settings"
- **THEN** the menu closes and the browser shows `/settings`

#### Scenario: GitHub link
- **WHEN** the user activates "GitHub"
- **THEN** the menu closes, `https://github.com/danielraab/boardly` opens in a new tab and the current page stays as it was

#### Scenario: GitHub item is separated
- **WHEN** a signed-in user opens the user menu
- **THEN** dividers separate "GitHub" from "Settings" above and from "Sign out" below

### Requirement: Page widths
Every page inside the app shell SHALL use exactly one of these page widths for its content. Each is a horizontally centred column inside the container; on a viewport too narrow for it, the column takes the container's whole width minus the side padding.

| Width  | At most                        | Pages |
|--------|--------------------------------|-------|
| full   | the container's width (1536 px) | a dashboard (own, default dashboard on the home page, public), the notes overview |
| wide   | 1152 px                        | favourites, the dashboards list, the media page, the landing page |
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

#### Scenario: Media page on a wide screen
- **WHEN** the media page is viewed 1920 px wide
- **THEN** its content is at most 1152 px wide and centred in the viewport
