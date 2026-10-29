# notes Specification

## Purpose
Lets a signed-in user keep notes with rich content and checkable todos: the note entity, its API, the card grid on `/notes` and the editor page with autosave.

## Requirements

### Requirement: Note entity
A note SHALL belong to one user and have an optional title (at most 200 characters after trimming; empty means none), an optional icon, an optional colour, at most one category of the same user (see the `note-categories` capability), a content document (see the `note-document` capability), a pinned flag, an archived flag, a version, a creation time and a time of last change. The icon is either a symbol from the same symbol set as the link widget or an image the user uploaded (see the `images` capability). The colour is one of the note colours `red`, `orange`, `yellow`, `green`, `teal`, `blue`, `purple`, `pink` and `grey` (see "Note colour palette" in the `theming` capability). The pinned and archived flags are independent of each other and both off for a new note unless set when it is created. The version starts at 1 and goes up by one with every change. Every change to the title, icon, colour, category, content, pinned flag or archived flag SHALL set the time of last change; a category becoming unset because it was deleted SHALL NOT.

#### Scenario: Minimal note
- **WHEN** Ann creates a note with only the content "call the plumber"
- **THEN** the note has no title, no icon, no colour, no category, is neither pinned nor archived, and has version 1

#### Scenario: Title too long
- **WHEN** a note is saved with a title of 201 characters
- **THEN** the server responds 400 with `invalid_note`

#### Scenario: Version goes up
- **WHEN** Ann changes the colour of a note with version 4
- **THEN** the note has version 5 and its time of last change is now

#### Scenario: Pinning is a change
- **WHEN** Ann pins a note with version 4 that was last changed a week ago
- **THEN** the note is pinned, has version 5 and its time of last change is now

#### Scenario: Pinned and archived
- **WHEN** Ann archives a pinned note
- **THEN** the note is both pinned and archived

### Requirement: Notes API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/notes` lists all of the user's notes, archived ones included, with all their fields, ordered by time of last change, newest first, then by ID.
- `GET /api/notes/{id}` returns one note.
- `POST /api/notes` creates a note from `title`, `icon`, `color`, `categoryId`, `pinned`, `archived` and either `content` or `text` (all optional; a missing content is an empty document, and a missing flag is `false`) and responds 201 with the note. `text` is plain text that the server turns into the content (see "Content from plain text" in the note-document spec). A request with both `content` and `text`, or a `text` that isn't a string, gets 400 `invalid_request`.
- `PATCH /api/notes/{id}` changes the given fields of a note. The body SHALL contain the `version` the change is based on; if it isn't the note's current version, the server responds 409 with `version_conflict` and changes nothing. Otherwise it responds 200 with the changed note.
- `DELETE /api/notes/{id}` deletes the note and responds 204.

A note is returned with `pinned` and `archived` as booleans besides its other fields. A note of another user, or one that doesn't exist, gets 404 `not_found`. An icon is `null`, `{"type": "symbol", "name": "<symbol>"}` or `{"type": "image", "imageId": "<id>"}`. An unknown symbol, an image that isn't an upload of the same user, an unknown colour, a category that isn't one of the user's, or a `pinned` or `archived` that isn't a boolean gets 400 `invalid_note`. An invalid content document, or a `text` whose document would be larger than allowed, gets 400 `invalid_document`.

#### Scenario: Create and list
- **WHEN** Ann creates the notes "Groceries" and then "Sprint plan", and lists her notes
- **THEN** the list shows "Sprint plan" first and "Groceries" second

#### Scenario: Edited note moves up
- **WHEN** Ann then edits "Groceries" and lists her notes again
- **THEN** "Groceries" comes first

#### Scenario: Stale version
- **WHEN** a note has version 3 and a PATCH with `version: 2` arrives
- **THEN** the server responds 409 with `version_conflict`, and the note is unchanged

