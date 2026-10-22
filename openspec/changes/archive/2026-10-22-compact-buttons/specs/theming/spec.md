## ADDED Requirements

### Requirement: Compact buttons with rounded corners
All buttons SHALL use a compact, consistent size and SHALL have rounded corners instead of a fully rounded pill shape. Text buttons of the same kind (primary, secondary, destructive) SHALL have the same size on every page and in every dialog. Chips and badges are not buttons and MAY stay pill-shaped.

#### Scenario: Dialog and page buttons match
- **WHEN** a user sees the "Save" button in a dialog and the "Add passkey" button on the settings page
- **THEN** both have the same height, padding and font size

#### Scenario: No pill-shaped buttons
- **WHEN** any page or dialog is rendered
- **THEN** no button is fully rounded; its corners are rounded but its sides are straight
