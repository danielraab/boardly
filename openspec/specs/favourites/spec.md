# favourites Specification

## Purpose

Defines each user's favourites tree: the root folder, folders and links, their order, moving them with drag and drop, deleting folders with or without their contents, tags on links, and the `/favourites` page that manages it all.

## Requirements

### Requirement: One favourites tree per user
Every user SHALL have exactly one favourites tree. The tree SHALL have a root folder, which can't be renamed, moved or deleted. Every other node SHALL have exactly one parent folder, so the tree can't contain cycles. A user SHALL only be able to see and change their own tree: nodes and tags of other users SHALL behave as if they didn't exist.

#### Scenario: New user has an empty tree
- **WHEN** a user who never saved a favourite opens `/favourites`
- **THEN** the page shows an empty tree with a hint on how to add the first folder or link

#### Scenario: Root can't be deleted
- **WHEN** a client sends a request to delete or move the root folder
- **THEN** the server rejects it with a 400 error and the tree is unchanged

#### Scenario: Other users' nodes are invisible
- **WHEN** a user requests, changes or deletes a node that belongs to another user
- **THEN** the server responds with 404, as for a node that doesn't exist

### Requirement: Folder nodes
A folder SHALL have a name of 1 to 100 characters after trimming, and SHALL contain any number of folders and links, nested to any depth.

#### Scenario: Create a nested folder
- **WHEN** the user adds a folder named "Work" to the root and then a folder named "Tools" inside "Work"
- **THEN** the tree shows "Tools" inside "Work"

#### Scenario: Rename a folder
- **WHEN** the user renames the folder "Work" to "Office"
- **THEN** the tree shows "Office" with the same contents

#### Scenario: Empty folder name
- **WHEN** the user tries to save a folder whose name is empty or only whitespace
- **THEN** the folder isn't saved and the form shows a translated validation message

### Requirement: Link nodes
A link SHALL have an absolute `http` or `https` URL of at most 2048 characters and an optional title of at most 1000 characters after trimming. An empty title SHALL be stored as no title. The tree SHALL show a link by its title, or by its URL when it has no title. Activating a link in the tree SHALL open its URL in a new tab. A link SHALL show the favicon of its URL's origin when one is known, and a globe icon otherwise. Creating a link or changing its URL SHALL start fetching that origin's favicon in the background, without delaying the response. In the API, a link node has `iconImageId` set to its favicon's image ID when one is known, and leaves it out otherwise.

#### Scenario: Create a link with a title
- **WHEN** the user adds a link with URL `https://example.org/docs` and title "Docs" to the folder "Work"
- **THEN** "Work" shows a link labelled "Docs"

#### Scenario: Link without title
- **WHEN** the user adds a link with URL `https://example.org` and no title
- **THEN** the tree shows the link labelled `https://example.org`

#### Scenario: Invalid URL
- **WHEN** the user tries to save a link with the URL `example` or `javascript:alert(1)`
- **THEN** the link isn't saved and the form shows a translated validation message

#### Scenario: Open a link
- **WHEN** the user activates the link "Docs" in the tree
- **THEN** `https://example.org/docs` opens in a new tab

#### Scenario: Edit a link
- **WHEN** the user changes the title of "Docs" to "Documentation" and its URL to `https://example.org/doc`
- **THEN** the tree shows "Documentation", which opens the new URL

#### Scenario: Favicon appears
- **WHEN** the user adds a link to `https://github.com/` and the favicon fetch succeeds
- **THEN** the link shows GitHub's favicon once the tree is loaded again

#### Scenario: No favicon
- **WHEN** a link's origin has no favicon
- **THEN** the link shows a globe icon

### Requirement: Sibling order
The children of a folder SHALL keep the order the user gave them. A new node SHALL be added at the end of its folder, except a link saved through a connected app, which SHALL be added first in the token's root folder.

#### Scenario: New node goes last
- **WHEN** the user adds a link to a folder that already contains "A" and "B"
- **THEN** the folder shows "A", "B" and then the new link

#### Scenario: Order survives a reload
- **WHEN** the user reorders the children of a folder and reloads the page
- **THEN** the folder shows the children in the new order

#### Scenario: Link saved from a connected app goes first
- **WHEN** the browser extension saves a page with a token rooted at "Reading", which already contains "A" and "B"
- **THEN** "Reading" shows the new link, then "A" and "B"

