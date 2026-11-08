# Spec Delta

## MODIFIED Requirements

### Requirement: Note list settings
The widget kind "Note list" / "Notizliste" SHALL have a category, "Include subcategories" / "Unterkategorien einbeziehen" (default on), a sort ("Sort" / "Sortierung" with the choices "Last changed" / "Zuletzt geändert" (default), "Added" / "Hinzugefügt" and "Alphabetical" / "Alphabetisch"), a maximum number of notes ("Number of notes" / "Anzahl Notizen", an integer from 1 to 100, default 10, always required), an optional title (at most 200 characters after trimming; empty means none), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off), besides the common width. The settings dialog SHALL offer as the category "All notes" / "Alle Notizen", "Uncategorized" / "Ohne Kategorie" and every category of the owner as a tree, and it can't be confirmed without a category or with a number of notes outside 1–100. "Include subcategories" SHALL only be changeable when a category of the tree is chosen. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles of the listed notes. A layout with a note list without a category, with a category ID longer than an ID can be, with a sort other than `updated`, `created` or `title`, with a missing or out-of-range number of notes, with a title longer than 200 characters, or with a size other than `S`, `M` or `L`, is rejected with 400 and `invalid_layout`. The category doesn't have to exist when the layout is saved.

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
