# Spec Delta

## MODIFIED Requirements

### Requirement: Editing controls on sections
In edit mode, every section and every content widget SHALL show a drag handle and an edit button. A section with a border keeps its normal border, and a section without a border shows a dashed outline in its place, so that it stays visible. A content widget always shows a dashed outline. The top level shows an "Add section" / "Abschnitt hinzufügen" button after the last section. Every section at levels 1 to 4 shows the same button after its last child. Sections at level 5 don't show it. Every section at every level shows an "Add widget" / "Widget hinzufügen" button next to its "Add section" button, or in its place at level 5, as long as at least one widget kind is available. The top level has no "Add widget" button. Outside edit mode, none of these controls are shown.

#### Scenario: Controls
- **WHEN** Ann enters edit mode on a dashboard with one borderless section
- **THEN** the section shows a dashed outline, a drag handle, an edit button and the "Add section" and "Add widget" buttons inside it, and there is an "Add section" button below it at the top level

#### Scenario: Section with a border
- **WHEN** Ann enters edit mode on a dashboard with a section that has a border
- **THEN** the section shows its normal border and no dashed outline

#### Scenario: Level 5
- **WHEN** a section is at level 5
- **THEN** it has an "Add widget" button and no "Add section" button

#### Scenario: Empty dashboard in edit mode
- **WHEN** Ann enters edit mode on a new dashboard
- **THEN** only the top-level "Add section" button is shown

### Requirement: Move sections by dragging
In edit mode, sections and content widgets SHALL be movable by their drag handle, with the pointer or with the keyboard. A section can be dropped before or after another widget in any parent, at the top level, or into an empty section. A content widget can be dropped before or after another widget inside a section, or into an empty section, but not at the top level. Nothing can be dropped into a content widget. Moving a section moves its contents with it. A section can't be dropped into itself or into one of its own descendants.

#### Scenario: Reorder
- **WHEN** Ann drags the second top-level section in front of the first
- **THEN** the order of the two is swapped in the draft

#### Scenario: Move into another section
- **WHEN** Ann drags a top-level section into the empty section "Work"
- **THEN** it becomes the child of "Work", with its contents

#### Scenario: Move a widget to another section
- **WHEN** Ann drags a content widget from the section "Work" into the section "Home"
- **THEN** the widget is a child of "Home" in the draft

#### Scenario: Widget to the top level
- **WHEN** Ann drags a content widget to the top level
- **THEN** the drop is not possible and the draft stays unchanged

#### Scenario: Keyboard
- **WHEN** Ann focuses a drag handle, presses Space, moves with the arrow keys and presses Space again
- **THEN** the widget is moved to the chosen place and the move is announced to screen readers

## ADDED Requirements

### Requirement: Add widgets
"Add widget" SHALL open a picker dialog that lists every available widget kind with its icon, name and a short description. Picking a kind closes the picker and opens that kind's settings dialog for a new widget. A new widget starts with the full width of the section and height `auto`, plus the defaults of its kind. Confirming the settings dialog adds the widget at the end of the section in the draft. Cancelling the picker or the settings dialog leaves the draft unchanged.

#### Scenario: Add a widget
- **WHEN** Ann clicks "Add widget" in the section "Work", picks a kind and confirms its settings dialog
- **THEN** the new widget is the last child of "Work" in the draft

#### Scenario: Cancel the settings
- **WHEN** Ann picks a kind in the picker and then cancels its settings dialog
- **THEN** no widget is added

### Requirement: Edit and delete widgets
A content widget's edit button SHALL open its kind's settings dialog with the widget's current values, including width and height. Confirming applies the changes to the draft, and cancelling leaves it unchanged. The dialog offers a delete action. Deleting asks for confirmation and then removes the widget from the draft.

#### Scenario: Edit a widget
- **WHEN** Ann opens the edit dialog of a content widget, changes its height to 3 rows and confirms
- **THEN** the widget is 3 rows tall in the draft

#### Scenario: Delete a widget
- **WHEN** Ann chooses delete in a widget's dialog and confirms
- **THEN** the widget is gone from the draft
