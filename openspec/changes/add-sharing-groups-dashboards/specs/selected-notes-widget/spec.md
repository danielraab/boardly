# Spec Delta

## MODIFIED Requirements

### Requirement: Selected notes settings
The widget kind "Selected notes" / "Ausgewählte Notizen" SHALL have a list of notes (at least 1 and at most 100, each at most once, in the order the owner set), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off), besides the common width. There is no sort setting: the notes are always shown in the order of the list. Next to "Show to other viewers", the dialog SHALL say that visitors then see the titles of the listed notes. The settings dialog can't be confirmed while the list is empty. A layout with selected notes without a note, with more than 100 notes, with a note ID that is empty or longer than an ID can be, with the same note ID twice, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The notes don't have to exist when the layout is saved.

#### Scenario: Add selected notes
- **WHEN** Ann picks "Selected notes" in the widget picker
- **THEN** its settings dialog opens with no notes, no title, size M and "Show to other viewers" off, and it can't be confirmed until she adds a note

#### Scenario: Duplicate note
- **WHEN** a layout is saved with selected notes that contain the same note ID twice
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Too many notes
- **WHEN** a layout is saved with selected notes that contain 0 or 101 note IDs
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Selected notes on public dashboards
For anyone other than the owner, selected notes with "Show to other viewers" off, or of which no note exists anymore, SHALL be left out of the dashboard lookup response entirely, together with their notes. With the flag on, visitors get the widget with the notes it shows, in its order, each with its display title, icon, colour, whether it is pinned and archived, and, if it has a category, its category's title, and nothing else from the owner's notes: no content, note IDs in the rows, versions, times, category paths or other notes. Like every widget's settings, the widget's own settings, including the list of note IDs, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with selected notes whose "Show to other viewers" is off
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