#### Scenario: Other user's note
- **WHEN** Bob requests `GET /api/notes/{id}` for one of Ann's notes
- **THEN** the server responds 404 with `not_found`

#### Scenario: Other user's category
- **WHEN** Ann saves a note with the ID of one of Bob's categories
- **THEN** the server responds 400 with `invalid_note`

#### Scenario: Other user's image
- **WHEN** Ann saves a note whose icon is an image Bob uploaded
- **THEN** the server responds 400 with `invalid_note`

#### Scenario: Create from text
- **WHEN** Ann sends `POST /api/notes` with `{"text": "milk\neggs"}`
- **THEN** the server responds 201, and the note's content is one paragraph "milk", a line break, "eggs"

#### Scenario: Content and text
- **WHEN** a client sends `POST /api/notes` with both `content` and `text`
- **THEN** the server responds 400 with `invalid_request` and creates nothing

#### Scenario: Archive through the API
- **WHEN** Ann sends `PATCH /api/notes/{id}` with `{"version": 2, "archived": true}` for a note with version 2
- **THEN** the server responds 200 with the note, `archived` `true` and version 3, and `GET /api/notes` still lists it

#### Scenario: Flag that isn't a boolean
- **WHEN** a client sends `PATCH /api/notes/{id}` with `"pinned": "yes"`
- **THEN** the server responds 400 with `invalid_note`, and the note is unchanged

### Requirement: Display title
Where a note's title is shown and the note has no title, the web app SHALL show the first line of text of its content instead, cut off with "…" after 80 characters, and "Untitled note" / "Unbenannte Notiz" when the content has no text either.

#### Scenario: Untitled note with content
- **WHEN** a note has no title and its content starts with the paragraph "call the plumber tomorrow"
- **THEN** its display title is "call the plumber tomorrow"

#### Scenario: Empty note
- **WHEN** a note has no title and no text
- **THEN** its display title is "Untitled note"

### Requirement: Notes page
`/notes` SHALL require a signed-in user and show the heading "Notes" / "Notizen", a "New note" / "Neue Notiz" button, the category side panel (see the `note-categories` capability) and the user's notes as a grid of cards. The grid SHALL show only the notes of the entry chosen in the side panel: archived notes only with "Archived", and every other entry only notes that aren't archived. Within the grid, pinned notes SHALL come first; pinned and unpinned notes are each ordered by time of last change, newest first. The number of columns SHALL follow the available width, with one column at phone width. Each card's "Open note" / "Notiz öffnen" button SHALL open `/notes/<id>`. Signed-out visitors SHALL be sent to `/login`. The document title is "Notes - <app title>".

