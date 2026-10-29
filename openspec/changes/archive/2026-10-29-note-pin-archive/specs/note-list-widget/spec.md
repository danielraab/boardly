## MODIFIED Requirements

### Requirement: Note list notes shown
A note list SHALL show, at most as many as its number of notes:
- with "All notes", every note of the owner;
- with "Uncategorized", the notes without a category;
- with a category and "Include subcategories" on, the notes in the category and in all its subcategories at every depth;
- with a category and "Include subcategories" off, only the notes directly in the category.

Archived notes are included. Pinned notes SHALL come first, and the sort SHALL set the order of pinned and of unpinned notes: "Last changed" by time of last change, newest first; "Added" by creation time, newest first; "Alphabetical" by display title (see "Display title" in the `notes` capability), ignoring case and accents, with untitled notes without text ("Untitled note") last. Notes with the same time or text are ordered by their ID. The number of notes applies after sorting. The list always shows the current notes.

#### Scenario: With subcategories
- **WHEN** "Home" has the note "Plumber" and its subcategory "Garden" has "Roses", "Include subcategories" is on and the category is "Home"
- **THEN** the note list shows "Plumber" and "Roses"

#### Scenario: Without subcategories
- **WHEN** the same note list has "Include subcategories" off
- **THEN** it shows only "Plumber"

#### Scenario: Uncategorized
- **WHEN** the category is "Uncategorized"
- **THEN** the note list shows only notes without a category

#### Scenario: Alphabetical
- **WHEN** the category contains "zoo", "Ärzte" and an untitled note whose content starts with "beta", and the sort is "Alphabetical"
- **THEN** the note list shows "Ärzte", "beta", "zoo"

#### Scenario: Count limit
- **WHEN** the category has 15 notes, the sort is "Last changed" and the number of notes is 5
- **THEN** the note list shows the 5 most recently changed of them

#### Scenario: Pinned first
- **WHEN** the sort is "Alphabetical", the category contains "Ärzte" and "zoo", and "zoo" is pinned
- **THEN** the note list shows "zoo" and then "Ärzte"

#### Scenario: Pinned note of another category
- **WHEN** the category is "Home" and "Groceries" in "Work" is pinned
- **THEN** the note list doesn't show "Groceries"

#### Scenario: Pinned within the count
- **WHEN** the sort is "Last changed", the number of notes is 5, and the category has 15 notes, one of them pinned and changed a year ago
- **THEN** the note list shows the pinned note first and then the 4 most recently changed of the others

### Requirement: Note list rows
Each note SHALL be shown in its own row with its icon, if it has one, and its display title. A note that isn't directly in the chosen category, that is a note of a subcategory, or with "All notes" a note with any category, SHALL show its category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip. Other notes have no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. A pinned note SHALL show a pin icon at the end of its row, with the tooltip "Pinned" / "Angeheftet". An archived note SHALL show an archive icon at the end of its row, with the tooltip "Archived" / "Archiviert", and its row is dimmed. Rows SHALL NOT show the note's content or any time. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

#### Scenario: Note directly in the category
- **WHEN** the category is "Home" and "Plumber" is in "Home"
- **THEN** the row of "Plumber" has no second line

#### Scenario: Note of a subcategory
- **WHEN** the category is "Home", "Include subcategories" is on and "Roses" is in "Garden" below "Home"
- **THEN** the row of "Roses" shows "Garden" below it, with the tooltip "Home › Garden"

#### Scenario: All notes
- **WHEN** the category is "All notes", "Roses" is in "Garden" and "Draft" has no category
- **THEN** "Roses" shows "Garden" as its second line and "Draft" has none

#### Scenario: Untitled note
- **WHEN** a note has no title and its content starts with "call the plumber"
- **THEN** the row shows "call the plumber" as its title

#### Scenario: Coloured row
- **WHEN** the widget shows "Groceries" with the colour `green` and "Draft" without a colour
- **THEN** both rows have a border, the row of "Groceries" has the green note background and the row of "Draft" the `surface` background

#### Scenario: Pinned and archived rows
- **WHEN** the widget shows "Groceries", which is pinned, and "Old plan", which is archived
- **THEN** the row of "Groceries" shows the pin icon, and the row of "Old plan" is dimmed and shows the archive icon

### Requirement: Note list on public dashboards
For anyone other than the owner, a note list with "Show on public dashboard" off, or whose category no longer exists, SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the note list with the notes it shows, in its order, each with its display title, icon, colour, whether it is pinned and archived, and, where a second line is shown, its category's title, and nothing else from the owner's notes: no content, note IDs, versions, times, category paths, the chosen category's name, or other notes. Like every widget's settings, the widget's own settings, including the chosen category's ID, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note list whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the note list or its notes, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the note list with its rows, and activating a row does nothing

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a note list for "Home" with an untitled note in "Garden" below "Home" whose content starts with "call the plumber" followed by a task list
- **THEN** it contains "call the plumber" and "Garden", and neither the task list, nor "Home", nor the note's ID

#### Scenario: Category missing for visitors
- **WHEN** the category of a public note list was deleted and a visitor opens the dashboard
- **THEN** the response contains no trace of the note list
