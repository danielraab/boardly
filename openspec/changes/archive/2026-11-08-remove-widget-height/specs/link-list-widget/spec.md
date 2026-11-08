# Spec Delta

## MODIFIED Requirements

### Requirement: Link list settings
A link list SHALL have a folder, an optional title (at most 200 characters after trimming; empty means none), "Show items from subfolders" / "Einträge aus Unterordnern zeigen" (default off), an optional tag ("Tag" / "Tag"; default none), a sort ("Sort" / "Sortierung": `favourites`, `newest` or `alphabetical`, shown as "Favourites order" / "Reihenfolge der Favoriten", "Newest first" / "Neueste zuerst" and "Alphabetical" / "Alphabetisch"; default `favourites`), a layout (`list`, `compact` or `tiles`, shown as "List" / "Liste", "Compact" / "Kompakt" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width. The settings dialog SHALL offer every folder of the owner's favourites as the folder, including the tree's root ("Favourites"), in the same searchable folder tree as the favourite explorer, and it SHALL show the path of the chosen folder. For the tag, the dialog SHALL offer "No tag filter" / "Kein Tag-Filter" followed by every tag of the owner, sorted by name. It can't be confirmed without a folder. There is no target setting: links always open in a new tab. A saved link list without a tag or sort has no tag and the sort `favourites`. A layout with a link list without a folder ID, with a tag ID longer than an ID can be, with a title longer than 200 characters, or with a sort, layout or size outside these values, is rejected with 400 and `invalid_layout`.

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
