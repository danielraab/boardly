# Spec Delta

## MODIFIED Requirements

### Requirement: Recent notes settings
The widget kind "Recent notes" / "Neueste Notizen" SHALL have "Sort by" / "Sortieren nach" with the choices "Added" / "Hinzugefügt" (the note's creation time, default) and "Last changed" / "Zuletzt geändert" (the note's time of last change), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional age limit ("Only the last … days" / "Nur die letzten … Tage", an integer from 1 to 365; empty means no age limit), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width. It can't be confirmed with a number of notes or days outside these ranges. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the newest notes. A layout with a recent notes widget with a sort other than `created` or `updated`, with a missing or out-of-range number of notes, with a number of days outside 1–365, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`.

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
