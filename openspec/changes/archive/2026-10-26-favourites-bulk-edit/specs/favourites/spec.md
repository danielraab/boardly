# Spec Delta

## ADDED Requirements

### Requirement: Selection mode
`/favourites` SHALL have a "Select" / "Auswählen" button that turns selection mode on. Only in selection mode, every row SHALL show a checkbox in place of its drag handle, and nodes can't be moved by drag and drop. "Done" / "Fertig" or Escape turns it off and clears the selection. Checking a folder SHALL check every node below it, and unchecking it unchecks them. A folder is checked when all nodes below it are checked, partly checked when some are, and unchecked otherwise (an empty folder is checked on its own). Shift-clicking a checkbox SHALL set every shown row between it and the last clicked one to the clicked state. "Select all" / "Alle auswählen" SHALL check every shown row. While a filter is active, checking a folder SHALL check only its shown nodes. The selection SHALL be kept when the filter changes.

#### Scenario: Checkboxes only in selection mode
- **WHEN** the user opens `/favourites`
- **THEN** no row shows a checkbox, until the user activates "Select"

#### Scenario: Check a folder
- **WHEN** "Work" contains "Tools" with two links, and the link "Mail", and the user checks "Work"
- **THEN** "Work", "Tools", both links and "Mail" are checked

#### Scenario: Uncheck one child
- **WHEN** "Work" is checked and the user unchecks "Mail"
- **THEN** "Work" is shown as partly checked and "Tools" stays checked

#### Scenario: Range
- **WHEN** the user checks the first of five shown links and shift-clicks the fifth
- **THEN** all five are checked

#### Scenario: Filtered folder
- **WHEN** a search shows two of the four links in "Work" and the user checks "Work"
- **THEN** only those two links are checked

#### Scenario: Leave selection mode
- **WHEN** links are checked and the user presses Escape
- **THEN** selection mode ends and nothing is selected

### Requirement: Selection bar
While anything is selected, `/favourites` SHALL show a bar that stays visible while scrolling. It shows how many links and folders are selected, including selected nodes the filter hides, and offers "Add tags", "Remove tags", "Set tags", "Move to…" and "Delete". Tag actions and "Delete" SHALL be disabled when no link is selected. Each action states how many links it applies to.

#### Scenario: Counts
- **WHEN** the user checks "Tools", which holds two links, and one other link
- **THEN** the bar says that 3 links and 1 folder are selected

#### Scenario: Only folders
- **WHEN** only empty folders are selected
- **THEN** "Move to…" is enabled, and the tag actions and "Delete" are disabled

### Requirement: Bulk tag actions
"Add tags" / "Tags hinzufügen" SHALL add the entered tags to every selected link and keep their other tags. "Remove tags" / "Tags entfernen" SHALL list the tags of the selected links with how many of them have each, and remove the chosen ones from every selected link. "Set tags" / "Tags festlegen" SHALL replace the tags of every selected link with the entered ones, and its dialog SHALL say that the links' current tags are replaced. The tag rules and suggestions of single links apply. Tags on no link afterwards are removed. Folders in the selection are ignored. The selection is kept after a tag action.

#### Scenario: Add a tag
- **WHEN** three links are selected, one of them already has "reading", and the user adds "reading" and "later"
- **THEN** all three have "reading" and "later", plus their other tags, and none has "reading" twice

#### Scenario: Remove a tag
- **WHEN** two of three selected links have "old" and the user removes "old"
- **THEN** none of the three has "old", and "old" is removed when no other link has it

#### Scenario: Set tags
- **WHEN** the selected links have various tags and the user sets "archive"
- **THEN** every selected link has only "archive"

### Requirement: Bulk move
"Move to…" / "Verschieben nach…" SHALL open a dialog to choose a folder, including the root. Folders that are selected, or are inside a selected folder, can't be chosen. A fully selected folder SHALL be moved with its contents, and its contents are not moved on their own. Any other selected node is moved on its own. The moved nodes SHALL be added at the end of the chosen folder in their order in the tree. The move is all or nothing.

#### Scenario: Move links
- **WHEN** the user selects "A" from "Inbox" and "B" from "Work" and moves them to "Archive"
- **THEN** "Archive" ends with "A" and then "B", and neither is in its old folder

#### Scenario: Move a folder as a whole
- **WHEN** "Tools" is fully selected and the user moves the selection to "Old"
- **THEN** "Old" contains "Tools" with all its contents

#### Scenario: Partly selected folder
- **WHEN** one of two links in "Tools" is selected and moved to "Old"
- **THEN** only that link is moved and "Tools" stays where it is

#### Scenario: Not into itself
- **WHEN** "Work" is selected and the user opens "Move to…"
- **THEN** "Work" and the folders inside it can't be chosen

### Requirement: Bulk delete with undo
"Delete" / "Löschen" SHALL delete the selected links. Folders SHALL NOT be deleted in bulk, also when all their contents are selected, and they stay in the tree, possibly empty. There is no confirmation dialog. The links disappear right away, and a message "N links deleted" / "N Links gelöscht" with "Undo" / "Rückgängig" is shown for 8 seconds. "Undo" SHALL bring the links back unchanged. The delete is sent to the server when the message closes, when another delete starts, or when the page is left. Tags on no link afterwards are removed. The selection is cleared.

#### Scenario: Delete links
- **WHEN** three links are selected and the user activates "Delete"
- **THEN** the three links disappear, and "3 links deleted" with "Undo" is shown

#### Scenario: Undo
- **WHEN** the user activates "Undo" while the message is shown
- **THEN** the three links are shown again in their places, and nothing is deleted on the server

#### Scenario: Folder stays
- **WHEN** "Tools" and both its links are selected and the user deletes
- **THEN** both links are deleted and "Tools" stays, empty

#### Scenario: Committed
- **WHEN** the message closes without "Undo" and the user reloads the page
- **THEN** the three links are gone

### Requirement: Bulk API
`POST /api/favourites/bulk` SHALL take `ids` (1 to 1000 node IDs, no duplicates), `action` (`delete`, `addTags`, `removeTags`, `setTags` or `move`), `tags` for the tag actions and `parentId` for `move`. It SHALL apply the action to all nodes in one transaction and respond 204, or change nothing and respond with an error. A node ID of another user, or one that doesn't exist, responds 404. A folder ID with `delete` or a tag action, the root, or a missing or invalid field responds 400 with `invalid_field`. With `move`, a node inside another given node, or a target folder inside a given node, responds 400 with `invalid_move`. Invalid tag names respond as on a single link.

#### Scenario: All or nothing
- **WHEN** a request deletes three links and one of the IDs doesn't exist
- **THEN** the server responds 404 and none of the three links is deleted

#### Scenario: Folder delete rejected
- **WHEN** a request with `action: "delete"` contains a folder ID
- **THEN** the server responds 400 with `invalid_field` and nothing is deleted

#### Scenario: Move into own subtree
- **WHEN** a move request contains "Work" and has the target "Work / Tools"
- **THEN** the server responds 400 with `invalid_move`
