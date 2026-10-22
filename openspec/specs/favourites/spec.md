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
The children of a folder SHALL keep the order the user gave them. A new node SHALL be added at the end of its folder.

#### Scenario: New node goes last
- **WHEN** the user adds a link to a folder that already contains "A" and "B"
- **THEN** the folder shows "A", "B" and then the new link

#### Scenario: Order survives a reload
- **WHEN** the user reorders the children of a folder and reloads the page
- **THEN** the folder shows the children in the new order

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
`/favourites` SHALL require a signed-in user. It SHALL show the user's tree with folders that can be collapsed and expanded, and actions to add a folder or link to a folder and to edit and delete each node. All of its text SHALL be translated (English and German).

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/favourites`
- **THEN** the browser shows `/login`

#### Scenario: Collapse a folder
- **WHEN** the user collapses the folder "Work"
- **THEN** the children of "Work" are hidden, and expanding it shows them again
