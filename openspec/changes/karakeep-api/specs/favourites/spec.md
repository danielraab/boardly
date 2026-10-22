# Spec Delta

## MODIFIED Requirements

### Requirement: Sibling order
The children of a folder SHALL keep the order the user gave them. A new node SHALL be added at the end of its folder, except a link saved through a connected app, which SHALL be added first in the token's root folder.

#### Scenario: New node goes last
- **WHEN** the user adds a link to a folder that already contains "A" and "B"
- **THEN** the folder shows "A", "B" and then the new link

#### Scenario: Order survives a reload
- **WHEN** the user reorders the children of a folder and reloads the page
- **THEN** the folder shows the children in the new order

#### Scenario: Link saved from a connected app goes first
- **WHEN** the browser extension saves a page with a token rooted at "Reading", which already contains "A" and "B"
- **THEN** "Reading" shows the new link, then "A" and "B"

