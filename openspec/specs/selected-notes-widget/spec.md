# selected-notes-widget Specification

## Purpose
A dashboard widget that lists notes the owner picked by hand, in the order the owner set, and opens them in the note editor modal.

## Requirements

### Requirement: Selected notes settings
The widget kind "Selected notes" / "Ausgewählte Notizen" SHALL have a list of notes (at least 1 and at most 100, each at most once, in the order the owner set), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. There is no sort setting: the notes are always shown in the order of the list. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the listed notes. The settings dialog can't be confirmed while the list is empty. A layout with selected notes without a note, with more than 100 notes, with a note ID that is empty or longer than an ID can be, with the same note ID twice, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The notes don't have to exist when the layout is saved.

#### Scenario: Add selected notes
- **WHEN** Ann picks "Selected notes" in the widget picker
- **THEN** its settings dialog opens with no notes, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she adds a note

#### Scenario: Duplicate note
- **WHEN** a layout is saved with selected notes that contain the same note ID twice
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Too many notes
- **WHEN** a layout is saved with selected notes that contain 0 or 101 note IDs
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Adding notes
The settings dialog SHALL offer an "Add a note…" / "Notiz hinzufügen …" search field over the owner's notes that aren't in the list yet. Each result SHALL show the note's icon, its display title (see "Display title" in the `notes` capability) and, if it has a category, the category's path (e.g. "Home › Garden"). Typing filters the results to notes whose display title or category path contains the typed text, ignoring case and accents. Results are ordered by display title, with untitled notes without text last. Choosing a result, with the pointer or the keyboard, adds the note at the end of the list and removes it from the results. When no note matches, the results show "No matching notes" / "Keine passenden Notizen".

#### Scenario: Search
- **WHEN** Ann types "plum" and she has the notes "Plumber" in "Home" and "Groceries"
- **THEN** the results show "Plumber" with "Home", and not "Groceries"

#### Scenario: Search by category
- **WHEN** Ann types "garden" and "Roses" is in "Home › Garden"
- **THEN** the results show "Roses"

#### Scenario: Add a note
- **WHEN** Ann chooses "Plumber" from the results while the list holds "Groceries"
- **THEN** the list holds "Groceries", "Plumber" in this order, and "Plumber" is no longer offered

#### Scenario: Limit reached
- **WHEN** the list holds 100 notes
- **THEN** the search field is disabled and says that at most 100 notes can be selected

### Requirement: Ordering and removing notes
The settings dialog SHALL show the list of notes in their order, each with its icon, its display title, a drag handle, "Move up" / "Nach oben", "Move down" / "Nach unten" and "Remove" / "Entfernen" buttons. Dragging a note by its handle, with the pointer, by touch or with the keyboard, moves it to the place where it is dropped. "Move up" is disabled for the first note and "Move down" for the last. "Remove" takes the note out of the list and offers it again in the search. Changes take effect only when the dialog is confirmed.

#### Scenario: Move up
- **WHEN** the list holds "Groceries", "Plumber", "Roses" and Ann activates "Move up" on "Roses"
- **THEN** the list holds "Groceries", "Roses", "Plumber"

#### Scenario: Drag
- **WHEN** Ann drags "Roses" by its handle above "Groceries" and confirms
- **THEN** the widget shows "Roses" first

#### Scenario: Remove
- **WHEN** Ann removes "Plumber" from the list
- **THEN** "Plumber" is offered again in the search

### Requirement: Selected notes shown
The widget SHALL show the notes of its list that still exist, archived ones included: first the pinned notes, then the others, each in the order of the list. Notes that no longer exist are skipped. The rows always show the current notes. Each note SHALL be shown in its own row with its icon, if it has one, and its display title. Every note with a category SHALL show its category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip. Notes without a category have no second line. Each row SHALL look like a small note card (see "Note cards" in the `notes` capability): it has a border and rounded corners, and the note's colour as its background, or `surface` when the note has none. Rows are separated by a small gap. A pinned note SHALL show a pin icon at the end of its row, with the tooltip "Pinned" / "Angeheftet". An archived note SHALL show an archive icon at the end of its row, with the tooltip "Archived" / "Archiviert", and its row is dimmed. Rows SHALL NOT show the note's content or any time. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

