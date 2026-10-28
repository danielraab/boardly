# recent-notes-widget Specification

## Purpose
A dashboard widget that lists the owner's most recently added or most recently changed notes, limited by a maximum count and optionally by age, and opens them in the note editor modal.

## Requirements

### Requirement: Recent notes settings
The widget kind "Recent notes" / "Neueste Notizen" SHALL have "Sort by" / "Sortieren nach" with the choices "Added" / "Hinzugefügt" (the note's creation time, default) and "Last changed" / "Zuletzt geändert" (the note's time of last change), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional age limit ("Only the last … days" / "Nur die letzten … Tage", an integer from 1 to 365; empty means no age limit), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width and height. It can't be confirmed with a number of notes or days outside these ranges. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the newest notes. A layout with a recent notes widget with a sort other than `created` or `updated`, with a missing or out-of-range number of notes, with a number of days outside 1–365, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`.

#### Scenario: Add a recent notes widget
- **WHEN** Ann picks "Recent notes" in the widget picker
- **THEN** its settings dialog opens with "Added", 10 notes, no day limit, no title, size M and "Show on public dashboard" off, and it can be confirmed right away

#### Scenario: Invalid count
- **WHEN** a layout is saved with a recent notes widget whose number of notes is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid days
- **WHEN** a layout is saved with a recent notes widget whose number of days is 0
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Invalid sort
- **WHEN** a layout is saved with a recent notes widget whose sort is `title`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Recent notes shown
A recent notes widget SHALL show the owner's notes of every category and uncategorized, newest first by the chosen time, at most as many as its number of notes: with "Added" by creation time, with "Last changed" by time of last change. Notes with the same time are ordered by their ID. With a number of days N, only notes whose chosen time is within the last N × 24 hours are shown. The list always shows the current notes.

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

### Requirement: Recent notes open the editor
For the owner, outside edit mode, activating a row with a click, a tap or the keyboard SHALL open the note's editor in a modal over the dashboard, the same as the note widget's "Open note" (see the `note-widget` capability), without leaving the dashboard. After the modal saves, or deletes the note, the widget shows the current notes. In edit mode, activating a row does nothing.

#### Scenario: Open a note
- **WHEN** Ann activates the row "Groceries" on her dashboard
- **THEN** a modal with the editor of "Groceries" opens and the browser stays on the dashboard

#### Scenario: Changed in the modal
- **WHEN** the sort is "Last changed" and Ann changes the title of the second note to "Shopping" in the modal and closes it
- **THEN** the widget shows "Shopping" as its first row

#### Scenario: Deleted in the modal
- **WHEN** Ann deletes a note in the modal
- **THEN** the modal closes and the widget no longer shows the note

#### Scenario: Edit mode
- **WHEN** Ann activates a row while editing the dashboard
- **THEN** no modal opens

### Requirement: Recent notes empty state
A recent notes widget without notes to show SHALL display "No notes" / "Keine Notizen", or with a number of days N, "No notes in the last N days" / "Keine Notizen in den letzten N Tagen" ("No notes in the last day" / "Keine Notizen am letzten Tag" for N = 1).

#### Scenario: No notes
- **WHEN** Ann has no notes and there is no day limit
- **THEN** the widget shows "No notes"

#### Scenario: Nothing recent
- **WHEN** the number of days is 7 and no note's chosen time is in the last 7 days
- **THEN** the widget shows "No notes in the last 7 days"

### Requirement: Recent notes size
The size SHALL set the icon size, the font size and the padding of the rows on the same scale as the recent favourites widget: S is the smallest, L the largest, and M is in between.

#### Scenario: Large
- **WHEN** the size is changed from M to L
- **THEN** its icons, text and padding become larger

### Requirement: Recent notes title
A recent notes widget with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit. Its heading level is that of a section's title at the same place in the layout. A widget without a title shows no heading.

#### Scenario: With title
- **WHEN** Ann gives the widget the title "Latest notes"
- **THEN** the widget shows the heading "Latest notes" above its rows

#### Scenario: Without title
- **WHEN** the widget has no title
- **THEN** the rows start at the top of the widget

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
