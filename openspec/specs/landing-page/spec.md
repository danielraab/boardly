# landing-page Specification

## Purpose

The public page at `/` that introduces Boardly to visitors who are not signed in. For now it shows placeholders for a product screenshot and feature descriptions.

## Requirements

### Requirement: Landing page for signed-out visitors
When no user is signed in, `/` SHALL show the landing page: a hero area with the app title (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default), a one-line translated tagline, a call to action that goes to `/login` and, next to it, a secondary "GitHub" link to the Boardly repository (see "GitHub links").

#### Scenario: Visitor opens root
- **WHEN** a signed-out visitor opens `/`
- **THEN** the page shows "Boardly", the tagline, a sign-in call to action and a "GitHub" link next to it

#### Scenario: Custom app title
- **WHEN** the app title is "Boardly Staging" and a signed-out visitor opens `/`
- **THEN** the hero shows "Boardly Staging"

#### Scenario: Call to action
- **WHEN** the visitor activates the call to action
- **THEN** the browser shows `/login`

#### Scenario: GitHub link in the hero
- **WHEN** the landing page is rendered
- **THEN** the hero's "GitHub" link points to `https://github.com/danielraab/boardly` and looks less prominent than the call to action

### Requirement: Screenshot placeholder
The landing page SHALL contain a screenshot placeholder area with a fixed aspect ratio, so a real screenshot can replace it later without changing the layout.

#### Scenario: Placeholder visible
- **WHEN** the landing page is rendered
- **THEN** it shows a screenshot placeholder area labelled as a placeholder

### Requirement: Feature placeholders
The landing page SHALL contain a list of feature placeholder cards, each with an icon, a title and a description. The content comes from a data list, so features can be added or replaced without changing the layout component.

#### Scenario: Feature cards rendered
- **WHEN** the landing page is rendered
- **THEN** it shows one card per entry in the feature list, with translated title and description

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
