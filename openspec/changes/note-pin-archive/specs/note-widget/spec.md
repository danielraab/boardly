## MODIFIED Requirements

### Requirement: Note widget rendering
A note widget SHALL show its note like a note card (see "Note cards" in the `notes` capability): the note's colour as background, or `surface` when it has none; at the top, its icon and its title, if any, followed on the same line by its category badge, if it has a category, and at the end of the line a pin icon with the tooltip "Pinned" / "Angeheftet" for a pinned note and the mark "Archived" / "Archiviert" for an archived note (a note with neither title, icon, category nor mark shows no title line); and its rendered content with task items as checkboxes in their state. A long title is shortened so the badge stays on the line. The badge SHALL NOT open a category picker. Content is not cut off or faded: with height `auto` the widget is as tall as the note, and with a fixed height content that doesn't fit scrolls inside the widget. Links in the content SHALL NOT open.

#### Scenario: Card look
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** the widget has the green note background, shows the cart symbol and "Groceries" with the badge "Home" next to it at the top, and its content below

#### Scenario: Fixed height
- **WHEN** a note widget has a height of 3 rows and the note is longer than that
- **THEN** the widget is 3 rows tall and its content scrolls

#### Scenario: Note changed
- **WHEN** Ann adds a task to "Groceries" on `/notes` and opens the dashboard
- **THEN** the widget shows the new task

#### Scenario: Archived note
- **WHEN** the widget's note is archived
- **THEN** the widget still shows the note, with the mark "Archived"

### Requirement: Note widget for the owner
For the owner, outside edit mode, a note widget SHALL show an "Open note" / "Notiz öffnen" button that opens the note's editor in a modal over the dashboard, without leaving it. The modal edits the note as on `/notes/<id>` (title, icon, colour, category, content, pin, archive, delete) and saves it the same way; it has a close button instead of the way back to the notes, and closing it, also with Escape or a click outside, saves what is left. While and after editing, the widget shows the saved note; when the note is deleted in the modal, the modal closes and the widget shows that the note no longer exists. A checkbox SHALL toggle its task item with a click, a tap or the keyboard. The change is saved right away and shown at once, without leaving the dashboard. When saving fails, the widget SHALL show the note as it is on the server again and a message SHALL say that the change couldn't be saved, or that the note was changed elsewhere when the save got `version_conflict`. The widget offers no other editing. In edit mode, the checkboxes SHALL NOT toggle and there is no "Open note" button.

#### Scenario: Tick off a todo
- **WHEN** Ann clicks the unchecked task "milk" in the "Groceries" widget on her dashboard
- **THEN** "milk" is shown checked, the browser stays on the dashboard, and the note's task item has `checked: true`

#### Scenario: Open the note
- **WHEN** Ann activates "Open note" on the widget
- **THEN** a modal with the editor of "Groceries" opens and the browser stays on the dashboard

#### Scenario: Edit in the modal
- **WHEN** Ann changes the title of "Groceries" to "Shopping" in the modal and closes it
- **THEN** the change is saved and the widget shows "Shopping"

#### Scenario: Pin in the modal
- **WHEN** Ann chooses "Pin note" in the modal and closes it
- **THEN** the widget shows the pin icon

#### Scenario: Delete in the modal
- **WHEN** Ann deletes "Groceries" in the modal
- **THEN** the modal closes and the widget shows "This note no longer exists"

#### Scenario: Changed elsewhere
- **WHEN** Ann changes "Groceries" in another tab and then ticks a task in the widget
- **THEN** the widget shows the note as it is on the server and a message says the note was changed elsewhere

#### Scenario: Edit mode
- **WHEN** Ann clicks a checkbox of the widget while editing the dashboard
- **THEN** the task doesn't change

### Requirement: Note widget on public dashboards
For anyone other than the owner, a note widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its note. With the flag on, visitors get the widget read-only: checkboxes show their state but can't be toggled, and there is no "Open note" button. A visitor's response contains only what the widget shows: the note's title, icon, colour, content, whether it is pinned and archived, and its category's title, icon and colour, and nothing else from the owner's notes, such as other notes, the note's version or times, or other category names. For visitors, the badge's tooltip is the category's title.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its note, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the note with its checkboxes, can't toggle them, and sees no "Open note" button

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a note widget for a note in "Garden" below "Home"
- **THEN** it contains the category "Garden" and not "Home"

