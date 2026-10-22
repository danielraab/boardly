## ADDED Requirements

### Requirement: Add a favourite from the explorer
When the signed-in owner views their dashboard, an explorer whose root folder exists SHALL show a button with the accessible name "Add favourite" / "Favorit hinzufügen" directly left of the expand-all button, in the same size and style (in its place when there is no expand-all button). Activating it opens the "New link" dialog of the favourites, with the explorer's root folder chosen as the folder; another folder can be chosen. Saving creates the link like on `/favourites`, closes the dialog, and the explorer then shows the new link without reloading the page. The button is not shown in edit mode, to visitors, or when the root folder no longer exists.

#### Scenario: Add a link
- **WHEN** Ann views her dashboard with an explorer for "Work", activates "Add favourite", enters `https://example.com` and saves
- **THEN** the link is added at the end of "Work" and the explorer shows it

#### Scenario: Other folder
- **WHEN** Ann opens the dialog from the explorer for "Work" and chooses the folder "Private" before saving
- **THEN** the link is added to "Private"

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with an explorer shown on public dashboards
- **THEN** the explorer has no "Add favourite" button

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** the explorer has no "Add favourite" button

#### Scenario: Deleted root folder
- **WHEN** an explorer's root folder no longer exists
- **THEN** it shows no "Add favourite" button