### Requirement: Move nodes with drag and drop
On `/favourites`, the user SHALL be able to drag any node except the root and drop it before or after another node, which puts it into that node's folder at that position, or onto a folder, which puts it at the end of that folder. A folder SHALL NOT be droppable into itself or into one of its descendants. Moving a folder SHALL move everything inside it. Moving SHALL also be possible with the keyboard.

#### Scenario: Reorder within a folder
- **WHEN** a folder contains "A", "B" and "C" and the user drags "C" and drops it before "A"
- **THEN** the folder shows "C", "A" and "B"

#### Scenario: Move into another folder
- **WHEN** the user drags the link "Docs" onto the folder "Archive"
- **THEN** "Docs" is the last child of "Archive" and is no longer in its old folder

#### Scenario: Move a folder with its contents
- **WHEN** the user drags the folder "Work", which contains "Tools", into the folder "Old"
- **THEN** "Old" contains "Work", which still contains "Tools"

#### Scenario: Folder can't go into its own subtree
- **WHEN** the user drags the folder "Work" over its child folder "Tools"
- **THEN** no drop is offered there, and a move request to that target is rejected by the server with a 400 error

#### Scenario: Move with the keyboard
- **WHEN** the user focuses the drag handle of a node, starts a drag with Space, moves it with the arrow keys and drops it with Space
- **THEN** the node is moved as if it had been dragged with the mouse

### Requirement: Delete nodes
Deleting a link or an empty folder SHALL ask for confirmation. Deleting a folder that isn't empty SHALL ask whether its contents should be deleted with it or moved to the folder's parent. Contents that are moved SHALL take the deleted folder's place in the parent and keep their order. Tags that are no longer on any link after a delete SHALL be removed.

#### Scenario: Delete a link
- **WHEN** the user deletes the link "Docs" and confirms
- **THEN** "Docs" is removed from the tree

#### Scenario: Delete a folder with its contents
- **WHEN** the user deletes the folder "Work", which contains "Tools" and "Docs", and chooses to delete the contents
- **THEN** "Work", "Tools", "Docs" and everything below them are removed

#### Scenario: Delete a folder and keep its contents
- **WHEN** the root contains "A", "Work" and "B", "Work" contains "Tools" and "Docs", and the user deletes "Work" and chooses to keep the contents
- **THEN** the root shows "A", "Tools", "Docs" and "B"

#### Scenario: Cancel a delete
- **WHEN** the user opens the delete dialog and cancels it
- **THEN** the tree is unchanged

### Requirement: Tags on links
A link SHALL have any number of tags, and folders SHALL NOT have tags. Tags SHALL belong to the user and be shared by all their links. A tag name SHALL be 1 to 100 characters after trimming. Tag names SHALL be unique per user regardless of case. Typing a tag name that already exists in another case SHALL attach the existing tag. A tag name that doesn't exist yet SHALL create the tag when the link is saved. A tag that is no longer on any link SHALL be removed.

#### Scenario: Create a tag on the fly
- **WHEN** the user adds the tag "reading" to a link and saves it, and the user has no tag "reading" yet
- **THEN** the tag "reading" is created and the link shows it

#### Scenario: Same tag in another case
- **WHEN** the user has a tag "Reading" and types "reading" on another link and saves it
- **THEN** that link gets the existing tag "Reading" and no second tag is created

#### Scenario: Remove a tag
- **WHEN** the user removes the tag "reading" from a link and saves it
- **THEN** the link no longer shows "reading", and other links that have "reading" keep it

#### Scenario: Unused tag disappears
- **WHEN** the user removes the tag "old" from the only link that had it
- **THEN** "old" is no longer suggested

### Requirement: Tag suggestions
While the user types a tag name on a link, the tag input SHALL suggest up to 10 of the user's existing tags whose name starts with the typed text, ignoring case and sorted by name. It SHALL leave out tags the link already has. Choosing a suggestion SHALL add that tag to the link.

#### Scenario: Suggest by prefix
- **WHEN** the user has the tags "reading", "recipes" and "tools" and types "re" in the tag input of a link without tags
- **THEN** the input suggests "reading" and "recipes"

#### Scenario: Already attached tags aren't suggested
- **WHEN** a link already has the tag "reading" and the user types "re"
- **THEN** only "recipes" is suggested

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

