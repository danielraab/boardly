# Spec Delta

## ADDED Requirements

### Requirement: Default folder for new links
Every user SHALL have an optional default folder for new links, stored on their account. `GET /api/me` SHALL return it as `shareFolderId`, the ID of a folder of the user, or `null` when none is set (the root folder is used). The Profile tab SHALL show a "Default folder for new links" / "Standardordner für neue Links" folder picker with a hint that the "Save link" page starts in this folder, and a save action. Choosing the root folder SHALL clear the setting. When the default folder is deleted, the setting SHALL be cleared. All text SHALL be translated (English and German).

#### Scenario: Choose a default folder
- **WHEN** the user picks "Inbox" in the Profile tab and saves
- **THEN** `GET /api/me` returns the ID of "Inbox" as `shareFolderId`, and `/share` preselects "Inbox"

#### Scenario: Back to the root
- **WHEN** the default folder is "Inbox" and the user picks the root folder and saves
- **THEN** `shareFolderId` is `null` and `/share` preselects the root

#### Scenario: Default folder deleted
- **WHEN** the default folder "Inbox" is deleted on `/favourites`
- **THEN** `shareFolderId` is `null` and `/share` preselects the root

#### Scenario: Every device
- **WHEN** the user chose "Inbox" in one browser and opens `/share` in another browser
- **THEN** "Inbox" is preselected

### Requirement: Default folder validation
`PATCH /api/me` with `{"shareFolderId": ...}` SHALL accept the ID of a folder of the user or `null`. The ID of the user's root folder SHALL be stored as `null`. An ID that isn't a folder of the user (unknown, a link, or another user's node) or a value that isn't a string or `null` SHALL be rejected with 400 and `error.code` `invalid_folder`, and the setting stays unchanged. On success it responds 200 with the updated user.

#### Scenario: Link instead of folder
- **WHEN** a client sends `PATCH /api/me` with the ID of one of the user's links as `shareFolderId`
- **THEN** the server responds 400 with `invalid_folder` and `shareFolderId` is unchanged

#### Scenario: Another user's folder
- **WHEN** Ann sends `PATCH /api/me` with the ID of one of Bob's folders as `shareFolderId`
- **THEN** the server responds 400 with `invalid_folder`

#### Scenario: Clear
- **WHEN** a client sends `PATCH /api/me` with `{"shareFolderId": null}`
- **THEN** the server responds 200 and `shareFolderId` is `null`

## MODIFIED Requirements

### Requirement: Edit user slug
The user SHALL be able to change their slug in the Profile tab. The web app saves it with `PATCH /api/me` and `{"slug": ...}`. The server lowercases it and checks the format (`invalid_slug`), the reserved words (`slug_reserved`) and uniqueness across users (409 `slug_taken`). `PATCH /api/me` accepts `name`, `slug` and `shareFolderId` alone or together. A request with none of them is rejected with 400 `invalid_request`. While the field differs from the saved slug, the tab shows a hint that links using the old address stop working. Saving then asks for confirmation, showing an example of the old and the new address. Old addresses are not redirected.

#### Scenario: Change slug
- **WHEN** Ann changes her slug from `ann` to `annsmith`, saves and confirms
- **THEN** her dashboards are at `/annsmith/…` and `/ann/home` shows not found

#### Scenario: Slug taken
- **WHEN** Ann tries to save the slug `bob`, which Bob uses
- **THEN** the field shows that the slug is taken and her slug is unchanged

#### Scenario: Cancel confirmation
- **WHEN** Ann changes her slug, saves and cancels the confirmation
- **THEN** nothing is saved and the field still shows her edit
