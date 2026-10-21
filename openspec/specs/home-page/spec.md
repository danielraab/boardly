# home-page Specification

## Purpose

What a signed-in user sees at `/`: their default dashboard, or a prompt to create a first dashboard or to pick a default.

## Requirements

### Requirement: Home for signed-in users
When a user is signed in, `/` SHALL show their start page instead of the landing page, and the URL stays `/`. What the start page shows depends on the user's dashboards:
- **Default dashboard set**: the default dashboard, exactly as at its own address, including the dashboard switcher.
- **No dashboards**: a greeting with the display name, or the email when no name is set, and a "Create your first dashboard" action that opens the create dialog on `/dashboards`.
- **Dashboards but no default**: the greeting and a "Select your default dashboard" dropdown with the user's dashboards. Picking one makes it the default and shows it right away.

#### Scenario: Default dashboard
- **WHEN** Ann, whose default dashboard is "Home", opens `/`
- **THEN** the URL stays `/` and the page shows "Home" as at `/ann/home`, with no landing-page content

#### Scenario: No dashboards
- **WHEN** a signed-in user named "Ann" without dashboards opens `/`
- **THEN** the page greets "Ann" and offers "Create your first dashboard"

#### Scenario: Create first dashboard
- **WHEN** that user activates "Create your first dashboard"
- **THEN** the browser shows `/dashboards` with the create dialog open

#### Scenario: Greeting without name
- **WHEN** a signed-in user without a display name and without dashboards opens `/`
- **THEN** the page greets them by their email address

#### Scenario: Select default
- **WHEN** Ann has "Home" and "Work" but no default, opens `/` and picks "Work" in "Select your default dashboard"
- **THEN** "Work" becomes her default and `/` shows "Work"
