# bookmark-import Specification

## Purpose
Defines the one-time import of Karakeep JSON, Linkwarden JSON and Netscape bookmark HTML files into a folder of the user's favourites tree: the import API, how each format maps onto folders, links and tags, duplicate handling, the report and the import form in the settings. The Linkwarden format is not yet verified against a real export.

## Requirements

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

### Requirement: Import folder
Every import SHALL go into an import folder that the user picks. The import folder SHALL be a folder in the user's own tree, and the tree root is allowed. A missing `folderId`, a node of another user, or a link fails with `400` and code `invalid_folder`, and nothing is imported. The source's top-level folders and links are placed directly in the import folder.

#### Scenario: Into the root
- **WHEN** the user imports a file with the tree root as import folder
- **THEN** the source's top-level folders and links are placed directly in the root

#### Scenario: Into a subfolder
- **WHEN** the user imports a file with the folder "Imported" as import folder
- **THEN** the source's top-level folders and links are placed in "Imported", and nothing outside "Imported" changes except tags

#### Scenario: Link as import folder
- **WHEN** the user posts the ID of a link as `folderId`
- **THEN** the response is `400` with code `invalid_folder`

### Requirement: Folder structure
The import SHALL recreate the source's folder structure below the import folder. At each level, a source folder whose name equals the name of an existing child folder, ignoring case, SHALL reuse that folder. Otherwise a new folder is created. Two source folders with the same name, ignoring case, under the same parent become one folder. Folders without links are created too. A folder name longer than 100 characters is cut to 100 characters. A folder with an empty name is not created, and its contents go into its parent. New folders get the import time as their creation time.

#### Scenario: Existing folder reused
- **WHEN** the import folder contains the folder "Reading" and the file has the folder "reading" with the subfolder "Later"
- **THEN** no second "Reading" is created, and "Later" is created inside the existing "Reading"

#### Scenario: Folder without links
- **WHEN** the file has a folder "Archive" that contains only the subfolder "2024" with links
- **THEN** "Archive" and "Archive/2024" are created

#### Scenario: Same name at different levels
- **WHEN** the file has a top-level folder "AI" and a folder "Dev/AI"
- **THEN** both are created as separate folders

### Requirement: Imported links
Each imported link SHALL keep its URL, title, tags, description and creation date from the source, following the rules of link nodes. URLs are normalized as for any link, and an entry with an invalid or non-`http(s)` URL is skipped and counted as an invalid URL. A title longer than 1000 characters is cut to 1000 characters, and an empty title becomes no title. Tags are attached as for any link: an existing tag of the user is matched regardless of case, and an empty tag or a tag longer than 100 characters is left out. The link's creation time is the source's creation date, or the import time when the source has none or it can't be read.

#### Scenario: Creation date kept
- **WHEN** a Karakeep bookmark has `createdAt` `1790365701`
- **THEN** the imported link's creation time is 2026-09-25T19:48:21Z, and the Karakeep and Linkwarden APIs sort it by that time

#### Scenario: Existing tag in another case
- **WHEN** the user has the tag "Dev" and an imported link has the tag "dev"
- **THEN** the link gets the existing tag "Dev"

#### Scenario: Invalid URL
- **WHEN** an entry has the URL `javascript:alert(1)`
- **THEN** it is not imported and is counted as an invalid URL

### Requirement: Import order
Within each folder, the imported nodes SHALL keep the order of the file and SHALL be placed after the folder's existing children. In a Karakeep or Linkwarden JSON file, a folder's subfolders come first, in the order the file lists the lists or collections, followed by its links in the order of the file. In a Netscape HTML file, folders and links keep their order in the document.

#### Scenario: Newest first, as exported
- **WHEN** a Karakeep export lists bookmark A before bookmark B, both in the list "Dev"
- **THEN** A is above B in "Dev"

#### Scenario: After existing content
- **WHEN** the import folder already contains the link "Home" and the file has two top-level links
- **THEN** "Home" stays first and the two links follow it

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

### Requirement: Description from notes
The description of an imported link SHALL be made of the source's description, then `Note: ` followed by the note, then `Summary: ` followed by the summary. Parts that are missing or empty are left out, and the parts are separated by a blank line. When the result is longer than 2048 characters, it is cut to 2047 characters followed by `…`, and the report counts it as a shortened description.

#### Scenario: Karakeep note
- **WHEN** a Karakeep bookmark has the note "login with the family account" and no summary
- **THEN** the link's description is "Note: login with the family account"

#### Scenario: Description and note
- **WHEN** an entry has the description "A tour of Go" and the note "Read later"
- **THEN** the link's description is "A tour of Go", a blank line, then "Note: Read later"

#### Scenario: Too long
- **WHEN** the description and note together have 3000 characters
- **THEN** the link's description has 2048 characters and ends with "…", and the report counts one shortened description

### Requirement: Karakeep JSON format
A Karakeep JSON export SHALL be read as an object with `bookmarks` and `lists`:

