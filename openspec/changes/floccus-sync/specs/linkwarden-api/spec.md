# Spec Delta

## ADDED Requirements

### Requirement: Get and edit a collection
`GET /api/v1/collections/{id}` SHALL respond 200 with `{"response": <collection>}` in the shape of "Collections are folders" for a collection in the subtree. An unknown or out-of-scope ID responds 404 with `{"response": "Collection not found."}`.

`PUT /api/v1/collections/{id}` SHALL accept a collection body, as sync clients such as Floccus send the collection they fetched with `name` and `parentId` changed, and respond 200 with `{"response": <the updated collection>}`. A non-empty `name` that differs from the folder's name renames the folder. A `parentId` that differs from the current parent moves the folder to the end of that collection's folder; `null` means the token's root collection. The parent must be in the subtree, else 400 with "Collection is not accessible.", and a move into the collection itself or one of its subcollections responds 400. The other fields of the collection shape (`id`, `description`, `icon`, `iconWeight`, `color`, `isPublic`, `ownerId`, `createdById`, `createdAt`, `updatedAt`, `parent`, `members`, `_count`) are ignored when they hold the values that `GET` returns or are missing; a non-empty `description`, `isPublic: true` or a non-empty `members` list responds 400 with a message that names the unsupported field. `color`, `icon` and `iconWeight` are ignored, as when creating. Renaming or moving the token's root collection responds 403, while a `PUT` that leaves its name and parent unchanged succeeds without changes. The rules of folder nodes apply, and violations respond 400. A rejected request changes nothing.

#### Scenario: Get a collection
- **WHEN** the client calls `GET /api/linkwarden/api/v1/collections/{id}` for the folder "Later"
- **THEN** the server responds 200 with `{"response": {…}}`, where `response.name` is "Later"

#### Scenario: Rename a collection
- **WHEN** Floccus sends `PUT /api/v1/collections/{id}` with the fetched collection "Later" and `name` "Read later"
- **THEN** the folder is called "Read later" and the response has `name` "Read later"

#### Scenario: Move a collection
- **WHEN** the client sends the collection "Videos" with `parentId` the ID of "Reading"
- **THEN** "Videos" is the last node of "Reading"

#### Scenario: Move into a subcollection
- **WHEN** the client moves "Later" into its own subcollection "Videos"
- **THEN** the server responds 400 and nothing changes

#### Scenario: Out-of-scope collection
- **WHEN** the client calls `GET /api/v1/collections/{id}` for a folder outside the token's subtree
- **THEN** the server responds 404 with `{"response": "Collection not found."}`

#### Scenario: Rename the root collection
- **WHEN** the client renames the token's root collection
- **THEN** the server responds 403 and nothing changes

## MODIFIED Requirements

### Requirement: Delete collections
`DELETE /api/v1/collections/{id}` SHALL delete the folder together with its contents (links and subfolders, at any depth), as Linkwarden does, and respond 200 with `{"response": <the deleted collection>}`. Sync clients such as Floccus rely on this: they delete a folder with one request and expect its links to be gone. Tags that are no longer on any link are removed, and app tokens rooted at the deleted folder or one of its subfolders are revoked, as for any folder delete. Deleting the token's root collection responds 403.

#### Scenario: Delete a collection with links
- **WHEN** the app deletes "Later", which contains 3 links and the subfolder "Videos" with 2 links
- **THEN** "Later", "Videos" and all 5 links are gone, and the parent folder is otherwise unchanged

#### Scenario: Delete the root collection
- **WHEN** the app deletes the token's root collection
- **THEN** the server responds 403 and nothing is deleted
