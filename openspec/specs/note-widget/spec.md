# note-widget Specification

## Purpose
A dashboard widget that shows one of the owner's notes like a note card, lets the owner tick off its todos in place, and can be shown read-only on public dashboards.

## Requirements

### Requirement: Note widget settings
The widget kind `note` ("Note" / "Notiz") SHALL have, besides the common width and height, a note, a size (`S`, `M` or `L`, default `M`) and a "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" flag (default off). The settings dialog SHALL offer the note as a select of the owner's notes by their display title (see "Display title" in the `notes` capability), and SHALL NOT be confirmable without a note. The height is the common choice of `auto` or a fixed number of rows. A layout with a note widget without a note ID, with an unknown size, or with a config the kind doesn't accept is rejected with 400 and `invalid_layout`. The note doesn't have to exist when the layout is saved.

#### Scenario: Add a note widget
- **WHEN** Ann picks "Note" in the widget picker, selects "Groceries" and confirms
- **THEN** a note widget for "Groceries" with size M and "Show on public dashboard" off is added to the draft

#### Scenario: No note chosen
- **WHEN** Ann opens the settings of a new note widget and hasn't selected a note
- **THEN** the dialog can't be confirmed

#### Scenario: Untitled note in the select
- **WHEN** Ann has a note without a title whose content starts with "call the plumber"
- **THEN** the select lists it as "call the plumber"

#### Scenario: Invalid size
- **WHEN** a layout is saved with a note widget of size `XL`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Note widget rendering
A note widget SHALL show its note like a note card (see "Note cards" in the `notes` capability): the note's colour as background, or `surface` when it has none; at the top, its icon and its title, if any, followed on the same line by its category badge, if it has a category (a note with neither title, icon nor category shows no title line); and its rendered content with task items as checkboxes in their state. A long title is shortened so the badge stays on the line. The badge SHALL NOT open a category picker. Content is not cut off or faded: with height `auto` the widget is as tall as the note, and with a fixed height content that doesn't fit scrolls inside the widget. Links in the content SHALL NOT open.

#### Scenario: Card look
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** the widget has the green note background, shows the cart symbol and "Groceries" with the badge "Home" next to it at the top, and its content below

#### Scenario: Fixed height
- **WHEN** a note widget has a height of 3 rows and the note is longer than that
- **THEN** the widget is 3 rows tall and its content scrolls

#### Scenario: Note changed
- **WHEN** Ann adds a task to "Groceries" on `/notes` and opens the dashboard
- **THEN** the widget shows the new task

### Requirement: Note widget size
The size SHALL set the text size of the widget's title and content, including the size of the checkboxes and the icon, on the same S/M/L scale as the other widgets. The padding SHALL stay the same for every size.

#### Scenario: Large
- **WHEN** a note widget has the size L
- **THEN** its text and checkboxes are larger than with M, and its padding is the same

### Requirement: Note widget for the owner
For the owner, outside edit mode, a note widget SHALL show an "Open note" / "Notiz öffnen" button that opens the note's editor in a modal over the dashboard, without leaving it. The modal edits the note as on `/notes/<id>` (title, icon, colour, category, content, delete) and saves it the same way; it has a close button instead of the way back to the notes, and closing it, also with Escape or a click outside, saves what is left. While and after editing, the widget shows the saved note; when the note is deleted in the modal, the modal closes and the widget shows that the note no longer exists. A checkbox SHALL toggle its task item with a click, a tap or the keyboard. The change is saved right away and shown at once, without leaving the dashboard. When saving fails, the widget SHALL show the note as it is on the server again and a message SHALL say that the change couldn't be saved, or that the note was changed elsewhere when the save got `version_conflict`. The widget offers no other editing. In edit mode, the checkboxes SHALL NOT toggle and there is no "Open note" button.

#### Scenario: Tick off a todo
- **WHEN** Ann clicks the unchecked task "milk" in the "Groceries" widget on her dashboard
- **THEN** "milk" is shown checked, the browser stays on the dashboard, and the note's task item has `checked: true`

