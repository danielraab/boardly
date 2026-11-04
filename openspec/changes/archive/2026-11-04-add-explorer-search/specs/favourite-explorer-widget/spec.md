## ADDED Requirements

### Requirement: Explorer search
Every explorer whose root folder exists and isn't empty SHALL show a search button with the accessible name "Search" / "Suchen", in the same size and style as the expand-all button. It sits left of the expand-all button and right of the "Add favourite" button when they are shown, and on the title's line when there is a title. It is shown to the owner, to visitors of public dashboards, and in edit mode. Activating it opens a search field with the accessible name "Search this folder" / "Diesen Ordner durchsuchen" directly above the tree and as wide as the tree, and moves focus into it. The button shows whether the field is open (`aria-expanded`).

While the field contains search terms, the tree SHALL show only what matches, live as the viewer types, without a request to the server. Matching follows the search of the favourites page and ignores case: the text is split at whitespace into terms, and a link is shown when every term occurs in its title, its URL, one of its tags or the name of a folder between the explorer's root folder and the link. A folder whose own name contains every term is shown with everything inside it. Other folders are shown when something inside them is shown. The order of the favourites is kept. Every shown folder is expanded, and the parts of the shown names that match a term are highlighted. When nothing matches, the explorer says "Nothing matches your search." / "Nichts passt zu deiner Suche." in place of the tree. Only whitespace counts as no search.

Activating the search button again, pressing Escape in the field, or activating the field's close button ("Close search" / "Suche schließen") SHALL close the field and clear the search, and the full tree is shown again. Searching SHALL NOT change which folders the viewer's browser remembers as expanded: after the search is cleared, the tree is expanded as it was before. The search text is not stored, neither in the browser nor on the server, and is empty after the page is reloaded.

#### Scenario: Open the search
- **WHEN** Ann activates the search button of an explorer for "Work"
- **THEN** a search field appears above the tree, it has focus, and the button reports itself as expanded

#### Scenario: Filter by link title
- **WHEN** "Work" contains the folder "Tools" with the links "GitLab" and "Jira", and the link "Mail", all folders are collapsed, and Ann types "git"
- **THEN** the explorer shows only "Tools", expanded, with "GitLab" inside it, and "Git" in "GitLab" is highlighted

#### Scenario: Several terms
- **WHEN** Ann types "tools jira"
- **THEN** only "Jira" is shown inside "Tools", because "tools" occurs in its folder path and "jira" in its title

#### Scenario: Folder name matches
- **WHEN** Ann types "tools"
- **THEN** "Tools" is shown, expanded, with both "GitLab" and "Jira"

#### Scenario: Match by URL or tag
- **WHEN** "Mail" has the URL `https://mail.example.com` and the tag "daily", and Ann types "example" or "daily"
- **THEN** "Mail" is shown

#### Scenario: No match
- **WHEN** Ann types "zzz"
- **THEN** the explorer shows "Nothing matches your search." and no rows

#### Scenario: Close restores the tree
- **WHEN** Ann had expanded only "Docs", searches for "git", and then presses Escape
- **THEN** the search field closes, the search is cleared, all rows are shown again, only "Docs" is expanded, and reloading the page still shows only "Docs" expanded

#### Scenario: Toggle with the button
- **WHEN** the search field is open and Ann activates the search button
- **THEN** the field closes and the search is cleared

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with an explorer shown on public dashboards and searches for "git"
- **THEN** the explorer shows the matching links

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** the explorer has a search button and its search filters the tree

#### Scenario: Not stored
- **WHEN** Ann searches for "git" and reloads the page
- **THEN** the search field is closed and the full tree is shown

#### Scenario: Empty or missing folder
- **WHEN** an explorer's root folder is empty or no longer exists
- **THEN** it shows no search button

## MODIFIED Requirements

### Requirement: Collapse state
Every folder in the explorer SHALL start collapsed. The folders the viewer expands or collapses SHALL be remembered in their browser for that widget and restored when the dashboard is opened again. Nothing about it is stored on the server. A small button in the widget expands all folders, or collapses all of them when any folder is expanded. Its accessible name says which ("Expand all" / "Alle aufklappen", "Collapse all" / "Alle zuklappen"). While the explorer's search contains search terms, this button SHALL be hidden.

#### Scenario: First visit
- **WHEN** someone opens a dashboard with an explorer for the first time in their browser
- **THEN** all its folders are collapsed

#### Scenario: Remembered
- **WHEN** Ann expands "Tools" and reloads the page
- **THEN** "Tools" is still expanded

#### Scenario: Expand all
- **WHEN** all folders are collapsed and Ann activates "Expand all"
- **THEN** every folder at every level is expanded, and the button now says "Collapse all"

#### Scenario: Hidden while searching
- **WHEN** Ann types "git" in the explorer's search
- **THEN** the explorer shows no "Expand all" or "Collapse all" button until the search is cleared

### Requirement: Add a favourite from the explorer
When the signed-in owner views their dashboard, an explorer whose root folder exists SHALL show a button with the accessible name "Add favourite" / "Favorit hinzufügen" in the same size and style as the expand-all button. It is the first of the explorer's buttons, left of the search button when that is shown, and left of the expand-all button otherwise; when neither is shown, it stands alone. Activating it opens the "New link" dialog of the favourites, with the explorer's root folder chosen as the folder; another folder can be chosen. Saving creates the link like on `/favourites`, closes the dialog, and the explorer then shows the new link without reloading the page. The button is not shown in edit mode, to visitors, or when the root folder no longer exists.

#### Scenario: Add a link
- **WHEN** Ann views her dashboard with an explorer for "Work", activates "Add favourite", enters `https://example.com` and saves
- **THEN** the link is added at the end of "Work" and the explorer shows it

#### Scenario: Other folder
- **WHEN** Ann opens the dialog from the explorer for "Work" and chooses the folder "Private" before saving
- **THEN** the link is added to "Private"

#### Scenario: Button order
- **WHEN** Ann views her dashboard with an explorer for "Work" that contains folders
- **THEN** its buttons are, from left to right, "Add favourite", "Search" and "Expand all"

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with an explorer shown on public dashboards
- **THEN** the explorer has no "Add favourite" button

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** the explorer has no "Add favourite" button

#### Scenario: Deleted root folder
- **WHEN** an explorer's root folder no longer exists
- **THEN** it shows no "Add favourite" button
