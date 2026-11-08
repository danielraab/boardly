# Spec Delta

## MODIFIED Requirements

### Requirement: Selected notes settings
The widget kind "Selected notes" / "Ausgewählte Notizen" SHALL have a list of notes (at least 1 and at most 100, each at most once, in the order the owner set), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width. There is no sort setting: the notes are always shown in the order of the list. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the listed notes. The settings dialog can't be confirmed while the list is empty. A layout with selected notes without a note, with more than 100 notes, with a note ID that is empty or longer than an ID can be, with the same note ID twice, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The notes don't have to exist when the layout is saved.

#### Scenario: Add selected notes
- **WHEN** Ann picks "Selected notes" in the widget picker
- **THEN** its settings dialog opens with no notes, no title, size M and "Show on public dashboard" off, and it can't be confirmed until she adds a note

#### Scenario: Duplicate note
- **WHEN** a layout is saved with selected notes that contain the same note ID twice
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Too many notes
- **WHEN** a layout is saved with selected notes that contain 0 or 101 note IDs
- **THEN** the server responds 400 with `invalid_layout`
