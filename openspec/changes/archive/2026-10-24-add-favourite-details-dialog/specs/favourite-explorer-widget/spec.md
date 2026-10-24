# Spec Delta

## ADDED Requirements

### Requirement: Explorer details buttons
For the owner, outside edit mode, every link row of an explorer SHALL show an info button at the right end of the row, which opens the link's details (see the favourite-details spec). The button SHALL always be visible, not only on hover or focus, and its icon SHALL be smaller than the row's text and muted. Activating it SHALL NOT follow the link. Folder rows SHALL have no info button.

#### Scenario: Link row
- **WHEN** Ann views her dashboard with an explorer for "Work" and activates the info button of "Grafana"
- **THEN** the details of "Grafana" open, and no new tab opens

#### Scenario: Folder row
- **WHEN** the explorer shows the folder "Docs"
- **THEN** the row of "Docs" has no info button

#### Scenario: Always visible
- **WHEN** Ann views the explorer on a touch device without hovering
- **THEN** every link row shows its info button
