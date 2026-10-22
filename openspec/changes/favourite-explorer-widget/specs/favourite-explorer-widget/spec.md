# Spec Delta

## Purpose

A dashboard widget that shows the contents of one folder of the owner's favourites as an expandable file tree.

## ADDED Requirements

### Requirement: Explorer settings
A favourite explorer SHALL have a root folder, a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" flag (default off), besides the common width and height. The settings dialog SHALL offer every folder of the owner's favourites as the root folder, including the tree's root ("Favourites"), shown with its path. It can't be confirmed without a folder. A layout with a size other than `S`, `M` or `L`, or without a root folder ID, is rejected with 400 and `invalid_layout`.

#### Scenario: Add an explorer
- **WHEN** Ann picks "Favourite explorer" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid size
- **WHEN** a layout is saved with an explorer of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Explorer tree
The explorer SHALL show the children of its root folder as a tree, not the root folder itself, in the order of the favourites. A folder shows a folder icon and its name and can be expanded and collapsed. A link shows its favicon, or a globe icon when it has none, and its title, or its URL when it has no title. Activating a link opens its URL in a new tab. Long names are cut off with "…" at the widget's width. An empty root folder shows "This folder is empty" / "Dieser Ordner ist leer". The tree always shows the current favourites.

#### Scenario: Tree
- **WHEN** the root folder "Work" contains the folder "Tools" with the link "GitLab", and the link "Mail"
- **THEN** the explorer shows "Tools" with a folder icon and "Mail" with its favicon, and "GitLab" appears after expanding "Tools"

#### Scenario: Link without favicon
- **WHEN** a link in the explorer has no favicon
- **THEN** it shows a globe icon

#### Scenario: Favourites change
- **WHEN** Ann renames a link inside the root folder on `/favourites` and opens the dashboard again
- **THEN** the explorer shows the new name

### Requirement: Explorer size
The size SHALL set the icon size, the font size and the padding of the rows: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** an explorer's size is changed from M to L
- **THEN** its icons, text and row padding become larger

### Requirement: Collapse state
Every folder in the explorer SHALL start collapsed. The folders the viewer expands or collapses SHALL be remembered in their browser for that widget and restored when the dashboard is opened again. Nothing about it is stored on the server. A small button in the widget expands all folders, or collapses all of them when any folder is expanded. Its accessible name says which ("Expand all" / "Alle aufklappen", "Collapse all" / "Alle zuklappen").

#### Scenario: First visit
- **WHEN** someone opens a dashboard with an explorer for the first time in their browser
- **THEN** all its folders are collapsed

#### Scenario: Remembered
- **WHEN** Ann expands "Tools" and reloads the page
- **THEN** "Tools" is still expanded

#### Scenario: Expand all
- **WHEN** all folders are collapsed and Ann activates "Expand all"
- **THEN** every folder at every level is expanded, and the button now says "Collapse all"

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
