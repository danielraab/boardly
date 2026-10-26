# Spec Delta

## ADDED Requirements

### Requirement: Manage tags dialog
`/favourites` SHALL have a "Manage tags" / "Tags verwalten" button that opens a dialog listing all of the user's tags, sorted by name, each with the number of links that have it, a rename action and a delete action. The list can be searched by name. Activating a tag's count SHALL close the dialog and set the tag filter to that tag alone. With no tags, the dialog says "No tags yet" / "Noch keine Tags".

#### Scenario: List
- **WHEN** the user has the tag "reading" on three links and "tools" on one, and opens "Manage tags"
- **THEN** the dialog lists "reading" with 3 and "tools" with 1

#### Scenario: Filter from the dialog
- **WHEN** the user activates the count of "reading"
- **THEN** the dialog closes and the tree shows only links with "reading"

### Requirement: Rename and merge tags
Renaming a tag SHALL change its name on all its links. The tag-name rules apply. Renaming to a name that differs only in case from the old name SHALL just change the case. When another tag already has the new name, ignoring case, the dialog SHALL ask whether to merge "<old>" into "<other>"; merging gives every link of the renamed tag the other tag, keeps the other tag's name, and removes the renamed tag. Cancelling changes nothing.

#### Scenario: Rename
- **WHEN** the user renames "reding" to "reading" and no tag "reading" exists
- **THEN** every link that had "reding" now shows "reading"

#### Scenario: Change case
- **WHEN** the user renames "reading" to "Reading"
- **THEN** the tag is called "Reading" and no merge is offered

#### Scenario: Merge
- **WHEN** "Reading" is on links A and B, "reading-list" is on B and C, and the user renames "reading-list" to "reading" and confirms the merge
- **THEN** A, B and C have "Reading", B has it once, and "reading-list" no longer exists

#### Scenario: Cancel merge
- **WHEN** the merge question is shown and the user cancels
- **THEN** both tags are unchanged

### Requirement: Delete tags
Deleting a tag SHALL ask for confirmation, saying on how many links it is ("Remove "<name>" from N links?" / "„<name>“ von N Links entfernen?"). Confirming removes the tag from all its links and deletes it. The links themselves are kept.

#### Scenario: Delete
- **WHEN** the user deletes "old", which is on two links, and confirms
- **THEN** both links are kept without "old", and "old" is no longer listed or suggested

### Requirement: Tag management API
`GET /api/tags?all=true` SHALL return all of the user's tags as `{id, name, linkCount}`, sorted by name ignoring case. Without `all`, `GET /api/tags` keeps suggesting by prefix as before. `PATCH /api/tags/{id}` with `{name}` SHALL rename the tag and respond with it; when another tag of the user has that name, ignoring case, it SHALL respond 409 with `tag_exists` and change nothing, unless the body has `"merge": true`, in which case it merges the tag into that other tag and responds with the other tag. `DELETE /api/tags/{id}` SHALL remove the tag from all links and delete it, responding 204. A tag of another user, or one that doesn't exist, responds 404. An invalid name responds 400 as for tags on a link.

#### Scenario: Name taken
- **WHEN** the user has "Reading" and "reading-list" and sends `PATCH /api/tags/<reading-list>` with `{"name": "reading"}`
- **THEN** the server responds 409 with `tag_exists` and both tags are unchanged

#### Scenario: Merge through the API
- **WHEN** the same request has `"merge": true`
- **THEN** the server responds with the tag "Reading", which is now on all links that had either tag

#### Scenario: Other user's tag
- **WHEN** a user sends `DELETE /api/tags/<id>` for another user's tag
- **THEN** the server responds 404 and the tag is unchanged
