# Spec Delta

## MODIFIED Requirements

### Requirement: Dashboard actions menu for owners
When the viewer owns the dashboard, the dashboard page SHALL show an icon-only button with three dots next to the dashboard title, with the accessible name "Dashboard actions" / "Dashboard-Aktionen". Clicking it opens a menu with "Edit layout" / "Layout bearbeiten", "Share" / "Teilen" and "Settings" / "Einstellungen". "Edit layout" enters edit mode. "Share" opens the share dialog for this dashboard (see the `sharing` capability). "Settings" opens the dashboard dialog for this dashboard. This applies at `/<user-slug>/<dashboard-slug>` and at `/` when it shows the default dashboard. Visitors don't see the button. The button is not shown in edit mode.

#### Scenario: Owner
- **WHEN** Ann opens her dashboard `/ann/home`
- **THEN** a "Dashboard actions" button is shown next to the title, and no "Edit" button

#### Scenario: Open the menu
- **WHEN** Ann clicks "Dashboard actions" on `/ann/home`
- **THEN** a menu with "Edit layout", "Share" and "Settings" opens

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard `/ann/home`
- **THEN** no "Dashboard actions" button is shown

#### Scenario: Viewer of a shared dashboard
- **WHEN** Bob opens `/ann/work`, which Ann shared with Bob
- **THEN** no "Dashboard actions" button is shown
