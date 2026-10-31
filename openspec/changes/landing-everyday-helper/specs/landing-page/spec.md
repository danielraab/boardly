# Spec Delta

## ADDED Requirements

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
The landing page SHALL contain a list of feature entries below the product sketch, each with an icon, a translated title and a translated description. The content comes from a data list so entries can be added or replaced without changing the layout component. The list SHALL present Dashboards, Favourites and Notes as Boardly's everyday capabilities. It SHALL NOT present connected-app compatibility (Karakeep, Linkwarden or Memos) as a landing feature entry.

#### Scenario: Feature entries rendered
- **WHEN** the landing page is rendered
- **THEN** it shows one entry per item in the feature list, with translated title and description

#### Scenario: Everyday capabilities
- **WHEN** the landing page is rendered
- **THEN** the feature list includes Dashboards, Favourites and Notes

#### Scenario: No connected-apps feature
- **WHEN** the landing page is rendered
- **THEN** the feature list has no entry whose title or description markets Karakeep, Linkwarden or Memos compatibility

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Screenshot placeholder
**Reason**: Replaced by the product sketch requirement, which shows a real overview of the everyday surface instead of a labelled placeholder.
**Migration**: Use "Product sketch". Remove the "Screenshot coming soon" / "Screenshot folgt" label and empty tile placeholder.

### Requirement: Feature placeholders
**Reason**: Replaced by the feature overview requirement with Boardly-native capabilities (Dashboards, Favourites, Notes) instead of placeholder cards that included connected apps.
**Migration**: Use "Feature overview". Update the feature data list and translations accordingly.
