## MODIFIED Requirements

### Requirement: Notes API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/notes` lists all of the user's notes, archived ones included, with all their fields, ordered by time of last change, newest first, then by ID.
- `GET /api/notes/{id}` returns one note.
- `POST /api/notes` creates a note from `title`, `icon`, `color`, `categoryId`, `pinned`, `archived` and either `content` or `text` (all optional; a missing content is an empty document, and a missing flag is `false`) and responds 201 with the note. `text` is plain text that the server turns into the content (see "Content from plain text" in the note-document spec). A request with both `content` and `text`, or a `text` that isn't a string, gets 400 `invalid_request`.
- `PATCH /api/notes/{id}` changes the given fields of a note. The body SHALL contain the `version` the change is based on; if it isn't the note's current version, the server responds 409 with `version_conflict` and changes nothing. Otherwise it responds 200 with the changed note.
- `DELETE /api/notes/{id}` deletes the note and responds 204. Its attachments are not deleted (see "Attachment references" in the `note-attachments` capability).

A note is returned with `pinned` and `archived` as booleans besides its other fields, and with `attachments`: the attachments its content contains, each once, in the order they first appear, as `{ "id", "name", "contentType", "size", "url" }`. A note of another user, or one that doesn't exist, gets 404 `not_found`. An icon is `null`, `{"type": "symbol", "name": "<symbol>"}` or `{"type": "image", "imageId": "<id>"}`. An unknown symbol, an image that isn't an upload of the same user, an unknown colour, a category that isn't one of the user's, or a `pinned` or `archived` that isn't a boolean gets 400 `invalid_note`. An invalid content document, or a `text` whose document would be larger than allowed, gets 400 `invalid_document`.

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

#### Scenario: Note with attachments
- **WHEN** "Garden" contains `roses.jpg`, then `plan.pdf`, then `roses.jpg` again
- **THEN** `GET /api/notes/{id}` returns `attachments` with `roses.jpg` and `plan.pdf`, in this order

### Requirement: Note editor page
`/notes/<id>` SHALL require a signed-in user and show a "Notes" / "Notizen" back link to `/notes`, the note's icon button, a title field with the placeholder "Title" / "Titel", the category badge, a colour button, the attachments button (see "Attachments button" in the `note-attachments` capability), a menu, and the content editor (see the `note-document` capability), with the note's colour as background. The menu SHALL hold, in this order, "Pin note" / "Notiz anheften" (or "Unpin note" / "Notiz lösen" for a pinned note), "Archive note" / "Notiz archivieren" (or "Unarchive note" / "Notiz wiederherstellen" for an archived note) and "Delete note" / "Notiz löschen". A pinned note SHALL show the pin icon and an archived note the mark "Archived" / "Archiviert" next to the category badge. The icon button SHALL open the icon chooser with the tabs "Own image" and "Symbol" and a "Remove icon" button, as for the link widget. The colour button SHALL offer the note colours and "No colour" / "Keine Farbe", each shown in the active theme. The document title is the note's display title followed by " - <app title>". An unknown note or one of another user SHALL show the not-found page.

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

#### Scenario: Attachments button
- **WHEN** Ann opens "Garden", which contains two attachments
- **THEN** the header shows the paperclip button with the number 2 next to the colour button