#### Scenario: Open a note
- **WHEN** Ann activates "Open note" on the card "Groceries" on `/notes`
- **THEN** the browser shows `/notes/<id of Groceries>`

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/notes`
- **THEN** the browser shows `/login`

#### Scenario: Phone width
- **WHEN** `/notes` is viewed 375 px wide
- **THEN** the cards are shown in one column

#### Scenario: Pinned first
- **WHEN** Ann changed "Plumber" an hour ago and "Groceries" a week ago, and "Groceries" is pinned
- **THEN** "Groceries" is the first card and "Plumber" the second

#### Scenario: Pinned note of another category
- **WHEN** "Groceries" is pinned and in "Home", and Ann chooses "Work"
- **THEN** the grid doesn't show "Groceries"

#### Scenario: Archived note left out
- **WHEN** "Old plan" is archived and Ann chooses "All notes"
- **THEN** the grid doesn't show "Old plan"

### Requirement: Note cards
A note card SHALL show the note's icon, if any, and its title, if any, followed by its rendered content, with task items shown as checkboxes in their state, and an "Open note" / "Notiz öffnen" button. A pinned note SHALL show a pin icon with the label "Pinned" / "Angeheftet" in the card's top corner. An archived note SHALL show the mark "Archived" / "Archiviert" with an archive icon next to the category badge and be shown dimmed. Content that doesn't fit into the card's maximum height is cut off and fades out. A note with a colour SHALL have that colour as the card's background; otherwise the card uses `surface`. A note with a category SHALL show the category badge (see "Category badge" in the `note-categories` capability) at the bottom of the card. A checkbox on the card SHALL toggle its task item with a click, a tap or the keyboard; the change is saved right away and shown at once. Links in the card's content SHALL NOT open; activating them starts inline editing like the rest of the content (see "Inline editing on cards"). When saving a toggled checkbox fails, the card SHALL show the note as it is on the server again and a message SHALL say that the change couldn't be saved, or that the note was changed elsewhere when the save got `version_conflict`.

#### Scenario: Card with colour and category
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** its card has the green note background, shows the cart symbol and "Groceries", its content, the "Open note" button and the badge "Home"

#### Scenario: Long note
- **WHEN** a note has 60 lines of content
- **THEN** its card shows the first lines and fades out at its maximum height

#### Scenario: Untitled card
- **WHEN** a note has no title
- **THEN** its card shows no title line and starts with its content

#### Scenario: Tick off a todo on the card
- **WHEN** Ann clicks the checkbox of the unchecked task "milk" on the card "Groceries"
- **THEN** "milk" is shown checked and struck through, the browser stays on `/notes`, and the note's task item has `checked: true`

#### Scenario: Toggle fails
- **WHEN** Ann toggles a checkbox on a card while the server can't be reached
- **THEN** the checkbox returns to its previous state and a message says the change couldn't be saved

#### Scenario: Pinned card
- **WHEN** "Groceries" is pinned
- **THEN** its card shows the pin icon, and a card of a note that isn't pinned doesn't

#### Scenario: Archived card
- **WHEN** Ann chooses "Archived" and "Old plan" is archived
- **THEN** its card is dimmed and shows the mark "Archived"

### Requirement: Notes empty states
When the user has no notes, `/notes` SHALL show "No notes yet" / "Noch keine Notizen" with a "New note" button. When the user has notes but all of them are archived, "All notes" SHALL show "All notes are archived" / "Alle Notizen sind archiviert". When the chosen category, or "Uncategorized", has no notes that aren't archived, it SHALL show "No notes in this category" / "Keine Notizen in dieser Kategorie". When no note is archived, "Archived" SHALL show "No archived notes" / "Keine archivierten Notizen".

#### Scenario: First visit
- **WHEN** Ann has no notes and opens `/notes`
- **THEN** the page shows "No notes yet" and a "New note" button

#### Scenario: Empty category
- **WHEN** Ann chooses the category "Garden", which has no notes
- **THEN** the grid shows "No notes in this category"

#### Scenario: Category with only archived notes
- **WHEN** every note of "Garden" is archived and Ann chooses "Garden"
- **THEN** the grid shows "No notes in this category"

#### Scenario: Everything archived
- **WHEN** Ann has two notes, both archived, and opens `/notes`
- **THEN** the grid shows "All notes are archived"

#### Scenario: Empty archive
- **WHEN** no note of Ann is archived and she chooses "Archived"
- **THEN** the grid shows "No archived notes"

### Requirement: Note editor page
`/notes/<id>` SHALL require a signed-in user and show a "Notes" / "Notizen" back link to `/notes`, the note's icon button, a title field with the placeholder "Title" / "Titel", the category badge, a colour button, a menu, and the content editor (see the `note-document` capability), with the note's colour as background. The menu SHALL hold, in this order, "Pin note" / "Notiz anheften" (or "Unpin note" / "Notiz lösen" for a pinned note), "Archive note" / "Notiz archivieren" (or "Unarchive note" / "Notiz wiederherstellen" for an archived note) and "Delete note" / "Notiz löschen". A pinned note SHALL show the pin icon and an archived note the mark "Archived" / "Archiviert" next to the category badge. The icon button SHALL open the icon chooser with the tabs "Own image" and "Symbol" and a "Remove icon" button, as for the link widget. The colour button SHALL offer the note colours and "No colour" / "Keine Farbe", each shown in the active theme. The document title is the note's display title followed by " - <app title>". An unknown note or one of another user SHALL show the not-found page.

#### Scenario: Change the colour
- **WHEN** Ann opens "Groceries", activates the colour button and chooses `yellow`
- **THEN** the editor background turns to the yellow note colour, and the card on `/notes` is yellow too

#### Scenario: Choose an icon
- **WHEN** Ann opens the icon chooser, switches to "Symbol" and picks "cart"
- **THEN** the icon button and the card show the cart symbol

#### Scenario: Document title
- **WHEN** Ann opens the note "Groceries" in English
- **THEN** the document title is "Groceries - Boardly"

#### Scenario: Unknown note
- **WHEN** Ann opens `/notes/doesnotexist`
- **THEN** the page shows the not-found message

#### Scenario: Menu order
- **WHEN** Ann opens the menu of a note that is neither pinned nor archived
- **THEN** it shows "Pin note", "Archive note" and "Delete note", in this order

### Requirement: Create notes
"New note" SHALL open `/notes/new`, an empty editor page whose note doesn't exist yet. The note SHALL be created on the server only when its title, content, icon, colour, category, pinned flag or archived flag is first changed; the address then changes to `/notes/<id>` without adding a history entry. Leaving `/notes/new` without a change SHALL NOT create a note. When a category is chosen in the side panel, the new note SHALL start with that category; "All notes", "Uncategorized" and "Archived" start without one, and a new note is never archived.

#### Scenario: Typing creates the note
- **WHEN** Ann activates "New note" and types "milk"
- **THEN** a note with the content "milk" is created, and the address becomes `/notes/<its id>`

#### Scenario: Abandoned new note
- **WHEN** Ann activates "New note" and goes back without typing
- **THEN** no note is created

#### Scenario: Category from the side panel
- **WHEN** Ann has chosen "Work" in the side panel and activates "New note"
- **THEN** the new note shows the badge "Work"

#### Scenario: New note from the archive
- **WHEN** Ann has chosen "Archived" in the side panel, activates "New note" and types "milk"
- **THEN** the new note has no category and isn't archived

### Requirement: Autosave
The editor page SHALL save changes automatically 1 second after the last change, and immediately when the user leaves the page. It SHALL show "Saving…" / "Wird gespeichert…" while a save is running, "Saved" / "Gespeichert" after it succeeded, and "Not saved" / "Nicht gespeichert" with a "Retry" / "Erneut versuchen" button after it failed. While changes are unsaved or a save is running, closing or reloading the browser tab SHALL ask for confirmation. When a save gets `version_conflict`, the editor SHALL stop saving, become read-only and say that the note was changed elsewhere, with a "Reload" / "Neu laden" button that loads the current note.

#### Scenario: Pause saves
- **WHEN** Ann types "eggs" and stops typing
- **THEN** about a second later the note is saved and the page shows "Saved"

#### Scenario: Server unavailable
- **WHEN** a save fails because the server can't be reached
- **THEN** the page shows "Not saved" and "Retry", and Ann's text stays in the editor

#### Scenario: Changed in another tab
- **WHEN** Ann has "Groceries" open in two tabs, edits it in the first and then types in the second
- **THEN** the second tab says the note was changed elsewhere, stops saving, and "Reload" shows the first tab's version

### Requirement: Delete notes
"Delete note" SHALL ask for confirmation, naming the note's display title. Confirming SHALL delete the note and show `/notes`. There is no way to restore a deleted note.

#### Scenario: Delete
- **WHEN** Ann chooses "Delete note" on "Groceries" and confirms
- **THEN** the browser shows `/notes`, and "Groceries" is no longer listed

#### Scenario: Cancel
- **WHEN** Ann chooses "Delete note" and cancels
- **THEN** the note is kept and the editor stays open

### Requirement: Inline editing on cards
Activating the title or content of a card on `/notes` with a click, a tap, or Enter when the card has focus SHALL turn the card into an inline editor without leaving `/notes`. The inline editor SHALL show a title field with the placeholder "Title" / "Titel" and the content editor with its Markdown shortcuts, keyboard shortcuts and task checkboxes (see "Content editor" and "Task items in the editor" in the `note-document` capability) but without the toolbar, and SHALL focus the title field when the title was activated, and the end of the content otherwise. While editing, the card SHALL grow to show all of its content instead of fading out. Only one card SHALL be in inline editing at a time. Escape, activating another card, or moving the focus out of the card SHALL end inline editing and show the card again. Changes SHALL be saved as described in "Autosave": 1 second after the last change and immediately when inline editing ends or the page is left, with the save status shown on the card. When a save gets `version_conflict`, inline editing SHALL end, the card SHALL show the current note, and a message SHALL say that the note was changed elsewhere.

Notes changed on `/notes`, whether by inline editing, a checkbox or the category badge, SHALL keep their place in the grid while the page stays open; the order by time of last change applies again the next time `/notes` is opened.

#### Scenario: Edit text in place
- **WHEN** Ann clicks the content of the card "Groceries", types "eggs" at the end and presses Escape
- **THEN** the browser still shows `/notes`, the card shows "eggs", and the saved note contains "eggs"

#### Scenario: Edit the title in place
- **WHEN** Ann clicks the content of an untitled card and types "Shopping" into the title field
- **THEN** after the autosave the note has the title "Shopping", and the card shows it once editing ends

#### Scenario: Markdown shortcut inline
- **WHEN** Ann edits a card inline and types `- [ ] bread` on an empty line
- **THEN** the line becomes an unchecked task item "bread"

#### Scenario: No toolbar
- **WHEN** Ann edits a card inline
- **THEN** the card shows no formatting toolbar, and Ctrl/Cmd+B still makes the selected text bold

#### Scenario: Card keeps its place
- **WHEN** "Groceries" is the third card and Ann edits it inline and ends editing
- **THEN** "Groceries" is still the third card, and after reopening `/notes` it is the first

#### Scenario: Changed elsewhere
- **WHEN** Ann edits "Groceries" inline while it was changed in another tab
- **THEN** inline editing ends, the card shows the other tab's version, and a message says the note was changed elsewhere

### Requirement: Pin and archive notes
Choosing "Pin note" or "Unpin note" in the note menu SHALL pin or unpin the note, and "Archive note" or "Unarchive note" SHALL archive or unarchive it, without asking for confirmation. The change SHALL be saved right away together with any unsaved changes of the editor, with the same save status, retry and `version_conflict` handling as in "Autosave", and the editor stays open. The menu, the pin icon and the archived mark SHALL show the new state at once. In the note editor modal of dashboards (see the `note-widget` capability) the menu offers the same actions, and the widgets show the new state after the modal saved.

#### Scenario: Pin from the editor
- **WHEN** Ann opens "Groceries", chooses "Pin note" and goes back to `/notes`
- **THEN** the editor showed the pin icon at once, the menu then offered "Unpin note", and "Groceries" is the first card on `/notes`

#### Scenario: Archive from the editor
- **WHEN** Ann opens "Old plan", chooses "Archive note" and goes back to `/notes`
- **THEN** "Old plan" isn't shown with "All notes" and is shown with "Archived"

#### Scenario: Unarchive
- **WHEN** Ann chooses "Archived", opens "Old plan" and chooses "Unarchive note"
- **THEN** "Old plan" is no longer archived and is shown again with "All notes"

#### Scenario: Changed elsewhere
- **WHEN** Ann chooses "Pin note" while the note was changed in another tab
- **THEN** the editor says that the note was changed elsewhere and becomes read-only, and the note isn't pinned
