# Spec Delta

## ADDED Requirements

### Requirement: Karakeep REST transport
Besides the tRPC procedures, Boardly SHALL answer the Karakeep REST endpoints named in this spec under `/api/karakeep/api/v1/`, which sync clients such as Floccus call as `<server address>/api/v1/…`. They require an active Karakeep token (see "Bearer authentication for app endpoints" in the app-tokens spec) and see only the token's subtree and note scope, like the procedures. Request and response bodies are plain JSON (no superjson envelope); times are ISO 8601 strings in UTC with milliseconds. Bookmarks and lists SHALL have the same shape as in the procedures. An error SHALL respond with `{"error": "<message>"}` and the status 400 for invalid input, 401 for a missing or invalid token, 404 for an unknown or out-of-scope bookmark or list, and 500 for anything else, with the same messages as the matching procedure. Karakeep REST paths and methods that this spec doesn't name stay unimplemented and respond 404.

#### Scenario: Without a token
- **WHEN** a client calls `GET /api/karakeep/api/v1/lists` without an `Authorization` header
- **THEN** the server responds 401 with `{"error": "Unauthorized"}`

#### Scenario: Unnamed REST path
- **WHEN** a client calls `GET /api/karakeep/api/v1/highlights` with an active Karakeep token
- **THEN** the server responds 404

#### Scenario: Out-of-scope bookmark
- **WHEN** a client calls `GET /api/karakeep/api/v1/bookmarks/{id}` for a link outside the token's subtree
- **THEN** the server responds 404

### Requirement: Karakeep REST bookmarks
The bookmark endpoints SHALL behave like the procedure named for each:

- `POST /api/v1/bookmarks` (`bookmarks.createBookmark`): responds 201 with the new bookmark and `alreadyExists: false`, or 200 with the existing bookmark and `alreadyExists: true` when a bookmark with the same URL exists in the subtree.
- `GET /api/v1/bookmarks/{id}` (`bookmarks.getBookmark`): responds 200 with the bookmark. `includeContent` is accepted and ignored.
- `PATCH /api/v1/bookmarks/{id}` (`bookmarks.updateBookmark`, with `bookmarkId` from the path): responds 200 with the updated bookmark.
- `DELETE /api/v1/bookmarks/{id}` (`bookmarks.deleteBookmark`): responds 204.
- `GET /api/v1/bookmarks/{id}/lists` (`lists.getListsOfBookmark`): responds 200 with `{"lists"}`.
- `POST /api/v1/bookmarks/{id}/tags` with `{"tags": [{"tagId"} | {"tagName"}]}` (`bookmarks.updateTags` with these tags to attach): responds 200 with `{"attached": [<tag IDs>]}`.
- `DELETE /api/v1/bookmarks/{id}/tags` with the same body (`bookmarks.updateTags` with these tags to detach): responds 200 with `{"detached": [<tag IDs>]}`.

#### Scenario: Create a bookmark
- **WHEN** Floccus sends `POST /api/karakeep/api/v1/bookmarks` with `{"type": "link", "url": "https://example.org", "title": "Example"}`
- **THEN** the server responds 201, the link "Example" is the first node of the token's root folder, and the response has `alreadyExists: false`

#### Scenario: Create an existing URL
- **WHEN** the subtree already has a link to `https://example.org` and Floccus creates a bookmark for the same URL
- **THEN** the server responds 200 with that link's bookmark and `alreadyExists: true`, and no link is created

#### Scenario: Rename a bookmark
- **WHEN** Floccus sends `PATCH /api/v1/bookmarks/{id}` with `{"url": "https://example.org", "title": "Example site"}`
- **THEN** the link's title is "Example site" and the server responds 200 with the bookmark

#### Scenario: Attach and detach tags
- **WHEN** Floccus attaches `{"tagName": "dev"}` to a link and then detaches it
- **THEN** the first call responds with the tag's ID in `attached`, the second with it in `detached`, and the link has no tag "dev"

#### Scenario: Delete a bookmark
- **WHEN** Floccus sends `DELETE /api/v1/bookmarks/{id}` for a link
- **THEN** the server responds 204 and the link is gone

### Requirement: Karakeep REST lists
The list endpoints SHALL behave like the procedure named for each:

- `GET /api/v1/lists` (`lists.list`): responds 200 with `{"lists"}`.
- `POST /api/v1/lists` (`lists.create`): responds 201 with the new list. `icon` and `type: "manual"` are accepted.
- `PATCH /api/v1/lists/{id}` (`lists.edit`, with `listId` from the path): responds 200 with the list.
- `DELETE /api/v1/lists/{id}` (`lists.delete`): responds 204.
- `PUT /api/v1/lists/{id}/bookmarks/{bookmarkId}` (`lists.addToList`): responds 204.
- `DELETE /api/v1/lists/{id}/bookmarks/{bookmarkId}` (`lists.removeFromList`): responds 204.
- `GET /api/v1/lists/{id}/bookmarks` (`bookmarks.getBookmarks` with that `listId`): responds 200 with `{"bookmarks", "nextCursor"}`. `limit` (1 to 100, default 20) and `sortOrder` (`desc` or `asc`) are read from the query; `includeContent` is ignored. `nextCursor` SHALL be an opaque string of URL-safe characters (letters, digits, `-` and `_`) when more bookmarks follow, else `null`, and passing it as the `cursor` query parameter returns the next page. An invalid `cursor` responds 400.

#### Scenario: Create a nested list
- **WHEN** Floccus creates the list "Bookmarks bar" with `parentId` the ID of the list "Floccus"
- **THEN** the server responds 201 and "Bookmarks bar" is the last node of the folder "Floccus"

#### Scenario: Page through a list
- **WHEN** the list "Later" has 25 links and Floccus calls `GET /api/v1/lists/{id}/bookmarks?includeContent=false&` and then again with the returned `cursor`
- **THEN** the first response has 20 bookmarks and a string `nextCursor`, and the second has the other 5 and `nextCursor` `null`

#### Scenario: Move a bookmark into a list
- **WHEN** Floccus sends `PUT /api/v1/lists/{id}/bookmarks/{bookmarkId}` for a link in another folder
- **THEN** the server responds 204 and the link is the first node of the list's folder

#### Scenario: Remove a bookmark from a list
- **WHEN** Floccus sends `DELETE /api/v1/lists/{id}/bookmarks/{bookmarkId}` for a link in that list
- **THEN** the server responds 204 and the link is the first node of the token's root folder

#### Scenario: Rename and move a list
- **WHEN** Floccus sends `PATCH /api/v1/lists/{id}` with `{"name": "Work", "parentId": "<id of Floccus>"}`
- **THEN** the folder is called "Work", is the last node of "Floccus", and the server responds 200 with the list
