## Purpose

A dashboard widget that lists the owner's notes of one chosen category, of all categories or of no category, sorted and limited by count, and opens them in the note editor modal.

## ADDED Requirements

### Requirement: Note list settings
The widget kind "Note list" / "Notizliste" SHALL have a category, "Include subcategories" / "Unterkategorien einbeziehen" (default on), a sort ("Sort" / "Sortierung" with the choices "Last changed" / "Zuletzt geändert" (default), "Added" / "Hinzugefügt" and "Alphabetical" / "Alphabetisch"), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. The settings dialog SHALL offer as the category "All notes" / "Alle Notizen", "Uncategorized" / "Ohne Kategorie" and every category of the owner as a tree, and it can't be confirmed without a category or with a number of notes outside 1–100. "Include subcategories" SHALL only be changeable when a category of the tree is chosen. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the listed notes. A layout with a note list without a category, with a category ID longer than an ID can be, with a sort other than `updated`, `created` or `title`, with a missing or out-of-range number of notes, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The category doesn't have to exist when the layout is saved.

#### Scenario: Add a note list
- **WHEN** Ann picks "Note list" in the widget picker
- **THEN** its settings dialog opens with no category chosen, "Include subcategories" on, the sort "Last changed", 10 notes, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she picks a category

#### Scenario: Subcategories for All notes
- **WHEN** Ann chooses "All notes" or "Uncategorized" as the category
- **THEN** "Include subcategories" can't be changed

#### Scenario: Invalid sort
- **WHEN** a layout is saved with a note list whose sort is `oldest`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid count
- **WHEN** a layout is saved with a note list whose number of notes is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Note list notes shown
A note list SHALL show, at most as many as its number of notes:
- with "All notes", every note of the owner;
- with "Uncategorized", the notes without a category;
- with a category and "Include subcategories" on, the notes in the category and in all its subcategories at every depth;
- with a category and "Include subcategories" off, only the notes directly in the category.

The sort SHALL set the order: "Last changed" by time of last change, newest first; "Added" by creation time, newest first; "Alphabetical" by display title (see "Display title" in the `notes` capability), ignoring case and accents, with untitled notes without text ("Untitled note") last. Notes with the same time or text are ordered by their ID. The number of notes applies after sorting. The list always shows the current notes.

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

### Requirement: Note list rows
Each note SHALL be shown in its own row with its icon, if it has one, and its display title. A note that isn't directly in the chosen category, that is a note of a subcategory, or with "All notes" a note with any category, SHALL show its category's title in smaller text as a second line below the display title, with the category's path (e.g. "Home › Garden") as its tooltip. Other notes have no second line. Rows SHALL NOT show the note's colour, content or any time. Text that doesn't fit is cut off with "…" and the row has the full display title as a tooltip.

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

### Requirement: Note list opens the editor
For the owner, outside edit mode, activating a row with a click, a tap or the keyboard SHALL open the note's editor in a modal over the dashboard, the same as the recent notes widget (see the `recent-notes-widget` capability), without leaving the dashboard. After the modal saves, or deletes the note, the widget shows the current notes. In edit mode, activating a row does nothing.

#### Scenario: Open a note
- **WHEN** Ann activates the row "Plumber" on her dashboard
- **THEN** a modal with the editor of "Plumber" opens and the browser stays on the dashboard

#### Scenario: Moved to another category in the modal
- **WHEN** Ann changes the category of a listed note to one outside the widget's category in the modal and closes it
- **THEN** the widget no longer shows the note

#### Scenario: Edit mode
- **WHEN** Ann activates a row while editing the dashboard
- **THEN** no modal opens

### Requirement: Note list empty state
A note list without notes to show SHALL display "No notes in this category" / "Keine Notizen in dieser Kategorie", or with "All notes" "No notes" / "Keine Notizen".

#### Scenario: Empty category
- **WHEN** the category "Work" has no notes
- **THEN** the widget shows "No notes in this category"

### Requirement: Deleted note list category
When a note list's category no longer exists, the note list SHALL show "This category no longer exists" / "Diese Kategorie existiert nicht mehr" to the owner. Its settings can still be opened to choose another category, and the layout can still be saved.

#### Scenario: Category deleted
- **WHEN** Ann deletes the category of a note list and opens the dashboard
- **THEN** the note list shows "This category no longer exists"

#### Scenario: Save with a missing category
- **WHEN** Ann edits another widget on a dashboard with such a note list and saves
- **THEN** the layout is saved

### Requirement: Note list size
The size SHALL set the icon size, the font size and the padding of the rows on the same scale as the recent notes widget: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** the size is changed from M to L
- **THEN** its icons, text and padding become larger

### Requirement: Note list title
A note list with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit. Its heading level is that of a section's title at the same place in the layout. A note list without a title shows no heading.

#### Scenario: With title
- **WHEN** Ann gives the note list the title "Work notes"
- **THEN** the widget shows the heading "Work notes" above its rows

#### Scenario: Without title
- **WHEN** the note list has no title
- **THEN** the rows start at the top of the widget

### Requirement: Note list on public dashboards
For anyone other than the owner, a note list with "Show on public dashboard" off, or whose category no longer exists, SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the note list with the notes it shows, in its order, each with its display title, icon and, where a second line is shown, its category's title, and nothing else from the owner's notes: no content, note IDs, versions, colours, times, category paths, the chosen category's name, or other notes. Like every widget's settings, the widget's own settings, including the chosen category's ID, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

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
