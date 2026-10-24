# Spec Delta

## MODIFIED Requirements

### Requirement: Favourites page
`/favourites` SHALL require a signed-in user. It SHALL show the user's tree with folders that can be collapsed and expanded, and actions to add a folder or link to a folder and to edit and delete each node. The icon of a link is refreshed from its edit dialog (see "Link edit dialog"). All of its text SHALL be translated (English and German).

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/favourites`
- **THEN** the browser shows `/login`

#### Scenario: Collapse a folder
- **WHEN** the user collapses the folder "Work"
- **THEN** the children of "Work" are hidden, and expanding it shows them again

#### Scenario: No refresh action in the tree
- **WHEN** the user looks at a link row in the tree
- **THEN** the row offers move, edit and delete, but no "Refresh icon" action

#### Scenario: Refresh an icon
- **WHEN** the user chooses "Refresh icon" in the edit dialog of the link "GitHub" and the site's favicon is found
- **THEN** the dialog and the tree show the refreshed favicon, and the dialog stays open

#### Scenario: Refresh finds no icon
- **WHEN** the user chooses "Refresh icon" in the edit dialog of a link whose site has no favicon
- **THEN** the dialog shows that no icon was found, and the link keeps its icon

## ADDED Requirements

### Requirement: Link form
The form for a link SHALL show its fields in this order: title, URL, tags and, when a folder can be chosen, the folder. When the form creates a new link, it SHALL show the hint "The description and preview image are loaded from the page in the background. You can edit them afterwards." / "Beschreibung und Vorschaubild werden im Hintergrund von der Seite geladen. Du kannst sie danach bearbeiten." All of its text SHALL be translated (English and German).

#### Scenario: Title before URL
- **WHEN** the user opens the dialog to add a link
- **THEN** the title field comes before the URL field

#### Scenario: Hint when adding
- **WHEN** the user opens the dialog to add a link
- **THEN** the dialog shows the hint about the description and preview image, and has no description or preview fields

### Requirement: Link edit dialog
The dialog to edit an existing link SHALL additionally show, after the tags:

- a "Description" / "Beschreibung" multi-line field with the link's description, at most 2048 characters. Saving sends it with the other fields; an empty field clears the description. A description that is too long isn't saved, and the form shows a translated validation message.
- the link's preview image, or "No preview image" / "Kein Vorschaubild" when it has none. With a preview, a "Remove preview" / "Vorschaubild entfernen" action marks it for removal and the dialog stops showing it; saving the dialog removes it, and cancelling keeps it.
- the link's current icon with a "Refresh icon" / "Icon neu laden" action that refreshes the favicon of the link's saved URL right away (see "Manual favicon refresh" in the images spec), without saving the dialog, and then shows the new icon in the dialog and the tree. While the refresh runs, the action is disabled. When no favicon is found, the address is blocked or too many refreshes were made, the dialog shows a translated message and the link keeps its icon.

The dialog to edit a folder SHALL NOT show these fields.

#### Scenario: Edit a description
- **WHEN** the user opens the edit dialog of "Docs", enters the description "Team docs" and saves
- **THEN** "Docs" has the description "Team docs", and opening the dialog again shows it

#### Scenario: Description too long
- **WHEN** the user enters a description of 3000 characters and saves
- **THEN** the link isn't saved and the dialog shows a translated validation message

#### Scenario: Remove the preview
- **WHEN** the user opens the edit dialog of a link with a preview, chooses "Remove preview" and saves
- **THEN** the link has no preview, and opening the dialog again shows "No preview image"

#### Scenario: Cancel keeps the preview
- **WHEN** the user chooses "Remove preview" and then cancels the dialog
- **THEN** the link keeps its preview

#### Scenario: Folder dialog
- **WHEN** the user opens the edit dialog of a folder
- **THEN** it shows only the name field
