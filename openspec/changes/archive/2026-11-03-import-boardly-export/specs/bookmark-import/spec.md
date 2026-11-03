# Spec Delta

## MODIFIED Requirements

### Requirement: Import form
The "Import / Export" settings tab SHALL show an "Import" / "Importieren" section with:

- the format: "Karakeep (JSON)", preselected, "Linkwarden (JSON)", "Bookmarks file (HTML)" / "Lesezeichen-Datei (HTML)" or "Boardly export (JSON or ZIP)" / "Boardly-Export (JSON oder ZIP)", with the logo of Karakeep and Linkwarden next to their options, and a hint that Karakeep's JSON export keeps lists while its HTML export doesn't,
- a file input that accepts JSON, HTML and ZIP files,
- the import folder, picked from the favourites tree, folders only, including the top-level folder, with no folder selected at first,
- for the Karakeep format only, the import category for text bookmarks, picked from the user's categories as a tree or "Top level" / "Oberste Ebene", with "Top level" selected at first and a hint that Karakeep text bookmarks become notes,
- a checkbox "Skip links and notes that are already saved" / "Bereits gespeicherte Links und Notizen überspringen", checked at first, that sends `skipDuplicates` `true` when checked and `false` when not,
- an "Import" / "Importieren" button, disabled until a file and an import folder are chosen.

While the import runs, the button is disabled and shows that the import is in progress. Afterwards the section SHALL show the report: the number of imported links and created folders, the number of imported notes and created categories when not zero, and, when not zero, the number of duplicate links, duplicate notes, invalid URLs, entries that are not links, smart lists, notes too large to import and shortened descriptions. Errors are shown in the section with a message for each error code. After a successful import, the favourites tree, the notes and the categories are reloaded wherever they are shown. For the Boardly format, the parts, the import folder and category, the request and the report follow the `data-import` capability. The section "Export" of the same tab is specified in the `data-export` capability.

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

#### Scenario: Section title
- **WHEN** the user opens the "Import / Export" tab
- **THEN** the import section is titled "Import"

#### Scenario: Boardly format offered
- **WHEN** the user opens the format choice of the import section
- **THEN** "Boardly export (JSON or ZIP)" is one of the formats, and the file input accepts `.zip` files
