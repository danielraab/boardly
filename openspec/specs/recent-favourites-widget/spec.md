# recent-favourites-widget Specification

## Purpose
A dashboard widget that lists the most recently added links of one folder of the owner's favourites, including its subfolders, limited by a maximum count and optionally by age.

## Requirements

### Requirement: Recent favourites settings
A recent favourites widget SHALL have a folder, a maximum number of links ("Number of links" / "Anzahl Links", an integer from 1 to 100, default 10, always required), an optional age limit ("Only the last … days" / "Nur die letzten … Tage", an integer from 1 to 365; empty means no age limit), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. The settings dialog SHALL offer every folder of the owner's favourites as the folder, including the tree's root ("Favourites"), in the same searchable folder tree as the favourite explorer, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder or with a number of links or days outside these ranges. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the newest links of the folder. A layout with a recent favourites widget without a folder ID, with a missing or out-of-range number of links, with a number of days outside 1–365, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a recent favourites widget
- **WHEN** Ann picks "Recent favourites" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, 10 links, no day limit, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid count
- **WHEN** a layout is saved with a recent favourites widget whose number of links is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid days
- **WHEN** a layout is saved with a recent favourites widget whose number of days is 0
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Recent links shown
A recent favourites widget SHALL show the links inside its folder and all its subfolders at every depth, newest first by the time they were added, at most as many as its number of links. Links added at the same time are ordered by their ID. With a number of days N, only links added within the last N × 24 hours are shown. Folders are never shown as items. The time a link was added is its creation time; a link imported from a file or another app keeps the creation time it had there, and moving or editing a link doesn't change it. The list always shows the current favourites.

#### Scenario: Newest first
- **WHEN** the folder "Work" contains "Mail" added three days ago and the folder "Tools" with "GitLab" added yesterday, and the number of links is 10
- **THEN** the widget shows "GitLab" and then "Mail", and not "Tools"

#### Scenario: Count limit
- **WHEN** the folder contains 15 links and the number of links is 5
- **THEN** the widget shows the 5 most recently added links

#### Scenario: Day limit
- **WHEN** the number of links is 10, the number of days is 7, and the folder contains 3 links added in the last 7 days and 20 older links
- **THEN** the widget shows only those 3 links

#### Scenario: Imported links
- **WHEN** Ann imports a bookmark today that was created in her browser two years ago
- **THEN** a widget limited to the last 7 days doesn't show it, and one without a day limit places it by its date two years ago

#### Scenario: Root folder
- **WHEN** the folder is the tree's root ("Favourites")
- **THEN** the widget considers every link of Ann's favourites

### Requirement: Recent favourites rows
Each link SHALL be shown in its own row with its favicon, or a globe icon when it has none, its title, or its URL when it has no title, its full URL below, and the time it was added as a relative time in the viewer's language (e.g. "3 days ago" / "vor 3 Tagen", "yesterday" / "gestern"), with the full date and time as its tooltip. Title and URL that don't fit are cut off with "…"; the item has its full title and URL as a tooltip. Activating a link opens its URL in a new tab; in edit mode it doesn't open.

#### Scenario: Row
- **WHEN** the widget shows "Mail" with the URL `https://mail.example.org` added two days ago
- **THEN** the row shows the favicon, "Mail", `https://mail.example.org` below it and "2 days ago"

#### Scenario: Without title
- **WHEN** a link has no title
- **THEN** the row shows its URL in place of the title

#### Scenario: Long URL
- **WHEN** a link's URL is wider than the widget
- **THEN** it is cut off with "…" and the full URL is in the tooltip

#### Scenario: Edit mode
- **WHEN** Ann activates a link while editing the dashboard
- **THEN** no page opens

### Requirement: Recent favourites empty state
A recent favourites widget without links to show SHALL display "No favourites" / "Keine Favoriten", or with a number of days N, "No favourites in the last N days" / "Keine Favoriten in den letzten N Tagen" ("No favourites in the last day" / "Keine Favoriten am letzten Tag" for N = 1).

#### Scenario: Empty folder
- **WHEN** the folder contains no links and there is no day limit
- **THEN** the widget shows "No favourites"

#### Scenario: Nothing recent
- **WHEN** the number of days is 7 and no link in the folder was added in the last 7 days
- **THEN** the widget shows "No favourites in the last 7 days"

### Requirement: Recent favourites size
The size SHALL set the icon size, the font size and the padding of the rows on the same scale as the link list: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** the size is changed from M to L
- **THEN** its icons, text and padding become larger

### Requirement: Recent favourites title
A recent favourites widget with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit. Its heading level is that of a section's title at the same place in the layout. A widget without a title shows no heading.

#### Scenario: With title
- **WHEN** Ann gives the widget the title "Recently added"
- **THEN** the widget shows the heading "Recently added" above its links

#### Scenario: Without title
- **WHEN** the widget has no title
- **THEN** the links start at the top of the widget

### Requirement: Deleted recent favourites folder
When the folder of a recent favourites widget no longer exists, the widget SHALL show "This folder no longer exists" / "Dieser Ordner existiert nicht mehr" to the owner. Its settings can still be opened to choose another folder, and the layout can still be saved.

#### Scenario: Folder deleted
- **WHEN** Ann deletes the folder of a recent favourites widget and opens the dashboard
- **THEN** the widget shows "This folder no longer exists"

#### Scenario: Save with a missing folder
- **WHEN** Ann edits another widget on a dashboard with such a widget and saves
- **THEN** the layout is saved

### Requirement: Recent favourites on public dashboards
For anyone other than the owner, a recent favourites widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its links. With the flag on, visitors get the widget with the links it shows, each with its title, URL, favicon and the time it was added, and nothing else from the owner's favourites: no folder names, tags or descriptions. A widget whose folder no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a recent favourites widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its links, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the widget with its links

#### Scenario: Only shown links
- **WHEN** a visitor's response contains a recent favourites widget limited to 5 links of the folder "Work"
- **THEN** it contains those 5 links and no folder name, and no node from outside "Work"

### Requirement: Recent favourites details buttons
For the owner, outside edit mode, every link of a recent favourites widget SHALL show an info button at the right end of its row that opens the link's details (see the favourite-details spec). Activating it SHALL NOT follow the link. The widget has no "Add favourite" button.

#### Scenario: Details
- **WHEN** Ann views the widget and activates the info button of "Mail"
- **THEN** the details of "Mail" open, and no new tab opens

#### Scenario: Visitor
- **WHEN** a visitor views the widget on Ann's public dashboard
- **THEN** no info buttons are shown
