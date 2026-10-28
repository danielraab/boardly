# Spec Delta

## MODIFIED Requirements

### Requirement: Save link page
`/share` SHALL require a signed-in user and show a switch between "Link" / "Link" and "Note" / "Notiz" above the form. The switch starts on "Note" when a title or text was shared but no URL is found in the query parameters (see below), and on "Link" otherwise, so the "Add link" shortcut, which opens `/share` without parameters, starts on "Link". Switching keeps the values entered in both forms.

With "Link", the page SHALL show a "Save link" form with the title, the URL, tags and a folder choice, in the order of the link form (see "Link form" in the favourites spec) and with its hint about the description and preview image, prefilled from the query parameters:
- The URL is `url` if it is an absolute http or https URL. Otherwise it is the first http or https URL in `text`. Otherwise the field is empty.
- The title is `title` if present. Otherwise it is `text` without the URL taken from it, trimmed. It is cut to the link title limit.
- The folder is the user's default folder for new links (see "Default folder for new links" in the user-profile spec) if it is set and still exists, otherwise the root folder. Choosing another folder on the page SHALL apply only to that save and SHALL NOT change the default folder or be remembered.
Saving SHALL create a link in the chosen folder, with the same validation as the favourites page, and then show a confirmation with ways to open the favourites and to save another link.

With "Note", the page SHALL show the form described in "Save note from a share".

"Cancel" SHALL return to `/`. All text SHALL be translated (English and German).

#### Scenario: URL in the url parameter
- **WHEN** a signed-in user opens `/share?title=Docs&url=https%3A%2F%2Fexample.org%2Fdocs`
- **THEN** the switch is on "Link", and the form shows the URL `https://example.org/docs` and the title "Docs"

#### Scenario: URL inside the text
- **WHEN** a signed-in user opens `/share?text=Read%20this%20https%3A%2F%2Fexample.org%2Fa`
- **THEN** the switch is on "Link", and the form shows the URL `https://example.org/a` and the title "Read this"

#### Scenario: Nothing shared
- **WHEN** a signed-in user opens `/share` without parameters
- **THEN** the switch is on "Link", the form is empty and the folder is the default folder, or the root

#### Scenario: Title before URL and hint
- **WHEN** a signed-in user opens `/share`
- **THEN** the title field comes before the URL field, and the page shows the hint that the description and preview image are loaded in the background

#### Scenario: Save
- **WHEN** the user picks the folder "Work", adds the tag "docs" and saves a link
- **THEN** "Work" contains the link with the tag "docs", and the page shows the confirmation

#### Scenario: Default folder preselected
- **WHEN** the user's default folder is "Inbox" and they open `/share` with a URL
- **THEN** "Inbox" is preselected

#### Scenario: Folder remembered
- **WHEN** the default folder is "Inbox", the user saves a link into "Work" and opens `/share` again
- **THEN** "Inbox" is preselected again: the folder last used on this page is no longer remembered

#### Scenario: No default folder
- **WHEN** the user has no default folder set and opens `/share` with a URL
- **THEN** the root folder is preselected

#### Scenario: Invalid URL
- **WHEN** the user tries to save with the URL `javascript:alert(1)`
- **THEN** nothing is saved and the form shows the translated validation message

#### Scenario: Switch keeps values
- **WHEN** the user opens `/share` with a URL, edits the link title, switches to "Note" and back to "Link"
- **THEN** the edited link title is still there

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/share?url=https%3A%2F%2Fexample.org`
- **THEN** the browser shows `/login`, and after signing in on the login page the visitor is back on `/share?url=https%3A%2F%2Fexample.org`

## ADDED Requirements

### Requirement: Save note from a share
With "Note" on the share page, the page SHALL show a "Save note" / "Notiz speichern" form with a title, a multi-line text field and a category choice, prefilled from the query parameters:
- The title is `title` if present, cut to the note title limit. Otherwise it is empty.
- The text is `text`. When `url` is an absolute http or https URL that isn't already contained in `text`, it is added as a last line, separated by a line break when `text` isn't empty.
- The category is the user's default category for new notes (see "Default category for new notes" in the user-profile spec) if it is set and still exists, otherwise "No category". Choosing another category SHALL apply only to that save.
Saving SHALL create a note from the title, the text as plain text (see "Content from plain text" in the note-document spec) and the chosen category, and then show a confirmation with ways to open the note and to save something else. A note can be saved with an empty text. A failed save shows a translated error and keeps the form.

#### Scenario: Shared text without a URL
- **WHEN** a signed-in user opens `/share?title=Plan&text=Meet%20at%208`
- **THEN** the switch is on "Note", the title is "Plan" and the text is "Meet at 8"

#### Scenario: URL added to the text
- **WHEN** the user opens `/share?text=Look%20at%20this&url=https%3A%2F%2Fexample.org%2Fa` and switches to "Note"
- **THEN** the text is "Look at this", a line break, and `https://example.org/a`

#### Scenario: Save a note
- **WHEN** the user saves the note "Plan" with the text "Meet at 8" in the category "Inbox"
- **THEN** "Inbox" contains the note "Plan" with a paragraph "Meet at 8", and the page shows the confirmation with a way to open the note

#### Scenario: Default category preselected
- **WHEN** the user's default category is "Inbox" and they open `/share?text=Meet%20at%208`
- **THEN** "Inbox" is preselected in the note form

#### Scenario: No default category
- **WHEN** the user has no default category, opens `/share` and switches to "Note"
- **THEN** "No category" is preselected