- A list is `{"id", "name", "parentId", "type"}`. A list of `type` `manual` becomes a folder, and its parent is the list with the ID in `parentId`, or the import folder when `parentId` is `null` or unknown. A parent may appear after its child in the file. A list of `type` `smart` is not created and is counted as a smart list. Manual lists also become categories for notes (see "Categories from lists").
- A bookmark is `{"createdAt", "title", "tags", "lists", "content", "note", "summary", "archived"}`. `createdAt` is in Unix seconds, `tags` are tag names, and `lists` are list IDs. A bookmark with `content.type` `link` becomes a link with the URL `content.url`. It goes into the folder of the first ID in `lists` that is a manual list, or into the import folder when there is none.
- A bookmark with `content.type` `text` becomes a note with the Markdown text `content.text` (see "Imported notes"). It goes into the category of the first ID in `lists` that is a manual list, or into the import category when there is none.
- A bookmark of another `content.type`, such as `asset`, is not imported and is counted as "not a link".
- For a link, `note` and, when present, `summary` go into the description (see "Description from notes"), and `archived` is ignored. For a text bookmark, `archived` `true` makes the note archived (see "Imported notes"); a missing `archived` counts as `false`. `favourited`, which Karakeep doesn't export, is ignored for every bookmark.

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

#### Scenario: Archived text bookmark
- **WHEN** a text bookmark has `archived` `true`
- **THEN** its note is imported archived, into the category it would go to anyway

### Requirement: Linkwarden JSON format
A Linkwarden JSON export SHALL be read as an object with a `collections` array:

- A collection is `{"id", "name", "parentId", "links"}` and becomes a folder. Its parent is the collection with the ID in `parentId`, or the import folder when `parentId` is `null` or unknown. A parent may appear after its child in the file.
- A link is `{"name", "url", "description", "createdAt", "tags"}`. `name` is the title, `createdAt` is an ISO 8601 time, and `tags` are `{"name"}` objects. The link goes into its collection's folder.
- A link without an `http(s)` URL, such as an uploaded PDF or image, is not imported and is counted as "not a link". Pins and other Linkwarden fields are ignored.

Unknown fields SHALL be ignored. A file that isn't a JSON object with a `collections` array is not a Linkwarden export.

#### Scenario: Collection with sub-collection
- **WHEN** the file has the collection "Reading" with a link, and the collection "Later" with `parentId` of "Reading"
- **THEN** the folders "Reading" and "Reading/Later" are created, and the link is in "Reading"

#### Scenario: Description and tags
- **WHEN** a Linkwarden link has the description "A tour of Go" and the tags "go" and "docs"
- **THEN** the imported link has that description and both tags

#### Scenario: Uploaded PDF
- **WHEN** a Linkwarden link has no URL because it is an uploaded PDF
- **THEN** it is not imported and is counted as "not a link"

### Requirement: Netscape HTML format
A Netscape bookmark HTML file SHALL be read as nested `<DL>` lists, as exported by Karakeep, Linkwarden and web browsers:

- `<H3>` starts a folder named by its text, whose contents are the `<DL>` that follows it.
- `<A>` is a link with the URL `HREF`, its text as title, `ADD_DATE` (Unix seconds) as creation date and `TAGS` (comma-separated) as tags.
- A `<DD>` directly after an `<A>` is the link's description.

Tags and attributes it doesn't know, such as `ICON` and `LAST_MODIFIED`, and any content outside the bookmark lists, such as scripts added by a proxy, SHALL be ignored. A file without the Netscape doctype and without any `<DT>` entries is not a Netscape bookmark file.

#### Scenario: Flat Karakeep HTML export
- **WHEN** the file is a Karakeep HTML export with links and `TAGS`, but no `<H3>`
- **THEN** all links are imported directly into the import folder with their tags and dates

#### Scenario: Browser export with folders
- **WHEN** the file has an `<H3>` "Toolbar" with a link and a nested `<H3>` "Dev" with a link
- **THEN** the folders "Toolbar" and "Toolbar/Dev" are created, each with its link

#### Scenario: Script after the list
- **WHEN** the file ends with a `<script>` element after the last `</DL>`
- **THEN** the import ignores it and imports the links

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

### Requirement: Fetches for imported links
Every imported link SHALL get the same favicon and page-metadata fetches as a link created in Boardly. As for any link, fetched values only fill empty fields, so an imported description is kept. The fetches run in the background after the import, so the response doesn't wait for them.

#### Scenario: Preview after import
- **WHEN** a link without a description is imported
- **THEN** its description and preview image are later filled from the page's metadata

#### Scenario: Imported description kept
- **WHEN** a link with the description "Note: Read later" is imported
- **THEN** the metadata fetch doesn't replace that description

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
- no icon and no colour, not pinned, and archived when the bookmark is archived,
- the bookmark's creation date as creation time and time of last change, or the import time when the bookmark has none or it can't be read,
- version 1.

A note whose content would be larger than the document size limit is not imported and is counted as a note too large to import. Within the import, notes are created in the order of the file.

#### Scenario: Text bookmark with tags
- **WHEN** a text bookmark has the title "Pasta", the text "- [ ] buy basil", the tags "cooking" and "weekend plans", and `createdAt` `1790365701`
- **THEN** the note "Pasta" has a task list with an unchecked "buy basil", then a paragraph "#cooking #weekend-plans", and its creation time and time of last change are 2026-09-25T19:48:21Z

#### Scenario: Karakeep note on a text bookmark
- **WHEN** a text bookmark has the text "Door code 1234" and the note "ask Bob first"
- **THEN** the note's content is a paragraph "Door code 1234" followed by a paragraph "Note: ask Bob first"

#### Scenario: Archived note
- **WHEN** a text bookmark with the text "old plan" has `archived` `true`
- **THEN** the note is archived, isn't pinned, and is shown with "Archived" on `/notes`

#### Scenario: Archived duplicate
- **WHEN** `skipDuplicates` is `true`, the user has a note that isn't archived, and the file has the same note as an archived text bookmark
- **THEN** the bookmark is skipped as a duplicate, and the existing note stays unarchived

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
