# Spec Delta

## MODIFIED Requirements

### Requirement: Import endpoint
`POST /api/favourites/import` SHALL import a bookmark file into the signed-in user's favourites tree, and the notes of the file into their notes. The request body is the file itself, sent as `application/octet-stream` like image uploads, so it can't be posted from another site without a CORS preflight. The query parameters are:

- `format`: `karakeep` (Karakeep JSON export), `linkwarden` (Linkwarden JSON export) or `netscape` (Netscape bookmark HTML),
- `folderId`: the ID of the import folder,
- `categoryId`: the ID of the import category, optional. Missing or empty means the top level,
- `skipDuplicates`: `true` or `false`, optional. Missing means `true`.

The import is one-time: nothing is remembered about the source, and importing the same file again only adds what the duplicate rules let through. The file SHALL be at most 10 MB, and a larger body fails with `413` and code `file_too_large`. A missing or unknown `format`, a `skipDuplicates` other than `true` or `false`, or an empty body, fails with `400` and code `invalid_request`. A file that can't be read as the chosen format fails with `400` and code `invalid_import_file`. Form posts (`multipart/form-data` and other form types) are rejected with `403` like for every API endpoint. The endpoint requires a signed-in user.

#### Scenario: Import a Karakeep file
- **WHEN** a signed-in user posts a Karakeep JSON export with `format` `karakeep` and the ID of their root folder
- **THEN** the response is `200` with the import report, the links appear in the favourites tree and the text bookmarks appear as notes

#### Scenario: Wrong format
- **WHEN** the user posts a Netscape HTML file with `format` `karakeep`
- **THEN** the response is `400` with code `invalid_import_file`, and the tree and the notes are unchanged

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

### Requirement: Karakeep JSON format
A Karakeep JSON export SHALL be read as an object with `bookmarks` and `lists`:

- A list is `{"id", "name", "parentId", "type"}`. A list of `type` `manual` becomes a folder, and its parent is the list with the ID in `parentId`, or the import folder when `parentId` is `null` or unknown. A parent may appear after its child in the file. A list of `type` `smart` is not created and is counted as a smart list. Manual lists also become categories for notes (see "Categories from lists").
- A bookmark is `{"createdAt", "title", "tags", "lists", "content", "note", "summary", "archived"}`. `createdAt` is in Unix seconds, `tags` are tag names, and `lists` are list IDs. A bookmark with `content.type` `link` becomes a link with the URL `content.url`. It goes into the folder of the first ID in `lists` that is a manual list, or into the import folder when there is none.
- A bookmark with `content.type` `text` becomes a note with the Markdown text `content.text` (see "Imported notes"). It goes into the category of the first ID in `lists` that is a manual list, or into the import category when there is none.
- A bookmark of another `content.type`, such as `asset`, is not imported and is counted as "not a link".
- For a link, `note` and, when present, `summary` go into the description (see "Description from notes"). `archived` and `favourited` are ignored.

Unknown fields SHALL be ignored. A file that isn't a JSON object with a `bookmarks` array is not a Karakeep export.

#### Scenario: Nested lists
- **WHEN** the file has the list "Containers" with `parentId` of "Docker", "Docker" with `parentId` of "Dev", and "Dev" with `parentId` `null`, in this order
- **THEN** the folders "Dev/Docker/Containers" are created below the import folder

#### Scenario: Bookmark in two lists
- **WHEN** a bookmark has `lists` with the IDs of "dev" and "Linux", in this order
- **THEN** the link is imported once, into "dev"

#### Scenario: Smart list
- **WHEN** the file has a smart list "Youtube" with the query `url:youtube`
- **THEN** no folder "Youtube" is created, and the report counts one smart list

#### Scenario: Text bookmark
- **WHEN** a bookmark has `content` `{"type": "text", "text": "serienjunkies.team"}` and no lists
- **THEN** a note with the content "serienjunkies.team" is created in the import category, and it is not counted as "not a link"

#### Scenario: Asset bookmark
- **WHEN** a bookmark has `content` `{"type": "asset", "assetType": "image"}`
- **THEN** it is not imported and is counted as "not a link"

#### Scenario: Archived bookmark
- **WHEN** a link bookmark has `archived` `true`
- **THEN** it is imported like any other link

