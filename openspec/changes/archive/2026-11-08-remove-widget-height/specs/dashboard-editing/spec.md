# Spec Delta

## MODIFIED Requirements

### Requirement: Add widgets
"Add widget" SHALL open a picker dialog that lists every available widget kind with its icon, name and a short description, grouped under translated headings in this order: "Favourites" / "Favoriten" (favourite explorer, link, link list, recent favourites), "Notes" / "Notizen" (note, recent notes, note list, selected notes) and "Media" / "Medien" (media). Picking a kind closes the picker and opens that kind's settings dialog for a new widget. A new widget starts with "Full width" / "Volle Breite", which stores the width 12 so that it follows its section, plus the defaults of its kind. Confirming the settings dialog adds the widget at the end of the section in the draft. Cancelling the picker or the settings dialog leaves the draft unchanged.

#### Scenario: Add a widget
- **WHEN** Ann clicks "Add widget" in the section "Work", picks a kind and confirms its settings dialog
- **THEN** the new widget is the last child of "Work" in the draft

#### Scenario: Cancel the settings
- **WHEN** Ann picks a kind in the picker and then cancels its settings dialog
- **THEN** no widget is added

#### Scenario: Groups
- **WHEN** Ann opens the widget picker with the language set to German
- **THEN** it shows the heading "Favoriten" with the four favourite widgets, followed by the heading "Notizen" with "Notiz", "Neueste Notizen", "Notizliste" and "Ausgewählte Notizen", followed by the heading "Medien" with "Medien"

#### Scenario: New widget follows its section
- **WHEN** Ann adds a widget to a width-4 section, confirms, and then changes the section to width 6
- **THEN** the widget is 6 columns wide in the draft

### Requirement: Edit and delete widgets
A content widget's edit button SHALL open its kind's settings dialog with the widget's current values, including its width. The width is chosen from "Full width" / "Volle Breite" and the column counts up to the dashboard's desktop column count, and a stored width of 12 is shown as "Full width". The dialog SHALL NOT offer a height. Confirming applies the changes to the draft, and cancelling leaves it unchanged. The dialog offers a delete action. Deleting asks for confirmation and then removes the widget from the draft.

#### Scenario: Edit a widget
- **WHEN** Ann opens the edit dialog of a content widget, changes its width to 2 columns and confirms
- **THEN** the widget is 2 columns wide in the draft

#### Scenario: Delete a widget
- **WHEN** Ann chooses delete in a widget's dialog and confirms
- **THEN** the widget is gone from the draft

#### Scenario: No height choice
- **WHEN** Ann opens the edit dialog of a content widget of any kind
- **THEN** the dialog offers the width but no height
