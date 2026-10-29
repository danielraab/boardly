## ADDED Requirements

### Requirement: Delete from the link edit dialog
The dialog to edit an existing link SHALL show a "Delete" / "Löschen" button, apart from "Cancel" and "Save". Activating it SHALL open the same confirmation as deleting the link from the tree, on top of the edit dialog. Confirming SHALL delete the link and close both dialogs, without saving any unsaved changes in the edit dialog. Cancelling the confirmation SHALL return to the edit dialog with its unsaved changes kept. When the delete fails, the confirmation SHALL stay open and show a translated message.

The dialogs for a new link, a new folder and editing a folder SHALL NOT show this button.

#### Scenario: Delete a link from its edit dialog
- **WHEN** the user opens the edit dialog of "Docs" on `/favourites`, activates "Delete" and confirms
- **THEN** both dialogs close and "Docs" is removed from the tree

#### Scenario: Cancel the confirmation
- **WHEN** the user opens the edit dialog of "Docs", changes the title to "Team docs", activates "Delete" and cancels the confirmation
- **THEN** the edit dialog is still open and shows "Team docs"

#### Scenario: Delete fails
- **WHEN** the user confirms the delete of "Docs" and the request fails
- **THEN** the confirmation stays open, shows a translated error, and "Docs" is still in the tree

#### Scenario: No delete for folders and new nodes
- **WHEN** the user opens the edit dialog of a folder, or the dialog for a new link or folder
- **THEN** it shows no "Delete" button