### Requirement: Delete folders used by connected apps
A delete SHALL revoke every active app token whose root folder is removed by it: the deleted folder itself, and with "delete contents" also every folder below it. With "move contents to the parent", only tokens whose root is the deleted folder itself are revoked; tokens rooted in moved folders keep working. Before such a delete, the confirmation dialog SHALL say how many connected apps use the removed folders and that they will be revoked. Revoked tokens stay in the user's token list, and their root is shown as "Deleted folder" / "Gelöschter Ordner". Expired and already revoked tokens are not counted in the warning.

#### Scenario: Warning in the dialog
- **WHEN** the folder "Reading" is the root of two active tokens and the user opens its delete dialog
- **THEN** the dialog says that 2 connected apps use this folder and will be revoked

#### Scenario: Delete revokes
- **WHEN** the user deletes "Reading" and confirms
- **THEN** both tokens are revoked, requests with them are rejected, and the "Connected apps" tab lists them as revoked with the root "Deleted folder"

#### Scenario: Token rooted in a subfolder, contents deleted
- **WHEN** a token's root is "Work / Tools" and the user deletes "Work" with its contents
- **THEN** the warning counts that token and the token is revoked

#### Scenario: Token rooted in a subfolder, contents moved
- **WHEN** a token's root is "Work / Tools" and the user deletes "Work" and keeps its contents
- **THEN** the token stays active and its root is now "Tools" in the parent of "Work"

#### Scenario: No tokens affected
- **WHEN** no active token uses the folder being deleted or any folder that the delete removes
- **THEN** the dialog shows no connected-apps warning

### Requirement: Integer IDs for apps
Every folder, link and tag SHALL have an integer ID next to its ID, for apps whose clients expect numbers (see the Linkwarden API). The integer ID is a positive number, unique among all nodes or among all tags, assigned when the node or tag is created and never changed or reused, also after the node or tag is deleted. Existing nodes and tags get one when the server is upgraded. Moving, renaming or editing a node keeps its integer ID. The web app's own API doesn't show integer IDs.

#### Scenario: Stable after a move
- **WHEN** a link with integer ID 7 is moved to another folder and renamed
- **THEN** it still has integer ID 7

#### Scenario: Not reused
- **WHEN** the link with the highest integer ID is deleted and a new link is created
- **THEN** the new link gets a higher integer ID than the deleted one

#### Scenario: Existing data
- **WHEN** the server is upgraded and a user already has folders, links and tags
- **THEN** each of them has a distinct integer ID

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

### Requirement: Search favourites
`/favourites` SHALL have a search field ("Search favourites" / "Favoriten durchsuchen"). The text is split at whitespace into terms, and matching ignores case. A link matches when every term occurs in its title, its URL or one of its tag names, or in the name of one of its folders below the root. While the field has text, the tree SHALL show only matching links and the folders that contain them, and a folder whose name contains every term together with all its contents (subject to the tag filter). Every shown folder is expanded. Matching text in the shown names SHALL be highlighted. Pressing `/` outside a text field SHALL focus the search field.

#### Scenario: Find by title
- **WHEN** "Work / Tools" contains the link "Grafana" and the user types "graf"
- **THEN** the tree shows "Work", "Tools" and "Grafana", expanded, and no other node

#### Scenario: Find by URL
- **WHEN** a link titled "Dashboards" has the URL `https://grafana.example.com` and the user types "grafana"
- **THEN** the tree shows "Dashboards"

#### Scenario: Find by tag
- **WHEN** a link has the tag "monitoring" and the user types "monitor"
- **THEN** the tree shows that link

#### Scenario: Several terms
- **WHEN** the user types "work graf"
- **THEN** "Grafana" in "Work / Tools" is shown, and a link "Grafana" outside "Work" is not

#### Scenario: Folder name matches
- **WHEN** the user types "tools"
- **THEN** "Tools" is shown with all its contents

#### Scenario: Keyboard shortcut
- **WHEN** focus is not in a text field and the user presses `/`
- **THEN** the search field is focused

### Requirement: Filter favourites by tag
`/favourites` SHALL offer a tag filter that lists every tag on the user's links, sorted by name, with the number of links that have it. The user can choose several tags; the tree then shows only links that have all chosen tags, together with the folders that contain them, expanded. The chosen tags are shown as chips that can each be removed. Activating a tag on a link row SHALL add it to the filter. The tag filter and the search apply together.

