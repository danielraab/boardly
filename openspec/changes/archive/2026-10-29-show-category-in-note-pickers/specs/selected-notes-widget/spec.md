## MODIFIED Requirements

### Requirement: Adding notes
The settings dialog SHALL offer an "Add a note…" / "Notiz hinzufügen …" search field over the owner's notes that aren't in the list yet, archived ones included. Each result SHALL show the note's icon, its display title (see "Display title" in the `notes` capability) and, if it has a category, the category's path (e.g. "Home › Garden"). An archived note's result SHALL show an archive icon with the tooltip "Archived" / "Archiviert" and be dimmed. Typing filters the results to notes whose display title or category path contains the typed text, ignoring case and accents. Results are ordered by display title, with untitled notes without text last. Choosing a result, with the pointer or the keyboard, adds the note at the end of the list and removes it from the results. When no note matches, the results show "No matching notes" / "Keine passenden Notizen".

#### Scenario: Search
- **WHEN** Ann types "plum" and she has the notes "Plumber" in "Home" and "Groceries"
- **THEN** the results show "Plumber" with "Home", and not "Groceries"

#### Scenario: Search by category
- **WHEN** Ann types "garden" and "Roses" is in "Home › Garden"
- **THEN** the results show "Roses"

#### Scenario: Archived note in the search
- **WHEN** Ann's note "Old plan" is archived and not in the list
- **THEN** its result shows the archive icon with the tooltip "Archived" and is dimmed

#### Scenario: Add a note
- **WHEN** Ann chooses "Plumber" from the results while the list holds "Groceries"
- **THEN** the list holds "Groceries", "Plumber" in this order, and "Plumber" is no longer offered

#### Scenario: Limit reached
- **WHEN** the list holds 100 notes
- **THEN** the search field is disabled and says that at most 100 notes can be selected

### Requirement: Ordering and removing notes
The settings dialog SHALL show the list of notes in their order, each with its icon, its display title, its category's path (e.g. "Home › Garden") if it has a category, a drag handle, "Move up" / "Nach oben", "Move down" / "Nach unten" and "Remove" / "Entfernen" buttons. An archived note SHALL show an archive icon with the tooltip "Archived" / "Archiviert" and its entry is dimmed. When the title and the path don't both fit, they are cut off with "…" and the entry has the display title and path as a tooltip. Dragging a note by its handle, with the pointer, by touch or with the keyboard, moves it to the place where it is dropped. "Move up" is disabled for the first note and "Move down" for the last. "Remove" takes the note out of the list and offers it again in the search. Changes take effect only when the dialog is confirmed.

#### Scenario: Move up
- **WHEN** the list holds "Groceries", "Plumber", "Roses" and Ann activates "Move up" on "Roses"
- **THEN** the list holds "Groceries", "Roses", "Plumber"

#### Scenario: Drag
- **WHEN** Ann drags "Roses" by its handle above "Groceries" and confirms
- **THEN** the widget shows "Roses" first

#### Scenario: Remove
- **WHEN** Ann removes "Plumber" from the list
- **THEN** "Plumber" is offered again in the search

#### Scenario: Same title in different categories
- **WHEN** the list holds "Groceries" in "Home" and "Groceries" in "Work › Office"
- **THEN** the entries show "Groceries" with "Home" and "Groceries" with "Work › Office"

#### Scenario: Archived note in the list
- **WHEN** the list holds the archived note "Old plan"
- **THEN** its entry shows the archive icon with the tooltip "Archived" and is dimmed
