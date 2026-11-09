# Spec Delta

## MODIFIED Requirements

### Requirement: Explorer settings
A favourite explorer SHALL have a root folder, an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and a "Show to other viewers" flag (default off), besides the common width. The settings dialog SHALL offer every folder of the owner's favourites as the root folder, including the tree's root ("Favourites"), in a tree that can be searched by name, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder. A layout with a size other than `S`, `M` or `L`, a title longer than 200 characters, or without a root folder ID, is rejected with 400 and `invalid_layout`.

#### Scenario: Add an explorer
- **WHEN** Ann picks "Favourite explorer" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, no title, size M and "Show to other viewers" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid size
- **WHEN** a layout is saved with an explorer of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Explorer on public dashboards
For anyone other than the owner, an explorer with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely, together with its tree. With the flag on, visitors get the explorer with the tree of its root folder and nothing else from the owner's favourites. An explorer whose root folder no longer exists is left out for visitors.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with an explorer whose "Show to other viewers" is off
- **THEN** the response contains no trace of the explorer or its folder's contents, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the explorer with the root folder's contents

#### Scenario: Only the root folder
- **WHEN** a visitor's response contains an explorer for the folder "Work"
- **THEN** it contains no node from outside "Work"
