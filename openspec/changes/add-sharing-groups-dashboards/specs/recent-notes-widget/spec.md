# Spec Delta

## MODIFIED Requirements

### Requirement: Recent notes settings
The widget kind "Recent notes" / "Neueste Notizen" SHALL have "Sort by" / "Sortieren nach" with the choices "Added" / "Hinzugefügt" (the note's creation time, default) and "Last changed" / "Zuletzt geändert" (the note's time of last change), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional age limit ("Only the last … days" / "Nur die letzten … Tage", an integer from 1 to 365; empty means no age limit), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off), besides the common width. It can't be confirmed with a number of notes or days outside these ranges. Next to "Show to other viewers", the dialog SHALL say that visitors then see the titles of the newest notes. A layout with a recent notes widget with a sort other than `created` or `updated`, with a missing or out-of-range number of notes, with a number of days outside 1–365, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a recent notes widget
- **WHEN** Ann picks "Recent notes" in the widget picker
- **THEN** its settings dialog opens with "Added", 10 notes, no day limit, no title, size M and "Show to other viewers" off, and it can be confirmed right away

#### Scenario: Invalid count
- **WHEN** a layout is saved with a recent notes widget whose number of notes is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid days
- **WHEN** a layout is saved with a recent notes widget whose number of days is 0
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid sort
- **WHEN** a layout is saved with a recent notes widget whose sort is `title`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Recent notes on public dashboards
For anyone other than the owner, a recent notes widget with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely, together with its notes. With the flag on, visitors get the widget with the notes it shows, each with its display title, icon, colour, chosen time, whether it is pinned and archived, and its category's title, and nothing else from the owner's notes: no content, IDs, versions, category paths or other notes. The display title of an untitled note is worked out on the server, so only its first line of text up to the display title's length is sent. For visitors, the second line's tooltip is the category's title, and rows can't be activated.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a recent notes widget whose "Show to other viewers" is off
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
