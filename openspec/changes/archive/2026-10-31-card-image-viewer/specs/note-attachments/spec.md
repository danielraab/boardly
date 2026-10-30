# Spec Delta

## MODIFIED Requirements

### Requirement: Attachments in shown notes
Wherever a note's content is shown (the editor page and the editor modal, inline editing on cards,
note cards and note widgets), an attachment block SHALL be shown as its own block:
- an image as the image, at most as wide as the content and at most as wide as the image itself,
  with the attachment's name as its alternative text;
- audio and video with the browser's player;
- any other file as a file card with an icon for its kind, its name, its size and a download link.

Activating an image SHALL open the image viewer, also on note cards. On note cards, activating any
other content, including file cards, audio and video, starts inline editing as before (see "Inline
editing on cards" in the `notes` capability). The
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

#### Scenario: Image on a card
- **WHEN** Ann clicks the photo `roses.jpg` on the card "Garden" on `/notes`
- **THEN** the image viewer shows `roses.jpg`, the card doesn't start inline editing, and closing the viewer returns focus to the photo

#### Scenario: File on a card
- **WHEN** Ann clicks the file card `report.pdf` on the card "Taxes" on `/notes`
- **THEN** the card starts inline editing and nothing is downloaded
