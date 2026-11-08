# Spec Delta

## MODIFIED Requirements

### Requirement: Note widget settings
The widget kind `note` ("Note" / "Notiz") SHALL have, besides the common width, a note, a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" flag (default off). The settings dialog SHALL offer the note as a search field with the placeholder "Choose a note" / "Notiz wählen" over all of the owner's notes, archived ones included. Each result SHALL show the note's icon, its display title (see "Display title" in the `notes` capability) and, if it has a category, the category's path (e.g. "Home › Garden"). An archived note's result SHALL show an archive icon with the tooltip "Archived" / "Archiviert" and be dimmed. Typing filters the results to notes whose display title or category path contains the typed text, ignoring case and accents. Results are ordered by display title, with untitled notes without text last. When no note matches, the results show "No matching notes" / "Keine passenden Notizen". Choosing a result, with the pointer or the keyboard, makes it the widget's note. While the search isn't being typed in, the field SHALL show the chosen note with its icon, display title, category path and archived mark. The dialog SHALL NOT be confirmable without a note. A layout with a note widget without a note ID, with an unknown size, or with a config the kind doesn't accept is rejected with 400 and `invalid_layout`. The note doesn't have to exist when the layout is saved.

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

### Requirement: Note widget rendering
A note widget SHALL show its note like a note card (see "Note cards" in the `notes` capability): the note's colour as background, or `surface` when it has none; at the top, its icon and its title, if any, followed on the same line by its category badge, if it has a category, and at the end of the line a pin icon with the tooltip "Pinned" / "Angeheftet" for a pinned note and the mark "Archived" / "Archiviert" for an archived note (a note with neither title, icon, category nor mark shows no title line); and its rendered content with task items as checkboxes in their state. A long title is shortened so the badge stays on the line. The badge SHALL NOT open a category picker. Content is not cut off or faded: the widget is as tall as the note, and when it is the only child of a fixed-height section it fills that section, and a note that doesn't fit scrolls with the section. Links in the content SHALL NOT open.

#### Scenario: Card look
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** the widget has the green note background, shows the cart symbol and "Groceries" with the badge "Home" next to it at the top, and its content below

#### Scenario: Fixed height
- **WHEN** a note widget is the only child of a section with a height of 3 rows and the note is longer than that
- **THEN** the section is 3 rows tall and scrolls vertically, and the note isn't cut off

#### Scenario: Note changed
- **WHEN** Ann adds a task to "Groceries" on `/notes` and opens the dashboard
- **THEN** the widget shows the new task

#### Scenario: Archived note
- **WHEN** the widget's note is archived
- **THEN** the widget still shows the note, with the mark "Archived"

#### Scenario: Short note in a fixed-height section
- **WHEN** a note widget is the only child of a section with a height of 6 rows and the note is shorter than that
- **THEN** the note's card fills the section
