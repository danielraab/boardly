# note-categories Specification

## Purpose
Lets a user group notes into nested categories of their own, unrelated to favourites folders: the category entity and API, the side panel on `/notes` for filtering and managing them, and the category badge on notes.

## Requirements

### Requirement: Category entity
A note category SHALL belong to one user and have a title (1 to 200 characters after trimming, required), an optional icon and an optional colour with the same choices as for notes, and an optional parent category of the same user. Categories can be nested to any depth. Categories SHALL be kept when no note uses them. Sibling categories are ordered by title, case-insensitively in the viewer's language. Titles need not be unique.

#### Scenario: Nested category
- **WHEN** Ann creates the category "Home" and then "Garden" with the parent "Home"
- **THEN** "Garden" is shown below "Home"

#### Scenario: Empty title
- **WHEN** a category is saved with the title "   "
- **THEN** the server responds 400 with `invalid_category`

#### Scenario: Unused category stays
- **WHEN** Ann moves the only note of "Work" to another category
- **THEN** "Work" still exists

### Requirement: Categories API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/note-categories` lists all of the user's categories with `id`, `title`, `icon`, `color` and `parentId`.
- `POST /api/note-categories` creates a category and responds 201 with it.
- `PATCH /api/note-categories/{id}` changes the given fields and responds 200 with the category.
- `DELETE /api/note-categories/{id}` deletes the category and responds 204.

Icons and colours follow the same rules as for notes. A category of another user, or one that doesn't exist, gets 404 `not_found`. An invalid title, icon or colour gets 400 `invalid_category`. A parent that isn't one of the user's categories, or that is the category itself or one of its subcategories, gets 400 `invalid_parent`.

#### Scenario: Move into own subcategory
- **WHEN** Ann sets the parent of "Home" to its subcategory "Garden"
- **THEN** the server responds 400 with `invalid_parent`, and nothing changes

#### Scenario: Move to the top level
- **WHEN** Ann sets the parent of "Garden" to `null`
- **THEN** "Garden" becomes a top-level category

### Requirement: Delete categories
Deleting a category SHALL make every note in it uncategorized, without changing those notes' time of last change, and SHALL move its direct subcategories to its parent, or to the top level when it has none. Notes in the subcategories keep their category. Every active app token whose root category is the deleted category SHALL be revoked; tokens rooted at its subcategories keep working. The web app SHALL ask for confirmation and, when the category has notes or subcategories, say how many notes become uncategorized and that the subcategories move up. When active app tokens use the category as their root category, the confirmation SHALL also say how many connected apps use it and that they will be revoked. Revoked tokens stay in the user's token list, and their root category is shown as "Deleted category" / "Gelöschte Kategorie".

#### Scenario: Delete a category with notes
- **WHEN** Ann deletes "Work", which has 3 notes, and confirms
- **THEN** the 3 notes have no category, and their order on `/notes` is unchanged

#### Scenario: Delete a category with subcategories
- **WHEN** Ann deletes "Home", whose subcategory "Garden" has 2 notes
- **THEN** "Garden" becomes a top-level category and keeps its 2 notes

#### Scenario: Delete revokes tokens
- **WHEN** Ann deletes "Inbox", which is the root category of her active Karakeep token "Phone", and confirms
- **THEN** "Phone" is revoked, requests with it are rejected, and the "Connected apps" tab lists it with the root category "Deleted category"

#### Scenario: Warning about connected apps
- **WHEN** Ann starts deleting "Inbox", which is the root category of 2 active tokens and 1 revoked token
- **THEN** the confirmation says that 2 connected apps use this category and will be revoked

#### Scenario: Subcategory root keeps working
- **WHEN** a token has the root category "Home/Garden" and Ann deletes "Home"
- **THEN** the token stays active, and "Garden" is now a top-level category

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

### Requirement: Manage categories
Each category in the side panel SHALL have a menu with "Edit" / "Bearbeiten", "New subcategory" / "Neue Unterkategorie" and "Delete" / "Löschen". "New category", "New subcategory" and "Edit" SHALL open a dialog with the title, the icon (the icon chooser with "Own image" and "Symbol"), the colour and the parent category ("None" / "Keine" or a category; for "Edit" without the category itself and its subcategories). "New subcategory" preselects the category as parent. The dialog can't be confirmed without a title.

#### Scenario: Rename
- **WHEN** Ann chooses "Edit" on "Work", changes the title to "Job" and confirms
- **THEN** the side panel and every badge of its notes show "Job"

#### Scenario: Move
- **WHEN** Ann edits "Garden" and chooses the parent "Projects"
- **THEN** "Garden" is shown below "Projects" with its notes

#### Scenario: Parent choices
- **WHEN** Ann edits "Home", whose subcategory is "Garden"
- **THEN** the parent choices contain neither "Home" nor "Garden"

### Requirement: Category badge
A note with a category SHALL show a badge with the category's icon, if any, and its title, with the category's colour as background, or `surface` with a `line` border when it has none. Its tooltip SHALL be the category's path, e.g. "Home › Garden". In the editor and on the card, activating the badge SHALL open a picker with "No category" / "Keine Kategorie" and the user's categories as a tree; choosing one SHALL change the note's category right away. In the editor, a note without a category SHALL show an "Add category" / "Kategorie hinzufügen" button in the badge's place that opens the same picker. Activating the badge on a card SHALL NOT open the note.

#### Scenario: Badge path
- **WHEN** a note is in "Garden" below "Home"
- **THEN** its badge shows "Garden" with the tooltip "Home › Garden"

#### Scenario: Change from the card
- **WHEN** Ann activates the badge "Work" on a card and chooses "Home"
- **THEN** the card shows the badge "Home", and the note stays closed

#### Scenario: Remove the category
- **WHEN** Ann activates the badge in the editor and chooses "No category"
- **THEN** the badge is replaced by "Add category"
