## MODIFIED Requirements

### Requirement: Recent notes rows
Each note SHALL be shown in its own row with its icon, if it has one, its display title (see "Display title" in the `notes` capability), and the chosen time as a relative time in the viewer's language (e.g. "3 days ago" / "vor 3 Tagen", "yesterday" / "gestern"), with the full date and time as its tooltip. A note with a category SHALL show the category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip; a note without a category has no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

#### Scenario: Row
- **WHEN** the widget shows "Groceries" with the symbol icon "cart" in the category "Garden" below "Home", added two days ago, and the sort is "Added"
- **THEN** the row shows the cart symbol, "Groceries", "Garden" below it with the tooltip "Home › Garden", and "2 days ago"

#### Scenario: Untitled note
- **WHEN** a note has no title and its content starts with "call the plumber"
- **THEN** the row shows "call the plumber" as its title

#### Scenario: Uncategorized
- **WHEN** a note has no category
- **THEN** its row has no second line

#### Scenario: Time follows the sort
- **WHEN** a note was created a month ago and changed yesterday, and the sort is "Last changed"
- **THEN** its row shows "yesterday"

#### Scenario: Coloured row
- **WHEN** the widget shows "Groceries" with the colour `green` and "Draft" without a colour
- **THEN** both rows have a border, the row of "Groceries" has the green note background and the row of "Draft" the `surface` background

### Requirement: Recent notes on public dashboards
For anyone other than the owner, a recent notes widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the widget with the notes it shows, each with its display title, icon, colour, chosen time and its category's title, and nothing else from the owner's notes: no content, IDs, versions, category paths or other notes. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a recent notes widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its notes, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the widget with its rows, and activating a row does nothing

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a recent notes widget with an untitled note whose content starts with "call the plumber" followed by a task list, in the category "Garden" below "Home"
- **THEN** it contains "call the plumber" and "Garden", and neither the task list, nor "Home", nor the note's ID
