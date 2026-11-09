# Spec Delta

## MODIFIED Requirements

### Requirement: Note list settings
The widget kind "Note list" / "Notizliste" SHALL have a category, "Include subcategories" / "Unterkategorien einbeziehen" (default on), a sort ("Sort" / "Sortierung" with the choices "Last changed" / "Zuletzt geändert" (default), "Added" / "Hinzugefügt" and "Alphabetical" / "Alphabetisch"), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off), besides the common width. The settings dialog SHALL offer as the category "All notes" / "Alle Notizen", "Uncategorized" / "Ohne Kategorie" and every category of the owner as a tree, and it can't be confirmed without a category or with a number of notes outside 1–100. "Include subcategories" SHALL only be changeable when a category of the tree is chosen. Next to "Show to other viewers", the dialog SHALL say that visitors then see the titles of the listed notes. A layout with a note list without a category, with a category ID longer than an ID can be, with a sort other than `updated`, `created` or `title`, with a missing or out-of-range number of notes, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The category doesn't have to exist when the layout is saved.

#### Scenario: Add a note list
- **WHEN** Ann picks "Note list" in the widget picker
- **THEN** its settings dialog opens with no category chosen, "Include subcategories" on, the sort "Last changed", 10 notes, no title, size M and "Show to other viewers" off, and it can't be confirmed until she picks a category

#### Scenario: Subcategories for All notes
- **WHEN** Ann chooses "All notes" or "Uncategorized" as the category
- **THEN** "Include subcategories" can't be changed

#### Scenario: Invalid sort
- **WHEN** a layout is saved with a note list whose sort is `oldest`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid count
- **WHEN** a layout is saved with a note list whose number of notes is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Note list on public dashboards
For anyone other than the owner, a note list with "Show to other viewers" off, or whose category no longer exists, SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the note list with the notes it shows, in its order, each with its display title, icon, colour, whether it is pinned and archived, and, where a second line is shown, its category's title, and nothing else from the owner's notes: no content, note IDs, versions, times, category paths, the chosen category's name, or other notes. Like every widget's settings, the widget's own settings, including the chosen category's ID, are part of the response. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note list whose "Show to other viewers" is off
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
