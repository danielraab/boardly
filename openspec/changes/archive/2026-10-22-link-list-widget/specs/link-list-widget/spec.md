# Spec Delta

## Purpose

A dashboard widget that shows the links of one folder of the owner's favourites as a flat list, in one of three layouts, optionally including the links of its subfolders.

## ADDED Requirements

### Requirement: Link list settings
A link list SHALL have a folder, an optional title (at most 200 characters after trimming; empty means none), "Show items from subfolders" / "Einträge aus Unterordnern zeigen" (default off), a layout (`list`, `compact` or `tiles`, shown as "List" / "Liste", "Compact" / "Kompakt" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. The settings dialog SHALL offer every folder of the owner's favourites as the folder, including the tree's root ("Favourites"), in the same searchable folder tree as the favourite explorer, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder. There is no target setting: links always open in a new tab. A layout with a link list without a folder ID, with a title longer than 200 characters, or with a layout or size outside these values, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a link list
- **WHEN** Ann picks "Link list" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, no title, "Show items from subfolders" off, the layout "List", size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Root folder
- **WHEN** Ann chooses "Favourites" as the folder and confirms
- **THEN** the link list is added and shows the links directly inside the root of her favourites

#### Scenario: Invalid layout
- **WHEN** a layout is saved with a link list whose layout is `grid`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid size
- **WHEN** a layout is saved with a link list of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Links shown
A link list SHALL show the links directly inside its folder, in the order of the favourites. Folders are never shown as items. With "Show items from subfolders" off, the contents of subfolders are left out. With it on, the links of every subfolder at every depth are included at the place of their subfolder, depth first. Each link shows its favicon, or a globe icon when it has none. Activating a link opens its URL in a new tab; in edit mode it doesn't open. Every item has its full title and URL as a tooltip. A folder without links to show displays "No links in this folder" / "Keine Links in diesem Ordner". The list always shows the current favourites.

#### Scenario: Without subfolders
- **WHEN** the folder "Work" contains the link "Mail", the folder "Tools" with the link "GitLab", and the link "Wiki", and "Show items from subfolders" is off
- **THEN** the link list shows "Mail" and "Wiki", and neither "Tools" nor "GitLab"

#### Scenario: With subfolders
- **WHEN** the same folder "Work" also has the folder "Deep" with the link "Grafana" inside "Tools", and "Show items from subfolders" is on
- **THEN** the link list shows "Mail", "GitLab", "Grafana", "Wiki" in this order, and no folder names

#### Scenario: Only subfolders
- **WHEN** the folder contains only subfolders and "Show items from subfolders" is off
- **THEN** the link list shows "No links in this folder"

#### Scenario: Favourites change
- **WHEN** Ann renames a link inside the folder on `/favourites` and opens the dashboard again
- **THEN** the link list shows the new name

#### Scenario: Edit mode
- **WHEN** Ann activates a link in the link list while editing the dashboard
- **THEN** no page opens

### Requirement: Link list layouts
The layout SHALL set how the links are arranged:
- "List": one link per row, with its icon, its title on the first line and its URL on the second, or only its URL when it has no title, like a link widget.
- "Compact": one link per row, with its icon and its title, or its URL when it has no title.
- "Tiles": a grid of tiles that wraps to fit the widget's width, each with a large icon and the title (or the URL) below it.

In every layout, text that doesn't fit is cut off with "…".

#### Scenario: List
- **WHEN** a link list with the layout "List" contains "Mail" with the URL `https://mail.example.org`
- **THEN** it shows "Mail" and below it `https://mail.example.org`

#### Scenario: Compact
- **WHEN** the layout is changed to "Compact"
- **THEN** it shows "Mail" without the URL, and the URL in its tooltip

#### Scenario: Tiles wrap
- **WHEN** the layout is "Tiles" and the widget is narrowed
- **THEN** fewer tiles fit in a row and the rest move to the next rows

### Requirement: Link list size
The size SHALL set the icon size, the font size and the padding of the items on the same scale as the link widget and the favourite explorer: S is the smallest, L the largest, and M is in between. In the layout "Tiles", it also sets the size of the tiles, so fewer of them fit in a row at a larger size.

#### Scenario: Large
- **WHEN** a link list's size is changed from M to L
- **THEN** its icons, text and padding become larger

### Requirement: Link list title
A link list with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit. Its heading level is that of a section's title at the same place in the layout. A link list without a title shows no heading.

#### Scenario: With title
- **WHEN** Ann gives a link list the title "Work links"
- **THEN** the link list shows the heading "Work links" above its links

#### Scenario: Without title
- **WHEN** a link list has no title
- **THEN** the links start at the top of the widget

### Requirement: Deleted link list folder
When a link list's folder no longer exists, the link list SHALL show "This folder no longer exists" / "Dieser Ordner existiert nicht mehr" to the owner. Its settings can still be opened to choose another folder, and the layout can still be saved.

#### Scenario: Folder deleted
- **WHEN** Ann deletes the folder of a link list and opens the dashboard
- **THEN** the link list shows "This folder no longer exists"

#### Scenario: Save with a missing folder
- **WHEN** Ann edits another widget on a dashboard with such a link list and saves
- **THEN** the layout is saved

### Requirement: Link list on public dashboards
For anyone other than the owner, a link list with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its links. With the flag on, visitors get the link list with the links it shows and nothing else from the owner's favourites: no folder names and no links outside the folder, nor from subfolders when "Show items from subfolders" is off. A link list whose folder no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a link list whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the link list or its links, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the link list with its links

#### Scenario: Only shown links
- **WHEN** a visitor's response contains a link list for the folder "Work" with "Show items from subfolders" off, and "Work" contains the folder "Tools"
- **THEN** the response contains neither "Tools" nor any link inside it, nor any node from outside "Work"
