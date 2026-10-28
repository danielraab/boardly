## MODIFIED Requirements

### Requirement: Add widgets
"Add widget" SHALL open a picker dialog that lists every available widget kind with its icon, name and a short description, grouped under translated headings in this order: "Favourites" / "Favoriten" (favourite explorer, link, link list, recent favourites) and "Notes" / "Notizen" (note, recent notes, note list, selected notes). Picking a kind closes the picker and opens that kind's settings dialog for a new widget. A new widget starts with the full width of the section and height `auto`, plus the defaults of its kind. Confirming the settings dialog adds the widget at the end of the section in the draft. Cancelling the picker or the settings dialog leaves the draft unchanged.

#### Scenario: Add a widget
- **WHEN** Ann clicks "Add widget" in the section "Work", picks a kind and confirms its settings dialog
- **THEN** the new widget is the last child of "Work" in the draft

#### Scenario: Cancel the settings
- **WHEN** Ann picks a kind in the picker and then cancels its settings dialog
- **THEN** no widget is added

#### Scenario: Groups
- **WHEN** Ann opens the widget picker with the language set to German
- **THEN** it shows the heading "Favoriten" with the four favourite widgets, followed by the heading "Notizen" with "Notiz", "Neueste Notizen", "Notizliste" and "Ausgewählte Notizen"
