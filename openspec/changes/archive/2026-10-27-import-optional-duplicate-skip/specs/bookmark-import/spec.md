# Spec Delta

## MODIFIED Requirements

### Requirement: Import endpoint
`POST /api/favourites/import` SHALL import a bookmark file into the signed-in user's favourites tree. The request body is the file itself, sent as `application/octet-stream` like image uploads, so it can't be posted from another site without a CORS preflight. The query parameters are:

- `format`: `karakeep` (Karakeep JSON export), `linkwarden` (Linkwarden JSON export) or `netscape` (Netscape bookmark HTML),
- `folderId`: the ID of the import folder,
- `skipDuplicates`: `true` or `false`, optional. Missing means `true`.

The import is one-time: nothing is remembered about the source, and importing the same file again only adds what the duplicate rule lets through. The file SHALL be at most 10 MB, and a larger body fails with `413` and code `file_too_large`. A missing or unknown `format`, a `skipDuplicates` other than `true` or `false`, or an empty body, fails with `400` and code `invalid_request`. A file that can't be read as the chosen format fails with `400` and code `invalid_import_file`. Form posts (`multipart/form-data` and other form types) are rejected with `403` like for every API endpoint. The endpoint requires a signed-in user.

#### Scenario: Import a Karakeep file
- **WHEN** a signed-in user posts a Karakeep JSON export with `format` `karakeep` and the ID of their root folder
- **THEN** the response is `200` with the import report, and the links appear in the favourites tree

#### Scenario: Wrong format
- **WHEN** the user posts a Netscape HTML file with `format` `karakeep`
- **THEN** the response is `400` with code `invalid_import_file`, and the tree is unchanged

#### Scenario: Invalid skipDuplicates
- **WHEN** the user posts a file with `skipDuplicates` `yes`
- **THEN** the response is `400` with code `invalid_request`, and the tree is unchanged

#### Scenario: Too large
- **WHEN** the user posts a file of 11 MB
- **THEN** the response is `413` with code `file_too_large`, and the tree is unchanged

#### Scenario: Form post
- **WHEN** a request sends the file as `multipart/form-data`
- **THEN** the response is `403` with code `forbidden_origin`, and the tree is unchanged

#### Scenario: Signed out
- **WHEN** a visitor without a session posts to the endpoint
- **THEN** the response is `401`

### Requirement: Duplicate URLs are skipped
When `skipDuplicates` is `true`, an entry SHALL be skipped when its URL is already a link anywhere in the user's tree, or when the same URL appears earlier in the same file. URLs are compared as in the apps' "already saved" check: without the fragment and without a trailing slash of the path. A skipped duplicate doesn't change the existing link. Its tags are not added to it, and it is counted as a duplicate. When `skipDuplicates` is `false`, URLs SHALL NOT be compared: every entry with a valid URL is imported, and no entry is counted as a duplicate.

#### Scenario: Already in another folder
- **WHEN** the tree has `https://go.dev/` in "Work" and the file has `https://go.dev` in another folder
- **THEN** the entry is skipped, the link in "Work" is unchanged and the report counts one duplicate

#### Scenario: Twice in the file
- **WHEN** the file has `https://example.org/a` twice
- **THEN** one link is imported, at the place of the first entry

#### Scenario: Import the same file again
- **WHEN** the user imports the same file a second time
- **THEN** no link is added, and every link entry is counted as a duplicate

#### Scenario: Parameter missing
- **WHEN** the user imports a file without `skipDuplicates` and one of its URLs is already in the tree
- **THEN** that entry is skipped and counted as a duplicate

#### Scenario: Duplicates allowed, already in another folder
- **WHEN** the tree has `https://go.dev/` in "Work" and the file has `https://go.dev` in another folder, imported with `skipDuplicates` `false`
- **THEN** a second link to `https://go.dev` is created, the link in "Work" is unchanged and the report counts no duplicates

#### Scenario: Duplicates allowed, twice in the file
- **WHEN** the file has `https://example.org/a` twice and is imported with `skipDuplicates` `false`
- **THEN** both links are imported, in the order of the file

#### Scenario: Duplicates allowed, same file again
- **WHEN** the user imports the same file a second time with `skipDuplicates` `false`
- **THEN** every link is imported again into the reused folders, so each folder holds every link twice

### Requirement: Import form
The "Import / Export" settings tab SHALL show an "Import bookmarks" / "Lesezeichen importieren" section with:

- the format: "Karakeep (JSON)", preselected, "Linkwarden (JSON)" or "Bookmarks file (HTML)" / "Lesezeichen-Datei (HTML)", with the logo of Karakeep and Linkwarden next to their options, and a hint that Karakeep's JSON export keeps lists while its HTML export doesn't,
- a file input,
- the import folder, picked from the favourites tree, folders only, including the top-level folder, with no folder selected at first,
- a checkbox "Skip links that are already saved" / "Bereits gespeicherte Links überspringen", checked at first, that sends `skipDuplicates` `true` when checked and `false` when not,
- an "Import" / "Importieren" button, disabled until a file and an import folder are chosen.

While the import runs, the button is disabled and shows that the import is in progress. Afterwards the section SHALL show the report: the number of imported links and created folders, and, when not zero, the number of duplicates, invalid URLs, entries that are not links, smart lists and shortened descriptions. Errors are shown in the section with a message for each error code. After a successful import, the favourites tree is reloaded wherever it is shown. A section "Export" / "Export" is not part of this change.

#### Scenario: Button disabled without folder
- **WHEN** the user has chosen a file but no import folder
- **THEN** the "Import" button is disabled

#### Scenario: Successful import
- **WHEN** the user imports a Karakeep file into the top-level folder
- **THEN** the section shows how many links were imported and folders created, and the Favourites page shows the new links without a reload

#### Scenario: Invalid file
- **WHEN** the import fails with `invalid_import_file`
- **THEN** the section shows that the file isn't a valid export of the chosen format, and nothing is imported

#### Scenario: Skip duplicates by default
- **WHEN** the user imports without touching the checkbox
- **THEN** the import is sent with `skipDuplicates` `true`

#### Scenario: Allow duplicates
- **WHEN** the user unchecks "Skip links that are already saved" and imports
- **THEN** the import is sent with `skipDuplicates` `false`
