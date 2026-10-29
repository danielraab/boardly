## MODIFIED Requirements

### Requirement: Recent notes shown
A recent notes widget SHALL show the owner's notes of every category and uncategorized, archived ones included, at most as many as its number of notes. Pinned notes SHALL come first; pinned and unpinned notes are each ordered newest first by the chosen time: with "Added" by creation time, with "Last changed" by time of last change. Notes with the same time are ordered by their ID. The number of notes applies after this order. With a number of days N, only notes whose chosen time is within the last N × 24 hours are shown. The list always shows the current notes.

#### Scenario: Newest added first
- **WHEN** Ann created "Groceries" three days ago and "Plumber" yesterday, the sort is "Added" and the number of notes is 10
- **THEN** the widget shows "Plumber" and then "Groceries"

#### Scenario: Last changed
- **WHEN** Ann created "Groceries" three days ago, created "Plumber" yesterday, changed "Groceries" an hour ago, and the sort is "Last changed"
- **THEN** the widget shows "Groceries" and then "Plumber"

#### Scenario: Count limit
- **WHEN** Ann has 15 notes and the number of notes is 5
- **THEN** the widget shows the 5 newest notes by the chosen time

#### Scenario: Day limit
- **WHEN** the sort is "Added", the number of days is 7, and Ann created 3 notes in the last 7 days and 20 before
- **THEN** the widget shows only those 3 notes

#### Scenario: Old note changed recently
- **WHEN** the number of days is 7, and a note was created a month ago and changed yesterday
- **THEN** a widget sorted by "Last changed" shows it, and one sorted by "Added" doesn't

#### Scenario: Pinned first
- **WHEN** the sort is "Added", Ann created "Groceries" three days ago and pinned it, and created "Plumber" yesterday
- **THEN** the widget shows "Groceries" and then "Plumber"

#### Scenario: Pinned note outside the day limit
- **WHEN** the sort is "Added", the number of days is 7, and a pinned note was created a month ago
- **THEN** the widget doesn't show it

#### Scenario: Archived note shown
- **WHEN** "Old plan" is archived and among the newest notes
- **THEN** the widget shows it with the archived mark

### Requirement: Recent notes rows
Each note SHALL be shown in its own row with its icon, if it has one, its display title (see "Display title" in the `notes` capability), and the chosen time as a relative time in the viewer's language (e.g. "3 days ago" / "vor 3 Tagen", "yesterday" / "gestern"), with the full date and time as its tooltip. A note with a category SHALL show the category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip; a note without a category has no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. A pinned note SHALL show a pin icon at the end of its row, with the tooltip "Pinned" / "Angeheftet". An archived note SHALL show an archive icon at the end of its row, with the tooltip "Archived" / "Archiviert", and its row is dimmed. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

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

#### Scenario: Pinned and archived rows
- **WHEN** the widget shows "Groceries", which is pinned, and "Old plan", which is archived
- **THEN** the row of "Groceries" shows the pin icon, and the row of "Old plan" is dimmed and shows the archive icon

### Requirement: Recent notes on public dashboards
For anyone other than the owner, a recent notes widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the widget with the notes it shows, each with its display title, icon, colour, chosen time, whether it is pinned and archived, and its category's title, and nothing else from the owner's notes: no content, IDs, versions, category paths or other notes. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a recent notes widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its notes, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the widget with its rows, and activating a row does nothing

#### Scenario: Marks for visitors
- **WHEN** a visitor opens the dashboard and the widget shows a pinned note
- **THEN** the visitor sees its row with the pin icon

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a recent notes widget with an untitled note whose content starts with "call the plumber" followed by a task list, in the category "Garden" below "Home"
- **THEN** it contains "call the plumber" and "Garden", and neither the task list, nor "Home", nor the note's ID
