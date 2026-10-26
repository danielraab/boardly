# Spec Delta

## MODIFIED Requirements

### Requirement: Explorer tree
The explorer SHALL show the children of its root folder as a tree, not the root folder itself, in the order of the favourites. A folder shows a folder icon and its name and can be expanded and collapsed. A link shows its favicon, or a globe icon when it has none, and its title, or its URL when it has no title. Activating a link opens its URL in a new tab. Long names are cut off with "…" at the widget's width, but every name keeps a minimum width of about 6rem (shorter names keep their own width); when deeply nested rows don't fit, the tree scrolls horizontally inside the widget instead of hiding names, and the info buttons never overlap a name. An empty root folder shows "This folder is empty" / "Dieser Ordner ist leer". The tree always shows the current favourites.

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
