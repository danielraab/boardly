## MODIFIED Requirements

### Requirement: Note widget settings
The widget kind `note` ("Note" / "Notiz") SHALL have, besides the common width and height, a note, a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" flag (default off). The settings dialog SHALL offer the note as a search field with the placeholder "Choose a note" / "Notiz wählen" over all of the owner's notes, archived ones included. Each result SHALL show the note's icon, its display title (see "Display title" in the `notes` capability) and, if it has a category, the category's path (e.g. "Home › Garden"). An archived note's result SHALL show an archive icon with the tooltip "Archived" / "Archiviert" and be dimmed. Typing filters the results to notes whose display title or category path contains the typed text, ignoring case and accents. Results are ordered by display title, with untitled notes without text last. When no note matches, the results show "No matching notes" / "Keine passenden Notizen". Choosing a result, with the pointer or the keyboard, makes it the widget's note. While the search isn't being typed in, the field SHALL show the chosen note with its icon, display title, category path and archived mark. The dialog SHALL NOT be confirmable without a note. The height is the common choice of `auto` or a fixed number of rows. A layout with a note widget without a note ID, with an unknown size, or with a config the kind doesn't accept is rejected with 400 and `invalid_layout`. The note doesn't have to exist when the layout is saved.

#### Scenario: Add a note widget
- **WHEN** Ann picks "Note" in the widget picker, chooses "Groceries" in the search and confirms
- **THEN** a note widget for "Groceries" with size M and "Show on public dashboard" off is added to the draft

#### Scenario: No note chosen
- **WHEN** Ann opens the settings of a new note widget and hasn't chosen a note
- **THEN** the dialog can't be confirmed

#### Scenario: Untitled note in the select
- **WHEN** Ann has a note without a title whose content starts with "call the plumber"
- **THEN** the search lists it as "call the plumber"

#### Scenario: Same title in different categories
- **WHEN** Ann has a note "Groceries" in "Home" and one in "Office" below "Work", and types "groc"
- **THEN** the results show "Groceries" with "Home" and "Groceries" with "Work › Office"

#### Scenario: Search by category
- **WHEN** Ann types "office" and "Groceries" is in "Work › Office"
- **THEN** the results show that "Groceries", and not the one in "Home"

#### Scenario: Archived note in the search
- **WHEN** Ann's note "Old plan" is archived
- **THEN** its result shows the archive icon with the tooltip "Archived" and is dimmed, and it can still be chosen

#### Scenario: Chosen note shown
- **WHEN** Ann opens the settings of a note widget for "Groceries" in "Work › Office"
- **THEN** the field shows "Groceries" with its icon and "Work › Office"

#### Scenario: No match
- **WHEN** Ann types text that no display title or category path contains
- **THEN** the results show "No matching notes"

#### Scenario: Invalid size
- **WHEN** a layout is saved with a note widget of size `XL`
- **THEN** the server responds 400 with `invalid_layout`
