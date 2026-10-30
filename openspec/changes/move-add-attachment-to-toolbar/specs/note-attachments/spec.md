## MODIFIED Requirements

### Requirement: Attachments button
The note editor page and the note editor modal SHALL show an attachments button with a paperclip
icon, the accessible name "Attachments" / "Anhänge" and the number of the note's attachments when
there are any. It SHALL open a dropdown that lists the note's attachments, each once, in the order
they first appear in the content, with a small preview for images or an icon for other files, the
name and the size, and nothing else. Activating an image in the list SHALL open the image viewer,
and activating another file SHALL open its URL in a new tab. The dropdown SHALL show "No
attachments" / "Keine Anhänge" when the note has none.

#### Scenario: List
- **WHEN** "Garden" contains `roses.jpg` twice and `plan.pdf` once, and Ann opens the dropdown
- **THEN** it lists `roses.jpg` and `plan.pdf`, once each, and the button shows 2

#### Scenario: Only files
- **WHEN** Ann opens the dropdown of "Garden"
- **THEN** it offers no action other than opening the listed files

#### Scenario: Remove from note
- **WHEN** Ann deletes the file card of `plan.pdf` from the content of "Garden"
- **THEN** the dropdown no longer lists `plan.pdf`, the change is saved, and `plan.pdf` still exists on `/attachments`

### Requirement: Adding attachments
The content editor's toolbar SHALL offer an "Add attachment" / "Anhang hinzufügen" button right
after the link button. It SHALL open the device's file picker, allowing several files and offering
the allowed types (see "Attachments API"). Files can also be
pasted into the content editor or dropped onto it, wherever the content editor is shown. Each file
SHALL be uploaded and an attachment block for it inserted, in the order of the files: at the drop
position for dropped files, otherwise at the cursor when the content editor has or last had one,
and at the end of the note when it never had one. While a file is uploading, a placeholder with its
name SHALL show where it will appear. When an upload fails, its placeholder SHALL be removed and a
message SHALL name the file and say that it is too large (with the maximum size), that its type
isn't allowed, or that it couldn't be uploaded. Inserting attachments counts as a change and is
saved automatically.

#### Scenario: Insert at the cursor
- **WHEN** Ann puts the cursor between two paragraphs of "Garden" and adds `roses.jpg` with the toolbar's "Add attachment"
- **THEN** the photo appears between the two paragraphs, and after the autosave the note's content contains its attachment block there

#### Scenario: Insert at the end
- **WHEN** Ann opens "Garden" and adds two files with the toolbar's "Add attachment" without placing the cursor
- **THEN** both appear at the end of the note, in the order she chose them

#### Scenario: Paste a screenshot
- **WHEN** Ann pastes a screenshot into the content editor
- **THEN** it is uploaded and shown as an image at the cursor

#### Scenario: Upload too large
- **WHEN** Ann drops a 20 MB video onto the content and the maximum is 5 MB
- **THEN** no block is inserted and a message says the file is larger than 5 MB
