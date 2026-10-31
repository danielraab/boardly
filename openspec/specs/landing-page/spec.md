# landing-page Specification

## Purpose

The public page at `/` that introduces Boardly to signed-out visitors as an everyday helper: start page, bookmarks and notes in one self-hosted place, with a product sketch and a short feature overview.

## Requirements

### Requirement: Landing page for signed-out visitors
When no user is signed in, `/` SHALL show the landing page: a hero area with the app title (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default), a one-line translated tagline that presents Boardly as an everyday helper for a start page, bookmarks and notes (self-hosted), a call to action that goes to `/login` and, next to it, a secondary "GitHub" link to the Boardly repository (see "GitHub links").

#### Scenario: Visitor opens root
- **WHEN** a signed-out visitor opens `/`
- **THEN** the page shows "Boardly", the tagline, a sign-in call to action and a "GitHub" link next to it

#### Scenario: Tagline frames the everyday helper
- **WHEN** a signed-out visitor opens `/` with the English locale
- **THEN** the tagline presents Boardly as an everyday helper covering start page, bookmarks and notes, self-hosted

#### Scenario: Custom app title
- **WHEN** the app title is "Boardly Staging" and a signed-out visitor opens `/`
- **THEN** the hero shows "Boardly Staging"

#### Scenario: Call to action
- **WHEN** the visitor activates the call to action
- **THEN** the browser shows `/login`

#### Scenario: GitHub link in the hero
- **WHEN** the landing page is rendered
- **THEN** the hero's "GitHub" link points to `https://github.com/danielraab/boardly` and looks less prominent than the call to action

### Requirement: Responsive layout
The landing page SHALL be usable at widths from 320px upwards without horizontal scrolling.

#### Scenario: Narrow viewport
- **WHEN** the landing page is shown at 320px width
- **THEN** all content is visible without horizontal scrolling and the feature cards stack in one column

### Requirement: Landing page footer
The landing page SHALL end with a footer below the feature list that shows "Open source" / "Open Source" followed by a "GitHub" link to the Boardly repository (see "GitHub links"). The footer belongs to the landing page only. Other pages don't show it.

#### Scenario: Footer on the landing page
- **WHEN** a signed-out visitor opens `/`
- **THEN** the page ends with "Open source · GitHub", where "GitHub" links to `https://github.com/danielraab/boardly`

#### Scenario: No footer elsewhere
- **WHEN** a signed-in user opens a dashboard
- **THEN** the page shows no landing page footer

### Requirement: GitHub links
Every GitHub link on the landing page SHALL point to `https://github.com/danielraab/boardly`, even when the app title is customised. Each one SHALL show the GitHub mark, which is hidden from assistive technology, and a visible external-link marker. It SHALL open in a new tab without giving the new page access to the opener. Its accessible name SHALL say that it opens in a new tab.

#### Scenario: Opens in a new tab
- **WHEN** the visitor activates a GitHub link on the landing page
- **THEN** the repository opens in a new tab and the landing page stays open

#### Scenario: Accessible name
- **WHEN** a screen reader reads a GitHub link on the landing page
- **THEN** its name is "GitHub (opens in new tab)" / "GitHub (öffnet in neuem Tab)"

#### Scenario: Custom app title keeps the link
- **WHEN** the app title is "Boardly Staging"
- **THEN** the GitHub links still point to `https://github.com/danielraab/boardly`

### Requirement: Product sketch
The landing page SHALL show a stylised product sketch of a dashboard in a fixed-aspect area below the hero. The sketch MUST depict Boardly's everyday surface: a favourites tree, links, a note with todos, and a media widget (stations). It MUST use the page's theme colours so it remains readable in every theme. It MUST NOT show a fixed mini-player bar, third-party product marks, or a label that the area is a placeholder. Decorative sketch content that is not meaningful text MAY be hidden from assistive technology; the figure SHALL have an accessible name that describes it as an illustration of a Boardly dashboard.

#### Scenario: Sketch visible
- **WHEN** the landing page is rendered
- **THEN** it shows a product sketch area with a fixed aspect ratio that is not labelled as a placeholder

#### Scenario: Everyday surface cues
- **WHEN** a visitor looks at the product sketch
- **THEN** the sketch shows a favourites tree, links, a note with todos, and a media widget

#### Scenario: Theme colours
- **WHEN** the visitor switches the colour theme
- **THEN** the sketch colours follow the active theme

#### Scenario: Accessible name
- **WHEN** a screen reader reaches the product sketch
- **THEN** it announces a short description of a Boardly dashboard illustration

### Requirement: Feature overview
The landing page SHALL contain a list of feature entries below the product sketch, each with an icon, a translated title and a translated description. The content comes from a data list so entries can be added or replaced without changing the layout component. The list SHALL present, in this order, dashboards, favourites and notes as Boardly's everyday capabilities, followed by connected apps (the Karakeep and Linkwarden browser extensions and mobile apps, and Memos apps) and internet radio. Titles SHALL name the benefit for the visitor.

#### Scenario: Feature entries rendered
- **WHEN** the landing page is rendered
- **THEN** it shows one entry per item in the feature list, with translated title and description

#### Scenario: Everyday capabilities first
- **WHEN** the landing page is rendered
- **THEN** the first three feature entries cover dashboards, favourites and notes

#### Scenario: Connected apps
- **WHEN** the landing page is rendered
- **THEN** a feature entry says that links and notes can be saved from the Karakeep and Linkwarden apps and browser extensions and from Memos apps

#### Scenario: Radio
- **WHEN** the landing page is rendered
- **THEN** a feature entry says that favourite radio stations can be played with one click