#### Scenario: One tag
- **WHEN** the user chooses the tag "reading"
- **THEN** the tree shows only links with "reading" and their folders

#### Scenario: Several tags
- **WHEN** the user chooses "reading" and "later"
- **THEN** only links with both tags are shown

#### Scenario: Tag on a row
- **WHEN** no tag is chosen and the user activates the tag "reading" on a link row
- **THEN** "reading" is chosen in the tag filter

#### Scenario: Search and tag
- **WHEN** the tag "reading" is chosen and the user types "rust"
- **THEN** only links with "reading" that match "rust" are shown

### Requirement: Filter state and clearing
The search text and the chosen tags SHALL be kept in the page URL, so a reload or going back restores them. While a filter is active, a "Clear filters" / "Filter zurücksetzen" action SHALL clear both. When nothing matches, the page SHALL show "No favourites match" / "Keine passenden Favoriten" with the same action instead of the tree.

#### Scenario: Reload
- **WHEN** the user has typed "graf" and chosen "tools", and reloads the page
- **THEN** the search field shows "graf", "tools" is chosen, and the tree is filtered

#### Scenario: Nothing matches
- **WHEN** the user types a text no link matches
- **THEN** the page shows "No favourites match" and "Clear filters"

#### Scenario: Clear
- **WHEN** the user activates "Clear filters"
- **THEN** the search field is empty, no tag is chosen, and the whole tree is shown with the remembered collapse state

### Requirement: No moving while filtered
While a search or tag filter is active, nodes SHALL NOT be movable on `/favourites`, by pointer or keyboard, and the page SHALL show "Clear the filters to reorder." / "Setze die Filter zurück, um die Reihenfolge zu ändern." Editing, deleting and adding nodes stay possible.

#### Scenario: Drag handles off
- **WHEN** a filter is active
- **THEN** no row offers a move action, and the hint is shown

#### Scenario: Edit while filtered
- **WHEN** a filter is active and the user edits a shown link
- **THEN** the change is saved and the filter stays active

### Requirement: Expand and collapse all on the favourites page
`/favourites` SHALL have a button that expands all folders, or collapses all of them when any folder is expanded. Its label says which ("Expand all" / "Alle aufklappen", "Collapse all" / "Alle zuklappen").

#### Scenario: Collapse all
- **WHEN** some folders are expanded and the user activates "Collapse all"
- **THEN** every folder at every level is collapsed, and the button says "Expand all"

#### Scenario: Expand all
- **WHEN** all folders are collapsed and the user activates "Expand all"
- **THEN** every folder at every level is expanded

### Requirement: Remembered collapse state on the favourites page
On a first visit, every folder on `/favourites` SHALL be expanded. The folders the user expands or collapses, also with "Expand all" or "Collapse all", SHALL be remembered in their browser and restored when the page is opened again. Nothing about it is stored on the server. Expanding or collapsing while a filter is active SHALL NOT change the remembered state.

#### Scenario: Remembered
- **WHEN** the user collapses "Work" and reloads the page
- **THEN** "Work" is still collapsed

#### Scenario: Filter doesn't change it
- **WHEN** "Work" is collapsed, the user searches for a link inside it and then clears the filters
- **THEN** "Work" is collapsed again

### Requirement: Folder link counts
Every folder row on `/favourites` SHALL show the number of links it contains at all levels below it. While a filter is active, it SHALL show the number of shown links below it instead.

#### Scenario: Count
- **WHEN** "Work" contains two links and the folder "Tools" with three links
- **THEN** the row of "Work" shows 5 and the row of "Tools" shows 3

#### Scenario: Count while filtered
- **WHEN** a filter is active and one link inside "Work" matches
- **THEN** the row of "Work" shows 1

### Requirement: Link to a folder on the favourites page
`/favourites?folder=<id>` SHALL expand that folder and all folders above it, scroll its row into view and highlight it briefly. The expanded folders are remembered like other expansions. An unknown ID, or the ID of a link, is ignored.

#### Scenario: Open a nested folder
- **WHEN** "Work" and "Tools" are collapsed and the user opens `/favourites?folder=<id of Tools>`
- **THEN** "Work" and "Tools" are expanded, and the row of "Tools" is scrolled into view and highlighted

#### Scenario: Unknown folder
- **WHEN** the user opens `/favourites?folder=unknown`
- **THEN** the page shows the tree as usual

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
