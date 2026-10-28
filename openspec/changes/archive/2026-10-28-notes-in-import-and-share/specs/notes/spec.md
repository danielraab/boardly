# Spec Delta

## MODIFIED Requirements

### Requirement: Notes API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/notes` lists all of the user's notes with all their fields, ordered by time of last change, newest first, then by ID.
- `GET /api/notes/{id}` returns one note.
- `POST /api/notes` creates a note from `title`, `icon`, `color`, `categoryId` and either `content` or `text` (all optional; a missing content is an empty document) and responds 201 with the note. `text` is plain text that the server turns into the content (see "Content from plain text" in the note-document spec). A request with both `content` and `text`, or a `text` that isn't a string, gets 400 `invalid_request`.
- `PATCH /api/notes/{id}` changes the given fields of a note. The body SHALL contain the `version` the change is based on; if it isn't the note's current version, the server responds 409 with `version_conflict` and changes nothing. Otherwise it responds 200 with the changed note.
- `DELETE /api/notes/{id}` deletes the note and responds 204.

A note of another user, or one that doesn't exist, gets 404 `not_found`. An icon is `null`, `{"type": "symbol", "name": "<symbol>"}` or `{"type": "image", "imageId": "<id>"}`. An unknown symbol, an image that isn't an upload of the same user, an unknown colour, or a category that isn't one of the user's gets 400 `invalid_note`. An invalid content document, or a `text` whose document would be larger than allowed, gets 400 `invalid_document`.

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
