# Spec Delta

## ADDED Requirements

### Requirement: Create folder from the share page
With "Link" on the share page, the page SHALL show a "New folder" / "Neuer Ordner" button below the folder choice. It SHALL open the same dialog for a new folder as the favourites page, creating the folder inside the folder currently chosen in the form, or at the top level of the favourites when the root is chosen, with the same validation. After the folder is saved, the dialog closes and the new folder is chosen in the form. Like any other folder choice on the share page, this applies only to the next save and SHALL NOT change the default folder for new links. Closing the dialog without saving SHALL leave the form and all its values as they were.

#### Scenario: Create a subfolder
- **WHEN** a signed-in user on `/share` chooses the folder "Work", presses "New folder" and saves the folder "Reading"
- **THEN** "Work" contains the folder "Reading", and "Reading" is chosen in the form

#### Scenario: Create a top-level folder
- **WHEN** the root folder is chosen and the user creates the folder "Inbox" from the share page
- **THEN** "Inbox" is a top-level folder of the favourites and is chosen in the form

#### Scenario: Save into the new folder
- **WHEN** the user creates the folder "Reading" from the share page and then saves the link
- **THEN** "Reading" contains the link, and the confirmation names "Reading"

#### Scenario: Default folder unchanged
- **WHEN** the default folder is "Inbox", the user creates and saves into the folder "Reading" and opens `/share` again
- **THEN** "Inbox" is preselected

#### Scenario: Cancel keeps the form
- **WHEN** the user has entered a URL, a title and a tag, opens "New folder" and cancels the dialog
- **THEN** no folder is created, and the URL, title, tag and chosen folder are unchanged

### Requirement: Create category from the share page
With "Note" on the share page, the page SHALL show a "New category" / "Neue Kategorie" button below the category choice. It SHALL open the same dialog for a new category as the notes page, with the category currently chosen in the form preselected as the parent, or "None" when "No category" is chosen; the parent can be changed in the dialog. After the category is saved, the dialog closes and the new category is chosen in the form. Like any other category choice on the share page, this applies only to the next save and SHALL NOT change the default category for new notes. Closing the dialog without saving SHALL leave the form and all its values as they were.

#### Scenario: Create a subcategory
- **WHEN** a signed-in user on `/share` with "Note" chooses the category "Home", presses "New category" and saves the category "Garden" with the preselected parent
- **THEN** "Garden" is a subcategory of "Home", and "Garden" is chosen in the form

#### Scenario: Create a top-level category
- **WHEN** "No category" is chosen and the user presses "New category"
- **THEN** the dialog's parent is "None", and the saved category is a top-level category chosen in the form

#### Scenario: Save into the new category
- **WHEN** the user creates the category "Garden" from the share page and then saves the note
- **THEN** "Garden" contains the note, and the confirmation names the category

#### Scenario: Default category unchanged
- **WHEN** the default category is "Inbox", the user creates and saves into the category "Garden" and opens `/share` again
- **THEN** "Inbox" is preselected in the note form

#### Scenario: Cancel keeps the form
- **WHEN** the user has entered a note title and text, opens "New category" and cancels the dialog
- **THEN** no category is created, and the title, text and chosen category are unchanged
