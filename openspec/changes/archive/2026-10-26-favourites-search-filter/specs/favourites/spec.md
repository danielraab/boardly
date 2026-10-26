# Spec Delta

## ADDED Requirements

### Requirement: Search favourites
`/favourites` SHALL have a search field ("Search favourites" / "Favoriten durchsuchen"). The text is split at whitespace into terms, and matching ignores case. A link matches when every term occurs in its title, its URL or one of its tag names, or in the name of one of its folders below the root. While the field has text, the tree SHALL show only matching links and the folders that contain them, and a folder whose name contains every term together with all its contents (subject to the tag filter). Every shown folder is expanded. Matching text in the shown names SHALL be highlighted. Pressing `/` outside a text field SHALL focus the search field.

#### Scenario: Find by title
- **WHEN** "Work / Tools" contains the link "Grafana" and the user types "graf"
- **THEN** the tree shows "Work", "Tools" and "Grafana", expanded, and no other node

#### Scenario: Find by URL
- **WHEN** a link titled "Dashboards" has the URL `https://grafana.example.com` and the user types "grafana"
- **THEN** the tree shows "Dashboards"

#### Scenario: Find by tag
- **WHEN** a link has the tag "monitoring" and the user types "monitor"
- **THEN** the tree shows that link

#### Scenario: Several terms
- **WHEN** the user types "work graf"
- **THEN** "Grafana" in "Work / Tools" is shown, and a link "Grafana" outside "Work" is not

#### Scenario: Folder name matches
- **WHEN** the user types "tools"
- **THEN** "Tools" is shown with all its contents

#### Scenario: Keyboard shortcut
- **WHEN** focus is not in a text field and the user presses `/`
- **THEN** the search field is focused

### Requirement: Filter favourites by tag
`/favourites` SHALL offer a tag filter that lists every tag on the user's links, sorted by name, with the number of links that have it. The user can choose several tags; the tree then shows only links that have all chosen tags, together with the folders that contain them, expanded. The chosen tags are shown as chips that can each be removed. Activating a tag on a link row SHALL add it to the filter. The tag filter and the search apply together.

#### Scenario: One tag
- **WHEN** the user chooses the tag "reading"
- **THEN** the tree shows only links with "reading" and their folders

#### Scenario: Several tags
- **WHEN** the user chooses "reading" and "later"
- **THEN** only links with both tags are shown

#### Scenario: Tag on a row
- **WHEN** no tag is chosen and the user activates the tag "reading" on a link row
- **THEN** "reading" is chosen in the tag filter

#### Scenario: Search and tag
- **WHEN** the tag "reading" is chosen and the user types "rust"
- **THEN** only links with "reading" that match "rust" are shown

### Requirement: Filter state and clearing
The search text and the chosen tags SHALL be kept in the page URL, so a reload or going back restores them. While a filter is active, a "Clear filters" / "Filter zurücksetzen" action SHALL clear both. When nothing matches, the page SHALL show "No favourites match" / "Keine passenden Favoriten" with the same action instead of the tree.

#### Scenario: Reload
- **WHEN** the user has typed "graf" and chosen "tools", and reloads the page
- **THEN** the search field shows "graf", "tools" is chosen, and the tree is filtered

#### Scenario: Nothing matches
- **WHEN** the user types a text no link matches
- **THEN** the page shows "No favourites match" and "Clear filters"

#### Scenario: Clear
- **WHEN** the user activates "Clear filters"
- **THEN** the search field is empty, no tag is chosen, and the whole tree is shown with the remembered collapse state

### Requirement: No moving while filtered
While a search or tag filter is active, nodes SHALL NOT be movable on `/favourites`, by pointer or keyboard, and the page SHALL show "Clear the filters to reorder." / "Setze die Filter zurück, um die Reihenfolge zu ändern." Editing, deleting and adding nodes stay possible.

#### Scenario: Drag handles off
- **WHEN** a filter is active
- **THEN** no row offers a move action, and the hint is shown

#### Scenario: Edit while filtered
- **WHEN** a filter is active and the user edits a shown link
- **THEN** the change is saved and the filter stays active

### Requirement: Expand and collapse all on the favourites page
`/favourites` SHALL have a button that expands all folders, or collapses all of them when any folder is expanded. Its label says which ("Expand all" / "Alle aufklappen", "Collapse all" / "Alle zuklappen").

#### Scenario: Collapse all
- **WHEN** some folders are expanded and the user activates "Collapse all"
- **THEN** every folder at every level is collapsed, and the button says "Expand all"

#### Scenario: Expand all
- **WHEN** all folders are collapsed and the user activates "Expand all"
- **THEN** every folder at every level is expanded

### Requirement: Remembered collapse state on the favourites page
On a first visit, every folder on `/favourites` SHALL be expanded. The folders the user expands or collapses, also with "Expand all" or "Collapse all", SHALL be remembered in their browser and restored when the page is opened again. Nothing about it is stored on the server. Expanding or collapsing while a filter is active SHALL NOT change the remembered state.

#### Scenario: Remembered
- **WHEN** the user collapses "Work" and reloads the page
- **THEN** "Work" is still collapsed

#### Scenario: Filter doesn't change it
- **WHEN** "Work" is collapsed, the user searches for a link inside it and then clears the filters
- **THEN** "Work" is collapsed again

### Requirement: Folder link counts
Every folder row on `/favourites` SHALL show the number of links it contains at all levels below it. While a filter is active, it SHALL show the number of shown links below it instead.

#### Scenario: Count
- **WHEN** "Work" contains two links and the folder "Tools" with three links
- **THEN** the row of "Work" shows 5 and the row of "Tools" shows 3

#### Scenario: Count while filtered
- **WHEN** a filter is active and one link inside "Work" matches
- **THEN** the row of "Work" shows 1

### Requirement: Link to a folder on the favourites page
`/favourites?folder=<id>` SHALL expand that folder and all folders above it, scroll its row into view and highlight it briefly. The expanded folders are remembered like other expansions. An unknown ID, or the ID of a link, is ignored.

#### Scenario: Open a nested folder
- **WHEN** "Work" and "Tools" are collapsed and the user opens `/favourites?folder=<id of Tools>`
- **THEN** "Work" and "Tools" are expanded, and the row of "Tools" is scrolled into view and highlighted

#### Scenario: Unknown folder
- **WHEN** the user opens `/favourites?folder=unknown`
- **THEN** the page shows the tree as usual