### Requirement: All or nothing
An import SHALL run in a single transaction. When it fails for any reason, no folder, link, tag, category or note of the import is kept. Skipped entries don't make an import fail.

#### Scenario: Failure during the import
- **WHEN** writing to the database fails halfway through an import
- **THEN** the tree and the notes are unchanged and the response is an error

#### Scenario: Failure while adding notes
- **WHEN** the links of a Karakeep file were written and writing a note fails
- **THEN** none of the file's folders, links, categories or notes are kept

### Requirement: Import report
A successful import SHALL return `{"links", "folders", "notes", "categories", "skipped": {"duplicates", "duplicateNotes", "invalidUrls", "notLinks", "smartLists", "tooLargeNotes"}, "shortenedDescriptions"}`. `links` is the number of imported links, `folders` the number of created folders (not reused ones), `notes` the number of imported notes, `categories` the number of created categories (not reused ones), and the other fields are the counts defined in this spec.

#### Scenario: Report
- **WHEN** a Karakeep file with 166 link bookmarks, 2 text bookmarks and 2 smart lists is imported into an empty tree, and 1 link is a duplicate within the file
- **THEN** the report has `links` 165, `notes` 2, `skipped.duplicates` 1, `skipped.notLinks` 0 and `skipped.smartLists` 2

#### Scenario: File without notes
- **WHEN** a Netscape HTML file is imported
- **THEN** the report has `notes` 0, `categories` 0 and `skipped.duplicateNotes` 0

### Requirement: Import form
The "Import / Export" settings tab SHALL show an "Import bookmarks" / "Lesezeichen importieren" section with:

- the format: "Karakeep (JSON)", preselected, "Linkwarden (JSON)" or "Bookmarks file (HTML)" / "Lesezeichen-Datei (HTML)", with the logo of Karakeep and Linkwarden next to their options, and a hint that Karakeep's JSON export keeps lists while its HTML export doesn't,
- a file input,
- the import folder, picked from the favourites tree, folders only, including the top-level folder, with no folder selected at first,
- for the Karakeep format only, the import category for text bookmarks, picked from the user's categories as a tree or "Top level" / "Oberste Ebene", with "Top level" selected at first and a hint that Karakeep text bookmarks become notes,
- a checkbox "Skip links and notes that are already saved" / "Bereits gespeicherte Links und Notizen überspringen", checked at first, that sends `skipDuplicates` `true` when checked and `false` when not,
- an "Import" / "Importieren" button, disabled until a file and an import folder are chosen.

While the import runs, the button is disabled and shows that the import is in progress. Afterwards the section SHALL show the report: the number of imported links and created folders, the number of imported notes and created categories when not zero, and, when not zero, the number of duplicate links, duplicate notes, invalid URLs, entries that are not links, smart lists, notes too large to import and shortened descriptions. Errors are shown in the section with a message for each error code. After a successful import, the favourites tree, the notes and the categories are reloaded wherever they are shown. A section "Export" / "Export" is not part of this change.

#### Scenario: Button disabled without folder
- **WHEN** the user has chosen a file but no import folder
- **THEN** the "Import" button is disabled

#### Scenario: Successful import
- **WHEN** the user imports a Karakeep file into the top-level folder
- **THEN** the section shows how many links were imported and folders created, and the Favourites page shows the new links without a reload

#### Scenario: Notes in the report
- **WHEN** the user imports a Karakeep file with 3 text bookmarks in the list "Recipes"
- **THEN** the section shows 3 imported notes and 1 created category, and `/notes` shows them without a reload

#### Scenario: Category only for Karakeep
- **WHEN** the user chooses the format "Linkwarden (JSON)"
- **THEN** the import category is not shown, and the import is sent without `categoryId`

#### Scenario: Invalid file
- **WHEN** the import fails with `invalid_import_file`
- **THEN** the section shows that the file isn't a valid export of the chosen format, and nothing is imported

#### Scenario: Skip duplicates by default
- **WHEN** the user imports without touching the checkbox
- **THEN** the import is sent with `skipDuplicates` `true`

#### Scenario: Allow duplicates
- **WHEN** the user unchecks "Skip links and notes that are already saved" and imports
- **THEN** the import is sent with `skipDuplicates` `false`

## ADDED Requirements

