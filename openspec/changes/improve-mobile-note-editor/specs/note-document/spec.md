# Spec Delta

## ADDED Requirements

### Requirement: Toolbar layout
The toolbar of the content editor SHALL stay on a single row. When its buttons don't fit, the row of buttons SHALL scroll sideways, while the save status at the end of the row (see "Save status stays visible" in the `notes` capability) stays in place. While the toolbar is stuck to the top, it SHALL have the background of the note, including the note's colour, so that the content scrolling under it isn't visible through it.

#### Scenario: Toolbar on a narrow viewport
- **WHEN** Ann opens a note in a viewport 360 px wide
- **THEN** the toolbar is one row, and swiping it sideways reaches "Add location" while the save status stays in place

#### Scenario: Content scrolls under the toolbar
- **WHEN** Ann scrolls a long yellow note until the toolbar sticks to the top
- **THEN** the toolbar has the yellow note colour and the text passing under it isn't visible
