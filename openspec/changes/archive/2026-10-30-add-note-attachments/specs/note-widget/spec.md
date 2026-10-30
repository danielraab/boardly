## MODIFIED Requirements

### Requirement: Note widget on public dashboards
For anyone other than the owner, a note widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its note. With the flag on, visitors get the widget read-only: checkboxes show their state but can't be toggled, and there is no "Open note" button. A visitor's response contains only what the widget shows: the note's title, icon, colour, content, the name, content type, size and URL of the attachments its content contains, whether it is pinned and archived, and its category's title, icon and colour, and nothing else from the owner's notes, such as other notes, the note's version or times, other category names, or other attachments. For visitors, the badge's tooltip is the category's title. Visitors can view the note's images in the image viewer and download its files (see "Attachments in shown notes" in the `note-attachments` capability).

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a note widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its note, and the page doesn't show it

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor opens the dashboard
- **THEN** the visitor sees the note with its checkboxes, can't toggle them, and sees no "Open note" button

#### Scenario: Only what is shown
- **WHEN** a visitor's response contains a note widget for a note in "Garden" below "Home"
- **THEN** it contains the category "Garden" and not "Home"

#### Scenario: Attachments for visitors
- **WHEN** the flag is on and the note contains `roses.jpg`
- **THEN** the visitor's response contains the name, type, size and URL of `roses.jpg` and nothing else about Ann's attachments, and the visitor sees the photo
