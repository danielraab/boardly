## MODIFIED Requirements

### Requirement: Selected notes shown
The widget SHALL show the notes of its list that still exist, in the order of the list. Notes that no longer exist are skipped. The rows always show the current notes. Each note SHALL be shown in its own row with its icon, if it has one, and its display title. Every note with a category SHALL show its category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip. Notes without a category have no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. Rows SHALL NOT show the note's content or any time. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

#### Scenario: Manual order
- **WHEN** the list holds "Roses", "Groceries", "Plumber"
- **THEN** the widget shows them in this order, whatever their titles or times

#### Scenario: Category line
- **WHEN** "Roses" is in "Garden" below "Home" and "Groceries" has no category
- **THEN** the row of "Roses" shows "Garden" below it, with the tooltip "Home › Garden", and "Groceries" has no second line

#### Scenario: Untitled note
- **WHEN** a selected note has no title and its content starts with "call the plumber"
- **THEN** the row shows "call the plumber" as its title

#### Scenario: Note renamed
- **WHEN** Ann renames a selected note on `/notes` and opens the dashboard again
- **THEN** the widget shows the new title at the same place

#### Scenario: Coloured row
- **WHEN** the widget shows "Groceries" with the colour `green` and "Draft" without a colour
- **THEN** both rows have a border, the row of "Groceries" has the green note background and the row of "Draft" the `surface` background

### Requirement: Selected notes on public dashboards
For anyone other than the owner, selected notes with "Show on public dashboard" off, or of which no note exists anymore, SHALL be left out of the dashboard lookup response entirely, together with their notes. With the flag on, visitors get the widget with the notes it shows, in its order, each with its display title, icon, colour and, if it has a category, its category's title, and nothing else from the owner's notes: no content, note IDs in the rows, versions, times, category paths or other notes. Like every widget's settings, the widget's own settings, including the list of note IDs, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with selected notes whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its notes, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the widget with its rows, and activating a row does nothing

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains selected notes with an untitled note in "Garden" below "Home" whose content starts with "call the plumber" followed by a task list
- **THEN** its rows contain "call the plumber" and "Garden", and neither the task list, nor "Home", nor a note ID

#### Scenario: All notes missing for visitors
- **WHEN** every note of public selected notes was deleted and a visitor opens the dashboard
- **THEN** the response contains no trace of the widget
