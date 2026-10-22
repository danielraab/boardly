# dashboard-editing Specification

## Purpose

How a dashboard's owner changes its layout on the page: edit mode, the draft with save and cancel, adding, editing, moving and deleting sections, and the layout save API.

## Requirements

### Requirement: Dashboard actions menu for owners
When the viewer owns the dashboard, the dashboard page SHALL show an icon-only button with three dots next to the dashboard title, with the accessible name "Dashboard actions" / "Dashboard-Aktionen". Clicking it opens a menu with "Edit layout" / "Layout bearbeiten" and "Settings" / "Einstellungen". "Edit layout" enters edit mode. "Settings" opens the dashboard dialog for this dashboard. This applies at `/<user-slug>/<dashboard-slug>` and at `/` when it shows the default dashboard. Visitors don't see the button. The button is not shown in edit mode.

#### Scenario: Owner
- **WHEN** Ann opens her dashboard `/ann/home`
- **THEN** a "Dashboard actions" button is shown next to the title, and no "Edit" button

#### Scenario: Open the menu
- **WHEN** Ann clicks "Dashboard actions" on `/ann/home`
- **THEN** a menu with "Edit layout" and "Settings" opens

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard `/ann/home`
- **THEN** no "Dashboard actions" button is shown

### Requirement: Edit mode in the URL
Edit mode SHALL be reflected in the URL by the search parameter `edit`. "Edit layout" adds it, and leaving edit mode removes it. Opening a dashboard URL with `edit` puts the owner in edit mode. For anyone who isn't the owner, the parameter is ignored and the dashboard shows in view mode.

#### Scenario: Enter edit mode
- **WHEN** Ann opens "Dashboard actions" on `/ann/home` and picks "Edit layout"
- **THEN** the URL is `/ann/home?edit=true` and the editing controls are shown

#### Scenario: Reload in edit mode
- **WHEN** Ann reloads `/ann/home?edit=true`
- **THEN** the page opens in edit mode with the saved layout

#### Scenario: Visitor with the parameter
- **WHEN** a visitor opens `/ann/home?edit=true` of a public dashboard
- **THEN** the dashboard shows in view mode without any editing controls

### Requirement: Edit toolbar
In edit mode, the place of the "Dashboard actions" button SHALL show a "Save and exit" / "Speichern und beenden" button and a cancel button with a ✕ icon and the accessible name "Cancel editing" / "Bearbeiten abbrechen".

#### Scenario: Toolbar
- **WHEN** Ann is in edit mode
- **THEN** "Save and exit" and the ✕ button are shown instead of "Dashboard actions"

### Requirement: Draft until saved
Changes in edit mode SHALL only change a local draft, and nothing is sent to the server until "Save and exit". "Save and exit" saves the whole draft. When the save succeeds, the page leaves edit mode and shows the saved layout. When the save fails, edit mode and the draft stay, and an error message is shown.

#### Scenario: Save
- **WHEN** Ann adds a section in edit mode and clicks "Save and exit"
- **THEN** the layout is saved, the URL no longer has `edit`, and the new section is shown, also after a reload

#### Scenario: Cancel without changes
- **WHEN** Ann enters edit mode and clicks ✕ without changing anything
- **THEN** edit mode ends without a confirmation

#### Scenario: Save fails
- **WHEN** the save request fails with a network error
- **THEN** Ann stays in edit mode with her draft and sees an error message

### Requirement: Confirm before losing a draft
When the draft differs from the saved layout, ✕, navigating to another page in the app and reloading or closing the browser tab SHALL ask for confirmation first. Confirming drops the draft. Declining keeps edit mode and the draft. Reloading and closing use the browser's own confirmation.

#### Scenario: Cancel with changes, confirmed
- **WHEN** Ann has unsaved changes, clicks ✕ and confirms that they should be discarded
- **THEN** edit mode ends and the saved layout is shown

#### Scenario: Cancel with changes, declined
- **WHEN** Ann has unsaved changes, clicks ✕ and declines
- **THEN** she stays in edit mode with her draft

