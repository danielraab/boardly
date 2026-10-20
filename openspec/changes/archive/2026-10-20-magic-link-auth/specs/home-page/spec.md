# Spec Delta

## Purpose

What a signed-in user sees at `/`. For now it's a placeholder that will later become the user's start page with dashboards.

## ADDED Requirements

### Requirement: Home for signed-in users
When a user is signed in, `/` SHALL show the home page instead of the landing page. It greets the user by display name, or by email when no name is set, and shows a translated placeholder saying more is coming.

#### Scenario: Signed in with name
- **WHEN** a signed-in user named "Ann" opens `/`
- **THEN** the page greets "Ann" and shows the placeholder, and no landing-page content is visible

#### Scenario: Signed in without name
- **WHEN** a signed-in user without a display name opens `/`
- **THEN** the page greets them by their email address