#### Scenario: Open the note
- **WHEN** Ann activates "Open note" on the widget
- **THEN** a modal with the editor of "Groceries" opens and the browser stays on the dashboard

#### Scenario: Edit in the modal
- **WHEN** Ann changes the title of "Groceries" to "Shopping" in the modal and closes it
- **THEN** the change is saved and the widget shows "Shopping"

#### Scenario: Delete in the modal
- **WHEN** Ann deletes "Groceries" in the modal
- **THEN** the modal closes and the widget shows "This note no longer exists"

#### Scenario: Changed elsewhere
- **WHEN** Ann changes "Groceries" in another tab and then ticks a task in the widget
- **THEN** the widget shows the note as it is on the server and a message says the note was changed elsewhere

#### Scenario: Edit mode
- **WHEN** Ann clicks a checkbox of the widget while editing the dashboard
- **THEN** the task doesn't change

### Requirement: Deleted note
When the note of a note widget no longer exists, the widget SHALL show "This note no longer exists" / "Diese Notiz existiert nicht mehr" to the owner. Its settings can still be opened to choose another note, and the layout can still be saved. For visitors, the widget is left out.

#### Scenario: Note deleted
- **WHEN** Ann deletes "Groceries" and opens a dashboard with a widget for it
- **THEN** the widget shows "This note no longer exists"

#### Scenario: Save with a missing note
- **WHEN** Ann edits another widget on that dashboard and saves
- **THEN** the layout is saved

### Requirement: Note widget on public dashboards
For anyone other than the owner, a note widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its note. With the flag on, visitors get the widget read-only: checkboxes show their state but can't be toggled, and there is no "Open note" button. A visitor's response contains only what the widget shows: the note's title, icon, colour and content, and its category's title, icon and colour, and nothing else from the owner's notes, such as other notes, the note's version or times, or other category names. For visitors, the badge's tooltip is the category's title.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its note, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the note with its checkboxes, can't toggle them, and sees no "Open note" button

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a note widget for a note in "Garden" below "Home"
- **THEN** it contains the category "Garden" and not "Home"

### Requirement: Note editor modal in the address
Opening a note's editor in a modal from a dashboard widget SHALL add `note=<note ID>` to the dashboard's address as a new browser history entry, keeping the dashboard's path. The browser's back button, including Android's, SHALL close the modal and go back to the dashboard's address without `note`. Closing the modal with its close button, Escape or a click outside SHALL do the same, so no extra history entry stays behind; when the modal was opened from an address that already contained `note` (a reload or a link), closing it replaces the address with the one without `note` instead. Unsaved changes are saved when the modal closes in any of these ways. An address with `note` SHALL open the dashboard with the modal of that note for the owner outside edit mode. For anyone other than the owner, in edit mode, or when the note doesn't exist, the parameter is removed from the address and no modal opens. Starting edit mode removes `note`.

#### Scenario: Back closes the modal
- **WHEN** Ann activates a row "Groceries" on `/ann/home` and then presses the back button
- **THEN** the modal closes, the address is `/ann/home`, and a second back leaves the dashboard

#### Scenario: Address while open
- **WHEN** Ann opens "Groceries" from a widget on `/ann/home`
- **THEN** the address is `/ann/home?note=<ID of Groceries>`

#### Scenario: Close button
- **WHEN** Ann closes the modal with its close button
- **THEN** the address is `/ann/home` again and back leaves the dashboard

#### Scenario: Reload
- **WHEN** Ann reloads `/ann/home?note=<ID of Groceries>`
- **THEN** the dashboard shows the modal of "Groceries", and closing it leaves the address `/ann/home`

#### Scenario: Unknown note
- **WHEN** Ann opens `/ann/home?note=gone` and no such note exists
- **THEN** the dashboard is shown without a modal and the address is `/ann/home`

#### Scenario: Visitor
- **WHEN** a visitor opens `/ann/home?note=<ID of Groceries>` of a public dashboard
- **THEN** no modal opens and the address is `/ann/home`