#### Scenario: Manual order
- **WHEN** the list holds "Roses", "Groceries", "Plumber" and none of them is pinned
- **THEN** the widget shows them in this order, whatever their titles or times

#### Scenario: Pinned first
- **WHEN** the list holds "Roses", "Groceries", "Plumber" and "Plumber" is pinned
- **THEN** the widget shows "Plumber", "Roses", "Groceries"

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

#### Scenario: Pinned and archived rows
- **WHEN** the widget shows "Groceries", which is pinned, and "Old plan", which is archived
- **THEN** the row of "Groceries" shows the pin icon, and the row of "Old plan" is dimmed and shows the archive icon

### Requirement: Deleted selected notes
Notes of the list that no longer exist SHALL be skipped in the widget without a placeholder. When none of the notes exists anymore, the widget SHALL show "None of the selected notes exist anymore" / "Keine der ausgewählten Notizen existiert mehr" to the owner. In the settings dialog, a note of the list that no longer exists SHALL be shown at its place as "This note no longer exists" / "Diese Notiz existiert nicht mehr" with a warning icon and the "Remove" button. The dialog can be confirmed with such notes in the list, and the layout can still be saved.

#### Scenario: One note deleted
- **WHEN** the list holds "Groceries", "Plumber", "Roses" and Ann deletes "Plumber"
- **THEN** the widget shows "Groceries" and "Roses"

#### Scenario: All notes deleted
- **WHEN** Ann deletes every note of the list and opens the dashboard
- **THEN** the widget shows "None of the selected notes exist anymore"

#### Scenario: Deleted note in the settings
- **WHEN** Ann opens the settings of a widget whose list holds a deleted note
- **THEN** the dialog shows "This note no longer exists" at its place, and she can remove it

#### Scenario: Save with deleted notes
- **WHEN** Ann edits another widget on a dashboard with such selected notes and saves
- **THEN** the layout is saved

### Requirement: Selected notes open the editor
For the owner, outside edit mode, activating a row with a click, a tap or the keyboard SHALL open the note's editor in a modal over the dashboard, the same as the recent notes widget (see the `recent-notes-widget` capability), without leaving the dashboard. After the modal saves, or deletes the note, the widget shows the current notes. In edit mode, activating a row does nothing.

#### Scenario: Open a note
- **WHEN** Ann activates the row "Plumber" on her dashboard
- **THEN** a modal with the editor of "Plumber" opens and the browser stays on the dashboard

#### Scenario: Deleted in the modal
- **WHEN** Ann deletes "Plumber" in the modal
- **THEN** the widget no longer shows "Plumber"

#### Scenario: Edit mode
- **WHEN** Ann activates a row while editing the dashboard
- **THEN** no modal opens

### Requirement: Selected notes size
The size SHALL set the icon size, the font size and the padding of the rows on the same scale as the recent notes widget: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** the size is changed from M to L
- **THEN** its icons, text and padding become larger

### Requirement: Selected notes title
Selected notes with a title SHALL show it as a heading at the top of the widget, cut off with "…" when it doesn't fit. Its heading level is that of a section's title at the same place in the layout. Selected notes without a title show no heading.

#### Scenario: With title
- **WHEN** Ann gives the widget the title "Daily"
- **THEN** the widget shows the heading "Daily" above its rows

#### Scenario: Without title
- **WHEN** the widget has no title
- **THEN** the rows start at the top of the widget

### Requirement: Selected notes on public dashboards
For anyone other than the owner, selected notes with "Show on public dashboard" off, or of which no note exists anymore, SHALL be left out of the dashboard lookup response entirely, together with their notes. With the flag on, visitors get the widget with the notes it shows, in its order, each with its display title, icon, colour, whether it is pinned and archived, and, if it has a category, its category's title, and nothing else from the owner's notes: no content, note IDs in the rows, versions, times, category paths or other notes. Like every widget's settings, the widget's own settings, including the list of note IDs, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

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
