# favourite-explorer-widget Specification

## Purpose
A dashboard widget that shows the contents of one folder of the owner's favourites as an expandable file tree.

## Requirements

### Requirement: Explorer settings
A favourite explorer SHALL have a root folder, an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" flag (default off), besides the common width and height. The settings dialog SHALL offer every folder of the owner's favourites as the root folder, including the tree's root ("Favourites"), in a tree that can be searched by name, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder. A layout with a size other than `S`, `M` or `L`, a title longer than 200 characters, or without a root folder ID, is rejected with 400 and `invalid_layout`.

#### Scenario: Add an explorer
- **WHEN** Ann picks "Favourite explorer" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid size
- **WHEN** a layout is saved with an explorer of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Explorer tree
The explorer SHALL show the children of its root folder as a tree, not the root folder itself, in the order of the favourites. A folder shows a folder icon and its name and can be expanded and collapsed. A link shows its favicon, or a globe icon when it has none, and its title, or its URL when it has no title. Activating a link opens its URL in a new tab. Long names are cut off with "…" at the widget's width, but every name keeps a minimum width of about 6rem (shorter names keep their own width) and has a maximum width of 24rem; a folder shows its full name as a tooltip, a link its URL as before; when deeply nested rows don't fit, the tree scrolls horizontally inside the widget instead of hiding names, and the info buttons never overlap a name. An empty root folder shows "This folder is empty" / "Dieser Ordner ist leer". The tree always shows the current favourites.

#### Scenario: Tree
- **WHEN** the root folder "Work" contains the folder "Tools" with the link "GitLab", and the link "Mail"
- **THEN** the explorer shows "Tools" with a folder icon and "Mail" with its favicon, and "GitLab" appears after expanding "Tools"

#### Scenario: Link without favicon
- **WHEN** a link in the explorer has no favicon
- **THEN** it shows a globe icon

#### Scenario: Favourites change
- **WHEN** Ann renames a link inside the root folder on `/favourites` and opens the dashboard again
- **THEN** the explorer shows the new name

#### Scenario: Deep nesting in a narrow widget
- **WHEN** an explorer one column wide shows ten expanded nested folders
- **THEN** every name is at least partly visible and the tree scrolls sideways inside the widget

### Requirement: Explorer title
An explorer with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit, with the expand-all button on the same line. Its heading level is that of a section's title at the same place in the layout. An explorer without a title shows no heading. When the signed-in owner views their dashboard outside edit mode and the root folder exists, the title SHALL be a link to `/favourites?folder=<root folder ID>` (to `/favourites` when the root folder is the tree's root), opening in the same tab. For visitors, in edit mode, or when the root folder no longer exists, the title is plain text.

#### Scenario: With title
- **WHEN** Ann gives an explorer the title "Work links"
- **THEN** the explorer shows the heading "Work links" above its tree

#### Scenario: Without title
- **WHEN** an explorer has no title
- **THEN** the tree starts at the top of the widget

#### Scenario: Owner follows the title
- **WHEN** Ann views her dashboard with an explorer titled "Work links" for the folder "Work" and activates the title
- **THEN** the browser shows `/favourites` with "Work" expanded, scrolled into view and highlighted

#### Scenario: Tree root
- **WHEN** the explorer's root folder is the tree's root and Ann activates its title
- **THEN** the browser shows `/favourites`

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with an explorer titled "Work links"
- **THEN** the title is not a link

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** the explorer's title is not a link

### Requirement: Explorer size
The size SHALL set the icon size, the font size and the padding of the rows: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** an explorer's size is changed from M to L
- **THEN** its icons, text and row padding become larger

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

### Requirement: Deleted root folder
When an explorer's root folder no longer exists, the explorer SHALL show "This folder no longer exists" / "Dieser Ordner existiert nicht mehr" to the owner. Its settings can still be opened to choose another folder, and the layout can still be saved.

#### Scenario: Folder deleted
- **WHEN** Ann deletes the folder an explorer shows and opens the dashboard
- **THEN** the explorer shows "This folder no longer exists"

#### Scenario: Save with a missing folder
- **WHEN** Ann edits another widget on a dashboard with such an explorer and saves
- **THEN** the layout is saved

### Requirement: Explorer on public dashboards
For anyone other than the owner, an explorer with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its tree. With the flag on, visitors get the explorer with the tree of its root folder and nothing else from the owner's favourites. An explorer whose root folder no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with an explorer whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the explorer or its folder's contents, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the explorer with the root folder's contents

#### Scenario: Only the root folder
- **WHEN** a visitor's response contains an explorer for the folder "Work"
- **THEN** it contains no node from outside "Work"

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
