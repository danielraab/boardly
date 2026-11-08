# Spec Delta

## MODIFIED Requirements

### Requirement: Explorer settings
A favourite explorer SHALL have a root folder, an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" flag (default off), besides the common width. The settings dialog SHALL offer every folder of the owner's favourites as the root folder, including the tree's root ("Favourites"), in a tree that can be searched by name, and it SHALL show the path of the chosen folder. It can't be confirmed without a folder. A layout with a size other than `S`, `M` or `L`, a title longer than 200 characters, or without a root folder ID, is rejected with 400 and `invalid_layout`.

#### Scenario: Add an explorer
- **WHEN** Ann picks "Favourite explorer" in the widget picker
- **THEN** its settings dialog opens with no folder chosen, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she picks a folder

#### Scenario: Invalid size
- **WHEN** a layout is saved with an explorer of size `XL`
- **THEN** the server responds 400 with `invalid_layout`
