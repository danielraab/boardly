# Spec Delta

## ADDED Requirements

### Requirement: Link widget details button
For the owner, outside edit mode, a link widget whose source is a favourite that still exists SHALL show an info button to the right of the link, which opens the favourite's details (see the favourite-details spec). Activating it SHALL NOT follow the link. A link widget with the source "URL", or whose favourite was deleted, SHALL show no info button.

#### Scenario: Favourite link
- **WHEN** Ann views her dashboard with a link widget for the favourite "Mail" and activates its info button
- **THEN** the details of "Mail" open, and no new tab opens

#### Scenario: Plain URL
- **WHEN** a link widget has the source "URL"
- **THEN** it shows no info button

#### Scenario: Deleted favourite
- **WHEN** the favourite behind a link widget no longer exists
- **THEN** the widget shows no info button