#### Scenario: Navigate away
- **WHEN** Ann has unsaved changes and picks another dashboard in the header switcher
- **THEN** she is asked for confirmation before the other dashboard opens

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

### Requirement: Empty dashboard outside edit mode
A dashboard without sections SHALL show the empty-dashboard placeholder outside edit mode, with no add buttons. For the owner, the placeholder text points to "Edit layout".

#### Scenario: Owner sees an empty dashboard
- **WHEN** Ann opens her new, empty dashboard
- **THEN** she sees the empty placeholder telling her to use "Edit layout" to add sections, and no "Add section" button

#### Scenario: Visitor sees an empty dashboard
- **WHEN** a visitor opens an empty public dashboard
- **THEN** they see the empty placeholder without any hint about editing

### Requirement: Add and edit sections
"Add section" and a section's edit button SHALL open a section dialog with title, width (1 up to the dashboard's desktop column count), border and height (`auto` or a number of rows). A new section starts with no title, border on, height `auto` and the full desktop width. Confirming adds the section at the end of its parent or applies the changes to the draft. Cancelling leaves the draft unchanged.

#### Scenario: Add a nested section
- **WHEN** Ann clicks "Add section" inside the section "Work", sets the title "Tools" and confirms
- **THEN** "Tools" appears as the last child of "Work" in the draft

#### Scenario: Edit a section
- **WHEN** Ann opens the edit dialog of a section, turns off the border, sets the height to 4 rows and confirms
- **THEN** the section shows those settings in the draft

### Requirement: Delete sections
The section dialog SHALL offer a delete action. Deleting asks for confirmation, naming the section by its title (or as an untitled section) and saying that everything inside it is deleted too. Confirming removes the section and its contents from the draft.

#### Scenario: Delete with contents
- **WHEN** Ann deletes the section "Work", which contains two sections, and confirms
- **THEN** "Work" and both of its child sections are gone from the draft

#### Scenario: Delete cancelled
- **WHEN** Ann chooses delete and then cancels the confirmation
- **THEN** the section and its contents stay

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

### Requirement: Blocked drops beyond the nesting limit
A drop that would put any section deeper than level 5 SHALL be blocked. The draft stays unchanged, and a message says that sections can be nested at most 5 levels deep. The message is also announced to screen readers.

#### Scenario: Too deep
- **WHEN** Ann drags a section that has two levels of children into a section at level 4
- **THEN** the drop is blocked, nothing moves and the nesting message is shown

### Requirement: Layout save API
`PUT /api/dashboards/{id}/layout` SHALL replace the layout of one of the signed-in user's own dashboards with the submitted section tree and the layout version it was based on. A valid request responds 200 with the stored layout and the new layout version. Another user's dashboard ID gets 404 `not_found`, a request without a session gets 401, and an invalid tree gets 400 `invalid_layout` with nothing changed.

#### Scenario: Save
- **WHEN** Ann sends a valid tree with the current layout version for her dashboard
- **THEN** the server responds 200 with the stored tree and a higher layout version

#### Scenario: Foreign dashboard
- **WHEN** Ann sends a layout for Bob's dashboard ID
- **THEN** the server responds 404 with `not_found` and Bob's layout is unchanged

### Requirement: Conflicting saves
A save based on a layout version that is no longer current SHALL be rejected with 409 and `layout_conflict`, and the stored layout stays unchanged. The page then keeps edit mode and the draft, and says that the dashboard was changed elsewhere. It offers to reload, which drops the draft and loads the current layout. Changes to the dashboard's title, slug, flags or column settings don't change the layout version.

#### Scenario: Two tabs
- **WHEN** Ann saves a layout in one tab and then saves a different draft in a second tab that was opened before
- **THEN** the second save gets 409, and the second tab shows the conflict message with a reload action

#### Scenario: Rename doesn't conflict
- **WHEN** Ann renames the dashboard on `/dashboards` while editing its layout in another tab, and then saves the layout
- **THEN** the layout is saved

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
