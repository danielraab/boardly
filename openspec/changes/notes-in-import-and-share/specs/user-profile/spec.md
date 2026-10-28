# Spec Delta

## ADDED Requirements

### Requirement: Default category for new notes
Every user SHALL have an optional default category for new notes, stored on their account. `GET /api/me` SHALL return it as `noteCategoryId`, the ID of a category of the user, or `null` when none is set (new notes are uncategorized). The Profile tab SHALL show a "Default category for new notes" / "Standardkategorie für neue Notizen" category picker with the user's categories as a tree and the choice "No category" / "Keine Kategorie", a hint that notes saved from the share page start in this category, and a save action. Choosing "No category" SHALL clear the setting. When the default category is deleted, the setting SHALL be cleared; deleting its parent category keeps it, since the subcategories move up a level. All text SHALL be translated (English and German).

#### Scenario: Choose a default category
- **WHEN** the user picks "Inbox" in the Profile tab and saves
- **THEN** `GET /api/me` returns the ID of "Inbox" as `noteCategoryId`, and the note form of `/share` preselects "Inbox"

#### Scenario: Back to no category
- **WHEN** the default category is "Inbox" and the user picks "No category" and saves
- **THEN** `noteCategoryId` is `null`

#### Scenario: Default category deleted
- **WHEN** the default category "Inbox" is deleted on `/notes`
- **THEN** `noteCategoryId` is `null`

### Requirement: Default category validation
`PATCH /api/me` with `{"noteCategoryId": ...}` SHALL accept the ID of a category of the user or `null`. An ID that isn't a category of the user (unknown or another user's) or a value that isn't a string or `null` SHALL be rejected with 400 and `error.code` `invalid_category`, and the setting stays unchanged. On success it responds 200 with the updated user.

#### Scenario: Another user's category
- **WHEN** Ann sends `PATCH /api/me` with the ID of one of Bob's categories as `noteCategoryId`
- **THEN** the server responds 400 with `invalid_category` and `noteCategoryId` is unchanged

#### Scenario: Clear
- **WHEN** a client sends `PATCH /api/me` with `{"noteCategoryId": null}`
- **THEN** the server responds 200 and `noteCategoryId` is `null`
