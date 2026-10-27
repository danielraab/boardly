# Spec Delta

## MODIFIED Requirements

### Requirement: Save link page
`/share` SHALL require a signed-in user and show a "Save link" form with the title, the URL, tags and a folder choice, in the order of the link form (see "Link form" in the favourites spec) and with its hint about the description and preview image, prefilled from the query parameters:
- The URL is `url` if it is an absolute http or https URL. Otherwise it is the first http or https URL in `text`. Otherwise the field is empty.
- The title is `title` if present. Otherwise it is `text` without the URL taken from it, trimmed. It is cut to the link title limit.
- The folder is the user's default folder for new links (see "Default folder for new links" in the user-profile spec) if it is set and still exists, otherwise the root folder. Choosing another folder on the page SHALL apply only to that save and SHALL NOT change the default folder or be remembered.
Saving SHALL create a link in the chosen folder, with the same validation as the favourites page, and then show a confirmation with ways to open the favourites and to save another link. "Cancel" SHALL return to `/`. All text SHALL be translated (English and German).

#### Scenario: URL in the url parameter
- **WHEN** a signed-in user opens `/share?title=Docs&url=https%3A%2F%2Fexample.org%2Fdocs`
- **THEN** the form shows the URL `https://example.org/docs` and the title "Docs"

#### Scenario: URL inside the text
- **WHEN** a signed-in user opens `/share?text=Read%20this%20https%3A%2F%2Fexample.org%2Fa`
- **THEN** the form shows the URL `https://example.org/a` and the title "Read this"

#### Scenario: Nothing shared
- **WHEN** a signed-in user opens `/share` without parameters
- **THEN** the form is empty and the folder is the default folder, or the root

#### Scenario: Title before URL and hint
- **WHEN** a signed-in user opens `/share`
- **THEN** the title field comes before the URL field, and the page shows the hint that the description and preview image are loaded in the background

#### Scenario: Save
- **WHEN** the user picks the folder "Work", adds the tag "docs" and saves
- **THEN** "Work" contains the link with the tag "docs", and the page shows the confirmation

#### Scenario: Default folder preselected
- **WHEN** the user's default folder is "Inbox" and they open `/share`
- **THEN** "Inbox" is preselected

#### Scenario: Folder remembered
- **WHEN** the default folder is "Inbox", the user saves a link into "Work" and opens `/share` again
- **THEN** "Inbox" is preselected again: the folder last used on this page is no longer remembered

#### Scenario: No default folder
- **WHEN** the user has no default folder set and opens `/share`
- **THEN** the root folder is preselected

#### Scenario: Invalid URL
- **WHEN** the user tries to save with the URL `javascript:alert(1)`
- **THEN** nothing is saved and the form shows the translated validation message

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/share?url=https%3A%2F%2Fexample.org`
- **THEN** the browser shows `/login`, and after signing in on the login page the visitor is back on `/share?url=https%3A%2F%2Fexample.org`
