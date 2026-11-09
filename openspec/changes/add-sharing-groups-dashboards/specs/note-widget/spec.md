# Spec Delta

## MODIFIED Requirements

### Requirement: Note widget settings
The widget kind `note` ("Note" / "Notiz") SHALL have, besides the common width, a note, a size (`S`, `M` or `L`, default `M`) and a "Show to other viewers" / "Anderen Betrachtern zeigen" flag (default off). The settings dialog SHALL offer the note as a search field with the placeholder "Choose a note" / "Notiz wählen" over all of the owner's notes, archived ones included. Each result SHALL show the note's icon, its display title (see "Display title" in the `notes` capability) and, if it has a category, the category's path (e.g. "Home › Garden"). An archived note's result SHALL show an archive icon with the tooltip "Archived" / "Archiviert" and be dimmed. Typing filters the results to notes whose display title or category path contains the typed text, ignoring case and accents. Results are ordered by display title, with untitled notes without text last. When no note matches, the results show "No matching notes" / "Keine passenden Notizen". Choosing a result, with the pointer or the keyboard, makes it the widget's note. While the search isn't being typed in, the field SHALL show the chosen note with its icon, display title, category path and archived mark. The dialog SHALL NOT be confirmable without a note. A layout with a note widget without a note ID, with an unknown size, or with a config the kind doesn't accept is rejected with 400 and `invalid_layout`. The note doesn't have to exist when the layout is saved.

#### Scenario: Add a note widget
- **WHEN** Ann picks "Note" in the widget picker, chooses "Groceries" in the search and confirms
- **THEN** a note widget for "Groceries" with size M and "Show to other viewers" off is added to the draft

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

### Requirement: Note widget on public dashboards
For anyone other than the owner, a note widget with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely, together with its note. With the flag on, visitors get the widget read-only: checkboxes show their state but can't be toggled, and there is no "Open note" button. A visitor's response contains only what the widget shows: the note's title, icon, colour, content, the name, content type, size and URL of the attachments its content contains, whether it is pinned and archived, and its category's title, icon and colour, and nothing else from the owner's notes, such as other notes, the note's version or times, other category names, or other attachments. For visitors, the badge's tooltip is the category's title. Visitors can view the note's images in the image viewer and download its files (see "Attachments in shown notes" in the `note-attachments` capability).

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note widget whose "Show to other viewers" is off
- **THEN** the response contains no trace of the widget or its note, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the note with its checkboxes, can't toggle them, and sees no "Open note" button

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a note widget for a note in "Garden" below "Home"
- **THEN** it contains the category "Garden" and not "Home"

#### Scenario: Attachments for visitors
- **WHEN** the flag is on and the note contains `roses.jpg`
- **THEN** the visitor's response contains the name, type, size and URL of `roses.jpg` and nothing else about Ann's attachments, and the visitor sees the photo
