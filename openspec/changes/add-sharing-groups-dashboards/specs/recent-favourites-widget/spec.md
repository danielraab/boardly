# Spec Delta

## MODIFIED Requirements

### Requirement: Recent favourites settings
A recent favourites widget SHALL have a folder, a maximum number of links ("Number of links" / "Anzahl Links", an integer from 1 to 100, default 10, always required), an optional age limit ("Only the last … days" / "Nur die letzten … Tage", an integer from 1 to 365; empty means no age limit), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off), besides the common width. The settings dialog SHALL offer every folder of the owner's favourites as the folder, including the tree's root ("Favourites"), in the same searchable folder tree as the favourite explorer, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder or with a number of links or days outside these ranges. Next to "Show to other viewers", the dialog SHALL say that visitors then see the newest links of the folder. A layout with a recent favourites widget without a folder ID, with a missing or out-of-range number of links, with a number of days outside 1–365, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a recent favourites widget
- **WHEN** Ann picks "Recent favourites" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, 10 links, no day limit, no title, size M and "Show to other viewers" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid count
- **WHEN** a layout is saved with a recent favourites widget whose number of links is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid days
- **WHEN** a layout is saved with a recent favourites widget whose number of days is 0
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Recent favourites on public dashboards
For anyone other than the owner, a recent favourites widget with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely, together with its links. With the flag on, visitors get the widget with the links it shows, each with its title, URL, favicon and the time it was added, and nothing else from the owner's favourites: no folder names, tags or descriptions. A widget whose folder no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a recent favourites widget whose "Show to other viewers" is off
- **THEN** the response contains no trace of the widget or its links, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the widget with its links

#### Scenario: Only shown links
- **WHEN** a visitor's response contains a recent favourites widget limited to 5 links of the folder "Work"
- **THEN** it contains those 5 links and no folder name, and no node from outside "Work"
