## Purpose

Lets a user attach files to notes: storing and serving the files, the types that are allowed, which
notes refer to which file, when unused files are marked and deleted, and how attachments are added
and shown in notes.

## ADDED Requirements

### Requirement: Attachment storage
The server SHALL store each attachment as a file under `<data dir>/attachments/` and keep its ID,
owner, name, content type, size, creation time, a deletion mark (`deleted_at`, empty unless the
attachment is marked for deletion) and a keep flag (off for a new attachment). Attachment IDs SHALL
be random and unguessable. Attachments are separate from images (see the `images` capability): they
are not favicons, previews or icons, and the image cleanup doesn't touch them. Deleting a user
deletes their attachments.

#### Scenario: Stored on disk
- **WHEN** Ann uploads `report.pdf`
- **THEN** a file for it exists under `<data dir>/attachments/`, and it is still served after a restart

#### Scenario: Not an image
- **WHEN** Ann uploads a PNG as an attachment
- **THEN** it is not listed or cleaned up as an image, and its URL is under `/api/attachments/`

### Requirement: Known attachment types
The server SHALL know this fixed set of attachment types. Each has a content type, its file name
extensions, how it is recognised (by content or by extension) and how it is served (inline or as a
download):

| Content type | Extensions | Recognised by | Served |
|---|---|---|---|
| `image/png` | png | content | inline |
| `image/jpeg` | jpg, jpeg | content | inline |
| `image/gif` | gif | content | inline |
| `image/webp` | webp | content | inline |
| `image/avif` | avif | content | inline |
| `image/bmp` | bmp | content | inline |
| `image/vnd.microsoft.icon` | ico | content | inline |
| `image/svg+xml` | svg | extension | download |
| `application/pdf` | pdf | content | inline |
| `audio/mpeg` | mp3 | content | inline |
| `audio/ogg` | ogg, oga | content | inline |
| `audio/wav` | wav | content | inline |
| `video/mp4` | mp4 | content | inline |
| `video/webm` | webm | content | inline |
| `text/plain` | txt | extension | download |
| `text/markdown` | md | extension | download |
| `text/csv` | csv | extension | download |
| `application/json` | json | extension | download |
| `text/html` | html, htm | extension | download |
| `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | docx | extension | download |
| `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` | xlsx | extension | download |
| `application/vnd.openxmlformats-officedocument.presentationml.presentation` | pptx | extension | download |
| `application/vnd.oasis.opendocument.text` | odt | extension | download |
| `application/vnd.oasis.opendocument.spreadsheet` | ods | extension | download |
| `application/vnd.oasis.opendocument.presentation` | odp | extension | download |
| `application/zip` | zip | extension | download |
| `application/x-7z-compressed` | 7z | extension | download |
| `application/x-tar` | tar | extension | download |
| `application/gzip` | gz, tgz | extension | download |

An uploaded file's type SHALL be determined like this: when its content is recognised as one of the
types recognised by content, that is its type, whatever its name. Otherwise its type comes from the
extension of its name, ignoring case. A file whose extension belongs to a type recognised by content,
but whose content isn't recognised as that type, has no known type. Images are the `image/*` types,
SVG included; audio and video are the `audio/*` and `video/*` types.

#### Scenario: Recognised by content
- **WHEN** Ann uploads a PNG file named `scan.txt`
- **THEN** the attachment's type is `image/png`

#### Scenario: Recognised by extension
- **WHEN** Ann uploads a Word document named `Offer.DOCX`
- **THEN** the attachment's type is the Word document type

#### Scenario: Fake image
- **WHEN** Ann uploads a text file named `photo.jpg`
- **THEN** the file has no known type

### Requirement: Attachment configuration
The server SHALL read these settings:
- `BOARDLY_ATTACHMENT_MAX_SIZE_MB`: the largest attachment in megabytes, a positive whole number,
  default `5`.
- `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS`: how many days a marked attachment is kept before it is
  deleted, a positive whole number, default `30`.
- `BOARDLY_ATTACHMENT_TYPES`: a comma-separated list of the allowed types out of the known types,
  each a content type or `<group>/*` for every known type of that group (for example `image/*`),
  with spaces around entries ignored. When it isn't set or is empty, the allowed types are all
  images except `image/svg+xml`, and `application/pdf`.

The server SHALL refuse to start with a message naming the variable when a value is invalid,
including a type that isn't known or a group without known types. `.env.example` SHALL list all
three variables with their defaults, and every known type, marking `image/svg+xml` and `text/html`
as risky to allow.

#### Scenario: Default types
- **WHEN** `BOARDLY_ATTACHMENT_TYPES` isn't set and Ann uploads an SVG file
- **THEN** the server responds 400 with `unsupported_attachment_type`

#### Scenario: Group
- **WHEN** `BOARDLY_ATTACHMENT_TYPES=image/*,video/*` and Ann uploads an MP4 video
- **THEN** the upload succeeds

#### Scenario: Unknown type
- **WHEN** `BOARDLY_ATTACHMENT_TYPES=application/x-msdownload`
- **THEN** the server doesn't start and names `BOARDLY_ATTACHMENT_TYPES` in its message

### Requirement: Uploading attachments
`POST /api/attachments?name=<file name>` SHALL accept one file from a signed-in user, sent as the
raw request body, and respond 201 with the attachment (see "Attachments API"). The name SHALL be
1 to 255 characters after trimming and SHALL NOT contain `/`, `\` or control characters; otherwise
the server responds 400 `invalid_attachment_name`. An empty file, or one whose type is unknown or
not allowed, gets 400 `unsupported_attachment_type`. A file larger than the configured maximum gets
413 `attachment_too_large`, and the server stops reading it at that size. A request without a
session gets 401.

#### Scenario: Upload a PDF
- **WHEN** Ann uploads the 300 KB file `report.pdf`
- **THEN** the server responds 201 with an attachment named `report.pdf` of type `application/pdf` and size 300 KB, not marked and not kept

#### Scenario: Too large
- **WHEN** the maximum is 5 MB and Ann uploads a 6 MB JPEG
- **THEN** the server responds 413 with `attachment_too_large` and stores nothing

#### Scenario: Not allowed
- **WHEN** the allowed types are the default and Ann uploads `notes.zip`
- **THEN** the server responds 400 with `unsupported_attachment_type`

#### Scenario: Path in the name
- **WHEN** Ann uploads a file with the name `../etc/passwd`
- **THEN** the server responds 400 with `invalid_attachment_name`

### Requirement: Serving attachments
`GET /api/attachments/{id}/{name}` SHALL respond with the attachment's bytes and stored content type
to anyone, with or without a session; `{name}` may be any text and doesn't have to match. The
response SHALL have `X-Content-Type-Options: nosniff` and `Content-Security-Policy: sandbox`, SHALL
be cacheable for a year as immutable, and SHALL support range requests. Types served inline SHALL
have `Content-Disposition: inline`, all others `Content-Disposition: attachment`, both with the
attachment's name as the file name. Attachments marked for deletion SHALL still be served. An
unknown or deleted attachment gets 404 `not_found`.

#### Scenario: Visitor loads an image
- **WHEN** a signed-out visitor requests the URL of one of Ann's image attachments
- **THEN** the server responds 200 with the image, its content type and an inline disposition

#### Scenario: HTML is downloaded
- **WHEN** `text/html` is allowed and someone opens the URL of an HTML attachment
- **THEN** the response has `Content-Disposition: attachment` and the sandbox policy, so the browser downloads it instead of running it

#### Scenario: Marked attachment
- **WHEN** an attachment is marked for deletion
- **THEN** its URL still responds 200

#### Scenario: Deleted attachment
- **WHEN** an attachment was deleted
- **THEN** its URL responds 404 with `not_found`

### Requirement: Attachment references
A note SHALL refer to an attachment exactly when its content contains an attachment block with that
attachment's ID (see the `note-document` capability). The server SHALL update a note's references
in the same transaction as every change of its content, whatever made the change (web app, connected
app, import), and remove them when the note is deleted. Archived notes SHALL keep their references.
When a note's content is saved with an attachment that is marked for deletion, the mark SHALL be
removed.

#### Scenario: Removed from the content
- **WHEN** Ann removes the only attachment block of `report.pdf` from "Taxes" and the note is saved
- **THEN** "Taxes" no longer refers to `report.pdf`, and the attachment still exists

#### Scenario: Note deleted
- **WHEN** Ann deletes "Taxes", which contains `report.pdf`
- **THEN** the attachment still exists and no note refers to it

#### Scenario: Archived note
- **WHEN** "Taxes" is archived
- **THEN** it still refers to `report.pdf`, and the attachment is not marked for deletion

#### Scenario: Referred to again
- **WHEN** `report.pdf` is marked for deletion and Ann pastes its block into "Taxes 2026" and the note is saved
- **THEN** the mark is removed

### Requirement: Marking and deleting unused attachments
The server SHALL, on start and then once an hour:
- mark every attachment for deletion, by setting its deletion mark to the current time, that isn't
  marked, isn't kept, was created at least 24 hours ago and that no note refers to;
- delete, file and record, every attachment that isn't kept, that no note refers to and whose
  deletion mark is at least `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` days old;
- delete files under `<data dir>/attachments/` that have no record.

Its deletion date is its deletion mark plus `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` days. A kept
attachment keeps its deletion mark, as information, and is never deleted by the server.

#### Scenario: Recent upload
- **WHEN** Ann uploaded a file 2 hours ago and hasn't saved it into a note yet
- **THEN** it isn't marked

#### Scenario: Unused upload
- **WHEN** Ann uploaded a file 25 hours ago and no note refers to it
- **THEN** it is marked for deletion, and its deletion date is 30 days later

#### Scenario: Deleted after the period
- **WHEN** an attachment was marked 30 days ago, isn't kept and no note refers to it
- **THEN** it is deleted and its URL responds 404

#### Scenario: Kept
- **WHEN** an attachment was marked 40 days ago and Ann set "Keep" on it
- **THEN** it isn't deleted and still shows the date it was marked

### Requirement: Attachments API
The server SHALL offer these endpoints to the signed-in owner, and respond 401 without a session:
- `GET /api/attachments` responds `{ "attachments": [...], "maxSize": <bytes>, "allowedTypes":
  [<content type>, ...] }` with all of the user's attachments, newest first, then by ID, and the
  configured limits, so the web app can tell the user about them.
- `PATCH /api/attachments/{id}` with `{"keep": true|false}` sets or removes the keep flag and responds
  200 with the attachment. Removing the keep flag also removes the deletion mark, so the cleanup can
  mark the attachment again later. A `keep` that isn't a boolean gets 400 `invalid_request`.
- `DELETE /api/attachments/{id}` deletes the attachment, file and record, and responds 204. When a
  note refers to it, the server responds 409 `attachment_in_use` and deletes nothing.

An attachment is returned as `{ "id", "name", "contentType", "size", "url", "createdAt",
"deletedAt", "deleteAt", "keep", "noteIds" }`: `url` is its path `/api/attachments/{id}/{name}`,
`deletedAt` the deletion mark or `null`, `deleteAt` the deletion date or `null` when it isn't marked
or is kept, and `noteIds` the IDs of the notes that refer to it. An attachment of another user, or
one that doesn't exist, gets 404 `not_found`.

#### Scenario: Keep
- **WHEN** Ann sends `PATCH /api/attachments/{id}` with `{"keep": true}` for a marked attachment
- **THEN** the server responds 200 with `keep` `true`, `deletedAt` unchanged and `deleteAt` `null`

#### Scenario: Don't keep
- **WHEN** Ann sends `{"keep": false}` for a kept attachment that was marked
- **THEN** the server responds 200 with `keep` `false` and `deletedAt` `null`

#### Scenario: Delete an attachment in use
- **WHEN** Ann deletes `report.pdf` while "Taxes" refers to it
- **THEN** the server responds 409 with `attachment_in_use`, and the attachment still exists

#### Scenario: Delete an unused attachment
- **WHEN** Ann deletes an attachment that no note refers to
- **THEN** the server responds 204, and its URL responds 404

#### Scenario: Other user's attachment
- **WHEN** Bob sends `DELETE /api/attachments/{id}` for one of Ann's attachments
- **THEN** the server responds 404 with `not_found`

### Requirement: Attachments in shown notes
Wherever a note's content is shown (the editor page and the editor modal, inline editing on cards,
note cards and note widgets), an attachment block SHALL be shown as its own block:
- an image as the image, at most as wide as the content and at most as wide as the image itself,
  with the attachment's name as its alternative text;
- audio and video with the browser's player;
- any other file as a file card with an icon for its kind, its name, its size and a download link.

Activating an image SHALL open the image viewer, except on note cards, where activating content
starts inline editing as before (see "Inline editing on cards" in the `notes` capability). The
download link of a file card SHALL download the file in the editors and in note widgets, also for
visitors of public dashboards. An attachment block whose attachment doesn't exist any more SHALL be
shown as a file card "Attachment missing" / "Anhang fehlt" without a link.

#### Scenario: Image in a note
- **WHEN** "Garden" contains the attachment `roses.jpg`
- **THEN** the card, the editor and a note widget of "Garden" show the photo inside the content

#### Scenario: PDF in a note
- **WHEN** "Taxes" contains `report.pdf` of 300 KB
- **THEN** the note shows a file card "report.pdf · 300 KB" with a download link

#### Scenario: Visitor downloads
- **WHEN** a visitor of Ann's public dashboard activates the download link of `report.pdf` in a note widget
- **THEN** the browser downloads the file

### Requirement: Image viewer
The image viewer SHALL show the image as large as fits the window, its name, a download button, and
"Previous" / "Zurück" and "Next" / "Weiter" buttons that move between the images of the same note in
content order when it has more than one. The arrow keys SHALL move between images, and Escape, the
close button or a click outside the image SHALL close the viewer and return focus to the image that
opened it.

#### Scenario: Open an image
- **WHEN** Ann clicks the photo `roses.jpg` in the editor of "Garden"
- **THEN** the viewer shows `roses.jpg` large with its name and a download button

#### Scenario: Next image
- **WHEN** "Garden" has the images `roses.jpg` and `tulips.jpg` and Ann presses the right arrow key in the viewer of `roses.jpg`
- **THEN** the viewer shows `tulips.jpg`

### Requirement: Attachments button
The note editor page and the note editor modal SHALL show an attachments button with a paperclip
icon, the accessible name "Attachments" / "Anhänge" and the number of the note's attachments when
there are any. It SHALL open a dropdown that lists the note's attachments, each once, in the order
they first appear in the content, with a small preview for images or an icon for other files, the
name and the size. Activating an image in the list SHALL open the image viewer, and activating
another file SHALL open its URL in a new tab. Each entry SHALL have a "Remove from note" / "Aus
Notiz entfernen" action that removes every block of that attachment from the content; the
attachment itself stays. The dropdown SHALL end with "Add attachment" / "Anhang hinzufügen", and
show "No attachments" / "Keine Anhänge" above it when the note has none.

#### Scenario: List
- **WHEN** "Garden" contains `roses.jpg` twice and `plan.pdf` once, and Ann opens the dropdown
- **THEN** it lists `roses.jpg` and `plan.pdf`, once each, and the button shows 2

#### Scenario: Remove from note
- **WHEN** Ann chooses "Remove from note" for `plan.pdf`
- **THEN** the file card disappears from the content, the change is saved, and `plan.pdf` still exists on `/attachments`

### Requirement: Adding attachments
"Add attachment" SHALL open the device's file picker, allowing several files and offering the
allowed types (see "Attachments API"). Files can also be
pasted into the content editor or dropped onto it, wherever the content editor is shown. Each file
SHALL be uploaded and an attachment block for it inserted, in the order of the files: at the drop
position for dropped files, otherwise at the cursor when the content editor has or last had one,
and at the end of the note when it never had one. While a file is uploading, a placeholder with its
name SHALL show where it will appear. When an upload fails, its placeholder SHALL be removed and a
message SHALL name the file and say that it is too large (with the maximum size), that its type
isn't allowed, or that it couldn't be uploaded. Inserting attachments counts as a change and is
saved automatically.

#### Scenario: Insert at the cursor
- **WHEN** Ann puts the cursor between two paragraphs of "Garden" and adds `roses.jpg` from the dropdown
- **THEN** the photo appears between the two paragraphs, and after the autosave the note's content contains its attachment block there

#### Scenario: Insert at the end
- **WHEN** Ann opens "Garden" and adds two files from the dropdown without placing the cursor
- **THEN** both appear at the end of the note, in the order she chose them

#### Scenario: Paste a screenshot
- **WHEN** Ann pastes a screenshot into the content editor
- **THEN** it is uploaded and shown as an image at the cursor

#### Scenario: Upload too large
- **WHEN** Ann drops a 20 MB video onto the content and the maximum is 5 MB
- **THEN** no block is inserted and a message says the file is larger than 5 MB
