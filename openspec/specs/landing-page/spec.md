# landing-page Specification

## Purpose

The public page at `/` that introduces Boardly to visitors who are not signed in. For now it shows placeholders for a product screenshot and feature descriptions.

## Requirements

### Requirement: Landing page for signed-out visitors
When no user is signed in, `/` SHALL show the landing page: a hero area with the product name, a one-line translated tagline and a call to action that goes to `/login`.

#### Scenario: Visitor opens root
- **WHEN** a signed-out visitor opens `/`
- **THEN** the page shows "Famodash", the tagline and a sign-in call to action

#### Scenario: Call to action
- **WHEN** the visitor activates the call to action
- **THEN** the browser shows `/login`

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
