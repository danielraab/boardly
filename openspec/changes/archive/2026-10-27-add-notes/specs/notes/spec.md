# Spec Delta

## Purpose

Lets a signed-in user keep notes with rich content and checkable todos: the note entity, its API, the card grid on `/notes` and the editor page with autosave.

## ADDED Requirements

### Requirement: Note entity
A note SHALL belong to one user and have an optional title (at most 200 characters after trimming; empty means none), an optional icon, an optional colour, at most one category of the same user (see the `note-categories` capability), a content document (see the `note-document` capability), a version, a creation time and a time of last change. The icon is either a symbol from the same symbol set as the link widget or an image the user uploaded (see the `images` capability). The colour is one of the note colours `red`, `orange`, `yellow`, `green`, `teal`, `blue`, `purple`, `pink` and `grey` (see "Note colour palette" in the `theming` capability). The version starts at 1 and goes up by one with every change. Every change to the title, icon, colour, category or content SHALL set the time of last change; a category becoming unset because it was deleted SHALL NOT.

#### Scenario: Minimal note
- **WHEN** Ann creates a note with only the content "call the plumber"
- **THEN** the note has no title, no icon, no colour, no category and version 1

#### Scenario: Title too long
- **WHEN** a note is saved with a title of 201 characters
- **THEN** the server responds 400 with `invalid_note`

#### Scenario: Version goes up
- **WHEN** Ann changes the colour of a note with version 4
- **THEN** the note has version 5 and its time of last change is now

### Requirement: Notes API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/notes` lists all of the user's notes with all their fields, ordered by time of last change, newest first, then by ID.
- `GET /api/notes/{id}` returns one note.
- `POST /api/notes` creates a note from `title`, `icon`, `color`, `categoryId` and `content` (all optional; a missing content is an empty document) and responds 201 with the note.
- `PATCH /api/notes/{id}` changes the given fields of a note. The body SHALL contain the `version` the change is based on; if it isn't the note's current version, the server responds 409 with `version_conflict` and changes nothing. Otherwise it responds 200 with the changed note.
- `DELETE /api/notes/{id}` deletes the note and responds 204.

A note of another user, or one that doesn't exist, gets 404 `not_found`. An icon is `null`, `{"type": "symbol", "name": "<symbol>"}` or `{"type": "image", "imageId": "<id>"}`. An unknown symbol, an image that isn't an upload of the same user, an unknown colour, or a category that isn't one of the user's gets 400 `invalid_note`. An invalid content document gets 400 `invalid_document`.

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

### Requirement: Display title
Where a note's title is shown and the note has no title, the web app SHALL show the first line of text of its content instead, cut off with "…" after 80 characters, and "Untitled note" / "Unbenannte Notiz" when the content has no text either.

#### Scenario: Untitled note with content
- **WHEN** a note has no title and its content starts with the paragraph "call the plumber tomorrow"
- **THEN** its display title is "call the plumber tomorrow"

#### Scenario: Empty note
- **WHEN** a note has no title and no text
- **THEN** its display title is "Untitled note"

### Requirement: Notes page
`/notes` SHALL require a signed-in user and show the heading "Notes" / "Notizen", a "New note" / "Neue Notiz" button, the category side panel (see the `note-categories` capability) and the user's notes as a grid of cards, ordered by time of last change, newest first. The number of columns SHALL follow the available width, with one column at phone width. The grid SHALL show only the notes of the category chosen in the side panel. Activating a card SHALL open `/notes/<id>`. Signed-out visitors SHALL be sent to `/login`. The document title is "Notes - <app title>".

#### Scenario: Open a note
- **WHEN** Ann activates the card "Groceries" on `/notes`
- **THEN** the browser shows `/notes/<id of Groceries>`

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/notes`
- **THEN** the browser shows `/login`

#### Scenario: Phone width
- **WHEN** `/notes` is viewed 375 px wide
- **THEN** the cards are shown in one column

### Requirement: Note cards
A note card SHALL show the note's icon, if any, and its title, if any, followed by its rendered content, with task items shown as checkboxes in their state. Content that doesn't fit into the card's maximum height is cut off and fades out. A note with a colour SHALL have that colour as the card's background; otherwise the card uses `surface`. A note with a category SHALL show the category badge (see "Category badge" in the `note-categories` capability) at the bottom of the card. Links in the card's content and checkboxes on the card SHALL NOT react to activation; activating them opens the note like the rest of the card.

#### Scenario: Card with colour and category
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** its card has the green note background, shows the cart symbol and "Groceries", its content, and the badge "Home"

#### Scenario: Long note
- **WHEN** a note has 60 lines of content
- **THEN** its card shows the first lines and fades out at its maximum height

#### Scenario: Untitled card
- **WHEN** a note has no title
- **THEN** its card shows no title line and starts with its content

### Requirement: Notes empty states
When the user has no notes, `/notes` SHALL show "No notes yet" / "Noch keine Notizen" with a "New note" button. When the chosen category has no notes, it SHALL show "No notes in this category" / "Keine Notizen in dieser Kategorie".

#### Scenario: First visit
- **WHEN** Ann has no notes and opens `/notes`
- **THEN** the page shows "No notes yet" and a "New note" button

#### Scenario: Empty category
- **WHEN** Ann chooses the category "Garden", which has no notes
- **THEN** the grid shows "No notes in this category"

### Requirement: Note editor page
`/notes/<id>` SHALL require a signed-in user and show a "Notes" / "Notizen" back link to `/notes`, the note's icon button, a title field with the placeholder "Title" / "Titel", the category badge, a colour button, a menu with "Delete note" / "Notiz löschen", and the content editor (see the `note-document` capability), with the note's colour as background. The icon button SHALL open the icon chooser with the tabs "Own image" and "Symbol" and a "Remove icon" button, as for the link widget. The colour button SHALL offer the note colours and "No colour" / "Keine Farbe", each shown in the active theme. The document title is the note's display title followed by " - <app title>". An unknown note or one of another user SHALL show the not-found page.

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

### Requirement: Create notes
"New note" SHALL open `/notes/new`, an empty editor page whose note doesn't exist yet. The note SHALL be created on the server only when its title, content, icon, colour or category is first changed; the address then changes to `/notes/<id>` without adding a history entry. Leaving `/notes/new` without a change SHALL NOT create a note. When a category is chosen in the side panel, the new note SHALL start with that category; "All notes" and "Uncategorized" start without one.

#### Scenario: Typing creates the note
- **WHEN** Ann activates "New note" and types "milk"
- **THEN** a note with the content "milk" is created, and the address becomes `/notes/<its id>`

#### Scenario: Abandoned new note
- **WHEN** Ann activates "New note" and goes back without typing
- **THEN** no note is created

#### Scenario: Category from the side panel
- **WHEN** Ann has chosen "Work" in the side panel and activates "New note"
- **THEN** the new note shows the badge "Work"

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
