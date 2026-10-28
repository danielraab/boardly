# Spec Delta

## MODIFIED Requirements

### Requirement: Delete categories
Deleting a category SHALL make every note in it uncategorized, without changing those notes' time of last change, and SHALL move its direct subcategories to its parent, or to the top level when it has none. Notes in the subcategories keep their category. Every active app token whose root category is the deleted category SHALL be revoked; tokens rooted at its subcategories keep working. The web app SHALL ask for confirmation and, when the category has notes or subcategories, say how many notes become uncategorized and that the subcategories move up. When active app tokens use the category as their root category, the confirmation SHALL also say how many connected apps use it and that they will be revoked. Revoked tokens stay in the user's token list, and their root category is shown as "Deleted category" / "Gelöschte Kategorie".

#### Scenario: Delete a category with notes
- **WHEN** Ann deletes "Work", which has 3 notes, and confirms
- **THEN** the 3 notes have no category, and their order on `/notes` is unchanged

#### Scenario: Delete a category with subcategories
- **WHEN** Ann deletes "Home", whose subcategory "Garden" has 2 notes
- **THEN** "Garden" becomes a top-level category and keeps its 2 notes

#### Scenario: Delete revokes tokens
- **WHEN** Ann deletes "Inbox", which is the root category of her active Karakeep token "Phone", and confirms
- **THEN** "Phone" is revoked, requests with it are rejected, and the "Connected apps" tab lists it with the root category "Deleted category"

#### Scenario: Warning about connected apps
- **WHEN** Ann starts deleting "Inbox", which is the root category of 2 active tokens and 1 revoked token
- **THEN** the confirmation says that 2 connected apps use this category and will be revoked

#### Scenario: Subcategory root keeps working
- **WHEN** a token has the root category "Home/Garden" and Ann deletes "Home"
- **THEN** the token stays active, and "Garden" is now a top-level category
