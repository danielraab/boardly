## MODIFIED Requirements

### Requirement: Category side panel
`/notes` SHALL show a side panel with "All notes" / "Alle Notizen", "Uncategorized" / "Ohne Kategorie", the user's categories as a tree with their icon, title and number of notes, "Archived" / "Archiviert" with an archive icon and its number of notes below the tree, and a "New category" / "Neue Kategorie" button. Categories with subcategories can be collapsed and expanded. Choosing an entry SHALL show only its notes in the grid: "All notes" every note that isn't archived, "Uncategorized" the notes without a category that aren't archived, a category the notes in it and in all its subcategories at every depth that aren't archived, and "Archived" every archived note of every category and uncategorized; each entry's number of notes counts the same notes. The choice SHALL be part of the address (`/notes?category=<id>`, `/notes?category=none`, `/notes?category=archived`), and "All notes" is the default. At phone width the side panel SHALL be a drawer that opens from a button next to the heading and closes after a choice.

#### Scenario: Filter by a parent category
- **WHEN** "Home" has 1 note and its subcategory "Garden" has 2, and Ann chooses "Home"
- **THEN** the grid shows all 3 notes, and "Home" shows the count 3

#### Scenario: Uncategorized
- **WHEN** Ann chooses "Uncategorized"
- **THEN** the grid shows only notes without a category, and the address is `/notes?category=none`

#### Scenario: Unknown category in the address
- **WHEN** Ann opens `/notes?category=doesnotexist`
- **THEN** the page shows all notes and "All notes" is chosen

#### Scenario: Drawer on a phone
- **WHEN** Ann opens `/notes` 375 px wide, opens the drawer and chooses "Work"
- **THEN** the drawer closes and the grid shows the notes of "Work"

#### Scenario: Archived notes not counted
- **WHEN** "Home" has 3 notes and one of them is archived
- **THEN** "Home" shows the count 2, and "Archived" shows the count 1

#### Scenario: Choose the archive
- **WHEN** Ann chooses "Archived" while archived notes are in "Home" and uncategorized
- **THEN** the grid shows both archived notes, and the address is `/notes?category=archived`
