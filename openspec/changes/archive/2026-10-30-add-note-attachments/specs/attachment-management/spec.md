## Purpose

Gives a signed-in user an overview of all their attachments on the `/attachments` page: which notes
use them, which are marked for deletion, and keeping or deleting them.

## ADDED Requirements

### Requirement: Attachments page
`/attachments` SHALL require a signed-in user and show the heading "Attachments" / "Anhänge", the
number of the user's attachments and their total size, filters, and the attachments, newest first.
Signed-out visitors SHALL be sent to `/login`. The document title is "Attachments - <app title>".

#### Scenario: Open the page
- **WHEN** Ann has 3 attachments of together 2.5 MB and opens `/attachments`
- **THEN** the page shows "Attachments", "3 attachments · 2.5 MB" and the three attachments, newest first

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/attachments`
- **THEN** the browser shows `/login`

### Requirement: Attachment entries
Each attachment SHALL be shown with a small preview for images or an icon for its kind, its name,
its type, its size, its upload date and the notes that refer to it. Each note SHALL be a link to
`/notes/<id>` with the note's display title (see "Display title" in the `notes` capability), and an
archived note SHALL show the mark "Archived" / "Archiviert". An attachment without notes SHALL show
"Not used in any note" / "In keiner Notiz verwendet". A marked attachment that isn't kept SHALL show
"Will be deleted on <date>" / "Wird am <Datum> gelöscht", and a kept attachment "Kept" / "Behalten",
with "marked on <date>" / "markiert am <Datum>" when it has a deletion mark. Activating the name of
an image SHALL open the image viewer (see the `note-attachments` capability) and the name of another
file SHALL open its URL in a new tab. On wide screens each entry SHALL be one row; at phone width the entries SHALL be stacked
cards, without horizontal scrolling.

#### Scenario: Note link
- **WHEN** `report.pdf` is in "Taxes" and Ann activates "Taxes" in its entry
- **THEN** the browser shows `/notes/<id of Taxes>`

#### Scenario: Marked attachment
- **WHEN** `old.png` was marked on 1 October and the deletion period is 30 days
- **THEN** its entry shows "Not used in any note" and "Will be deleted on 31 October"

#### Scenario: Kept attachment
- **WHEN** Ann kept `old.png` after it was marked on 1 October
- **THEN** its entry shows "Kept" and "marked on 1 October"

### Requirement: Attachment filters
The page SHALL offer the filters "All" / "Alle", "In use" / "Verwendet" (at least one note refers to
it), "Unused" / "Unbenutzt" (no note refers to it), "Marked for deletion" / "Zum Löschen markiert"
(marked and not kept) and "Kept" / "Behalten", each with its number of attachments. The filter SHALL
be part of the address (`/attachments?filter=in-use`, `unused`, `marked`, `kept`), and "All" is the
default. When the chosen filter has no attachments, the page SHALL say so, and when the user has no
attachments at all it SHALL show "No attachments yet" / "Noch keine Anhänge".

#### Scenario: Unused
- **WHEN** Ann chooses "Unused"
- **THEN** the page shows only attachments that no note refers to, and the address is `/attachments?filter=unused`

#### Scenario: Unknown filter
- **WHEN** Ann opens `/attachments?filter=nonsense`
- **THEN** the page shows all attachments and "All" is chosen

### Requirement: Keep and delete attachments
Each entry SHALL have a "Keep" / "Behalten" action, or "Don't keep" / "Nicht behalten" for a kept
attachment, which sets or removes the keep flag and shows the result at once. Each entry SHALL have
a "Delete" / "Löschen" action that is only enabled when no note refers to the attachment; while
notes refer to it, the disabled action SHALL explain that the attachment is still used in notes.
Deleting SHALL ask for confirmation naming the attachment, then delete it and remove it from the
page. When a request fails, the entry SHALL show the attachment as it is on the server again and a
message SHALL say that the change couldn't be saved.

#### Scenario: Keep a marked attachment
- **WHEN** Ann activates "Keep" on the marked `old.png`
- **THEN** the entry shows "Kept", and `old.png` is not deleted when its deletion date passes

#### Scenario: Delete an unused attachment
- **WHEN** Ann activates "Delete" on the unused `old.png` and confirms
- **THEN** `old.png` disappears from the page and its URL responds 404

#### Scenario: Attachment in use
- **WHEN** `report.pdf` is in "Taxes"
- **THEN** its "Delete" action is disabled and says that it is still used in notes, and "Taxes" is shown as a link
