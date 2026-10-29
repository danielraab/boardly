## MODIFIED Requirements

### Requirement: Note list rows
Each note SHALL be shown in its own row with its icon, if it has one, and its display title. A note that isn't directly in the chosen category, that is a note of a subcategory, or with "All notes" a note with any category, SHALL show its category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip. Other notes have no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. Rows SHALL NOT show the note's content or any time. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

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

### Requirement: Note list on public dashboards
For anyone other than the owner, a note list with "Show on public dashboard" off, or whose category no longer exists, SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the note list with the notes it shows, in its order, each with its display title, icon, colour and, where a second line is shown, its category's title, and nothing else from the owner's notes: no content, note IDs, versions, times, category paths, the chosen category's name, or other notes. Like every widget's settings, the widget's own settings, including the chosen category's ID, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

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