### Requirement: Import category
Notes of an import SHALL go into an import category that the user picks, or the top level. The import category SHALL be a category of the user. An unknown category or one of another user fails with `400` and code `invalid_category`, and nothing is imported. With the top level, notes without a list are uncategorized and categories from lists are created at the top level. The import category is checked for every format, also when the file has no notes.

#### Scenario: Into the top level
- **WHEN** the user imports a Karakeep file without `categoryId`, with a text bookmark that has no list
- **THEN** the note is uncategorized

#### Scenario: Into a category
- **WHEN** the user imports with the category "Imported" as import category
- **THEN** notes without a list are in "Imported", and categories from lists are created below "Imported"

#### Scenario: Another user's category
- **WHEN** the user posts the ID of another user's category as `categoryId`
- **THEN** the response is `400` with code `invalid_category`, and nothing is imported

### Requirement: Imported notes
Each imported note SHALL have:
- the bookmark's title as title, trimmed and cut to 200 characters; an empty title becomes no title,
- as content, the Markdown text converted as in "Content from Markdown" in the note-document spec, followed by the bookmark's `note` as a paragraph starting with `Note: ` and its `summary` as a paragraph starting with `Summary: `, each only when not empty, and then a last paragraph with the bookmark's tags, each written as `#` followed by the tag name with whitespace replaced by `-`, separated by spaces, only when there are tags,
- no icon and no colour,
- the bookmark's creation date as creation time and time of last change, or the import time when the bookmark has none or it can't be read,
- version 1.

A note whose content would be larger than the document size limit is not imported and is counted as a note too large to import. Within the import, notes are created in the order of the file.

#### Scenario: Text bookmark with tags
- **WHEN** a text bookmark has the title "Pasta", the text "- [ ] buy basil", the tags "cooking" and "weekend plans", and `createdAt` `1790365701`
- **THEN** the note "Pasta" has a task list with an unchecked "buy basil", then a paragraph "#cooking #weekend-plans", and its creation time and time of last change are 2026-09-25T19:48:21Z

#### Scenario: Karakeep note on a text bookmark
- **WHEN** a text bookmark has the text "Door code 1234" and the note "ask Bob first"
- **THEN** the note's content is a paragraph "Door code 1234" followed by a paragraph "Note: ask Bob first"

#### Scenario: Too large
- **WHEN** a text bookmark has 700 KB of text
- **THEN** no note is created for it, and the report counts one note too large to import

### Requirement: Categories from lists
A manual list SHALL become a category below the import category when at least one imported note goes into it or into one of its sublists, at any depth. Lists without such notes create no category. The category tree follows the list tree: a list's category is below the category of its parent list. At each level, a list whose name equals the title of an existing child category, ignoring case, SHALL reuse that category; two lists with the same name, ignoring case, under the same parent become one category. A name longer than 200 characters is cut to 200 characters. A list with an empty name creates no category, and its notes go into its parent's category. New categories have no icon and no colour.

#### Scenario: Notes in a nested list
- **WHEN** the file has the list "Dev" with the sublist "Snippets", and one text bookmark is in "Snippets"
- **THEN** the categories "Dev" and "Dev/Snippets" are created below the import category, and the note is in "Snippets"

#### Scenario: List with only links
- **WHEN** the list "Reading" contains only link bookmarks
- **THEN** a folder "Reading" is created, but no category "Reading"

#### Scenario: Existing category reused
- **WHEN** the import category has the subcategory "Recipes" and a text bookmark is in the list "recipes"
- **THEN** no second category is created, and the note is in "Recipes"

### Requirement: Duplicate notes are skipped
When `skipDuplicates` is `true`, a note SHALL be skipped when one of the user's notes, in any category, or a note earlier in the same file has the same title and the same content, ignoring the IDs of tasks. A skipped duplicate doesn't change the existing note and is counted as a duplicate note. When `skipDuplicates` is `false`, notes SHALL NOT be compared, and every note is imported.

#### Scenario: Import the same file again
- **WHEN** the user imports the same Karakeep file with text bookmarks a second time
- **THEN** no note is added, and every text bookmark is counted as a duplicate note

#### Scenario: Same text, other title
- **WHEN** the user has the note "Shopping" with the text "milk" and the file has a text bookmark "Groceries" with the text "milk"
- **THEN** the note "Groceries" is imported

#### Scenario: Duplicates allowed
- **WHEN** the user imports the same file a second time with `skipDuplicates` `false`
- **THEN** every note is imported again into the reused categories
