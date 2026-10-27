# Spec Delta

## MODIFIED Requirements

### Requirement: Link list settings
A link list SHALL have a folder, an optional title (at most 200 characters after trimming; empty means none), "Show items from subfolders" / "Einträge aus Unterordnern zeigen" (default off), an optional tag ("Tag" / "Tag"; default none), a sort ("Sort" / "Sortierung": `favourites`, `newest` or `alphabetical`, shown as "Favourites order" / "Reihenfolge der Favoriten", "Newest first" / "Neueste zuerst" and "Alphabetical" / "Alphabetisch"; default `favourites`), a layout (`list`, `compact` or `tiles`, shown as "List" / "Liste", "Compact" / "Kompakt" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. The settings dialog SHALL offer every folder of the owner's favourites as the folder, including the tree's root ("Favourites"), in the same searchable folder tree as the favourite explorer, and it SHALL show the path of the chosen folder. For the tag, the dialog SHALL offer "No tag filter" / "Kein Tag-Filter" followed by every tag of the owner, sorted by name. It can't be confirmed without a folder. There is no target setting: links always open in a new tab. A saved link list without a tag or sort has no tag and the sort `favourites`. A layout with a link list without a folder ID, with a tag ID longer than an ID can be, with a title longer than 200 characters, or with a sort, layout or size outside these values, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a link list
- **WHEN** Ann picks "Link list" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, no title, "Show items from subfolders" off, "No tag filter", the sort "Favourites order", the layout "List", size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Root folder
- **WHEN** Ann chooses "Favourites" as the folder and confirms
- **THEN** the link list is added and shows the links directly inside the root of her favourites

#### Scenario: Tags offered
- **WHEN** Ann has the tags "reading" and "monitoring" and opens the tag setting
- **THEN** it offers "No tag filter", "monitoring" and "reading" in this order

#### Scenario: Existing link list
- **WHEN** a link list saved before tags and sorting existed is shown or saved again
- **THEN** it has no tag filter and the sort "Favourites order", and shows the same links in the same order as before

#### Scenario: Invalid sort
- **WHEN** a layout is saved with a link list whose sort is `oldest`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid layout
- **WHEN** a layout is saved with a link list whose layout is `grid`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid size
- **WHEN** a layout is saved with a link list of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Links shown
A link list SHALL show the links directly inside its folder. Folders are never shown as items. With "Show items from subfolders" off, the contents of subfolders are left out. With it on, the links of every subfolder at every depth are included; in the sort "Favourites order" they are at the place of their subfolder, depth first. With a tag, only links that have the tag are shown. A link list without links to show displays "No links in this folder" / "Keine Links in diesem Ordner", or with a tag "No links with this tag in this folder" / "Keine Links mit diesem „Tag“ in diesem Ordner". Each link shows its favicon, or a globe icon when it has none. Activating a link opens its URL in a new tab; in edit mode it doesn't open. Every item has its full title and URL as a tooltip. The list always shows the current favourites.

#### Scenario: Without subfolders
- **WHEN** the folder "Work" contains the link "Mail", the folder "Tools" with the link "GitLab", and the link "Wiki", and "Show items from subfolders" is off
- **THEN** the link list shows "Mail" and "Wiki", and neither "Tools" nor "GitLab"

#### Scenario: With subfolders
- **WHEN** the same folder "Work" also has the folder "Deep" with the link "Grafana" inside "Tools", "Show items from subfolders" is on and the sort is "Favourites order"
- **THEN** the link list shows "Mail", "GitLab", "Grafana", "Wiki" in this order, and no folder names

#### Scenario: Only subfolders
- **WHEN** the folder contains only subfolders and "Show items from subfolders" is off
- **THEN** the link list shows "No links in this folder"

#### Scenario: Tag filter
- **WHEN** in the same folder "Work" with subfolders on only "GitLab" and "Grafana" have the tag "monitoring", and the link list's tag is "monitoring"
- **THEN** the link list shows "GitLab" and "Grafana", and neither "Mail" nor "Wiki"

#### Scenario: All links with a tag
- **WHEN** the folder is "Favourites", "Show items from subfolders" is on and the tag is "monitoring"
- **THEN** the link list shows every link with "monitoring", wherever it is in the favourites

#### Scenario: Tag without links in the folder
- **WHEN** no link in the folder has the link list's tag
- **THEN** the link list shows "No links with this tag in this folder"

#### Scenario: Tag renamed
- **WHEN** Ann renames the tag of a link list from "monitoring" to "ops"
- **THEN** the link list still shows the links with that tag

#### Scenario: Favourites change
- **WHEN** Ann renames a link inside the folder on `/favourites` and opens the dashboard again
- **THEN** the link list shows the new name

#### Scenario: Edit mode
- **WHEN** Ann activates a link in the link list while editing the dashboard
- **THEN** no page opens

### Requirement: Deleted link list folder
When a link list's folder no longer exists, the link list SHALL show "This folder no longer exists" / "Dieser Ordner existiert nicht mehr" to the owner. When its folder exists but its tag no longer exists, it SHALL show "This tag no longer exists" / "Dieser „Tag“ existiert nicht mehr" to the owner. Its settings can still be opened to choose another folder or tag, and the layout can still be saved.

#### Scenario: Folder deleted
- **WHEN** Ann deletes the folder of a link list and opens the dashboard
- **THEN** the link list shows "This folder no longer exists"

#### Scenario: Tag deleted
- **WHEN** Ann deletes the tag of a link list in "Manage tags", or merges it into another tag, and opens the dashboard
- **THEN** the link list shows "This tag no longer exists"

#### Scenario: Tag removed from its last link
- **WHEN** Ann removes the tag of a link list from the only link that had it
- **THEN** the link list shows "This tag no longer exists"

#### Scenario: Save with a missing folder
- **WHEN** Ann edits another widget on a dashboard with such a link list and saves
- **THEN** the layout is saved

### Requirement: Link list on public dashboards
For anyone other than the owner, a link list with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its links. With the flag on, visitors get the link list with the links it shows, in its order, and nothing else from the owner's favourites: no folder or tag names, no dates, no links outside the folder, nor from subfolders when "Show items from subfolders" is off, nor links without its tag. A link list whose folder or tag no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a link list whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the link list or its links, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the link list with its links

#### Scenario: Only shown links
- **WHEN** a visitor's response contains a link list for the folder "Work" with "Show items from subfolders" off, and "Work" contains the folder "Tools"
- **THEN** the response contains neither "Tools" nor any link inside it, nor any node from outside "Work"

#### Scenario: Only tagged links
- **WHEN** a visitor's response contains a link list with the tag "monitoring"
- **THEN** the response contains only the links with "monitoring", and no tag names or dates

#### Scenario: Tag missing for visitors
- **WHEN** the tag of a public link list was deleted and a visitor opens the dashboard
- **THEN** the response contains no trace of the link list

## ADDED Requirements

### Requirement: Link list sort
The sort SHALL set the order of the links shown:
- "Favourites order": the order of the favourites, depth first.
- "Newest first": by the time the link was added, newest first; links added at the same time by ID.
- "Alphabetical": by the shown text (the title, or the URL when it has no title), ignoring case and accents, so that "Ärzte" comes before "Zoo"; equal texts by URL.

With "Show items from subfolders" on, "Newest first" and "Alphabetical" sort all links shown together, regardless of their subfolder.

#### Scenario: Newest first
- **WHEN** the folder contains "Mail" added on 1 March, "Wiki" added on 3 March and "Docs" added on 2 March, and the sort is "Newest first"
- **THEN** the link list shows "Wiki", "Docs", "Mail"

#### Scenario: Alphabetical
- **WHEN** the folder contains "zoo", "Ärzte" and a link without title with the URL `https://beta.example.org`, and the sort is "Alphabetical"
- **THEN** the link list shows "Ärzte", `https://beta.example.org`, "zoo"

#### Scenario: Sorted across subfolders
- **WHEN** "Show items from subfolders" is on, "Work" contains "Wiki" and the folder "Tools" with "GitLab", and the sort is "Alphabetical"
- **THEN** the link list shows "GitLab", "Wiki"
