# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Edit button for owners`
- TO: `### Requirement: Dashboard actions menu for owners`

## MODIFIED Requirements

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

### Requirement: Empty dashboard outside edit mode
A dashboard without sections SHALL show the empty-dashboard placeholder outside edit mode, with no add buttons. For the owner, the placeholder text points to "Edit layout".

#### Scenario: Owner sees an empty dashboard
- **WHEN** Ann opens her new, empty dashboard
- **THEN** she sees the empty placeholder telling her to use "Edit layout" to add sections, and no "Add section" button

#### Scenario: Visitor sees an empty dashboard
- **WHEN** a visitor opens an empty public dashboard
- **THEN** they see the empty placeholder without any hint about editing
