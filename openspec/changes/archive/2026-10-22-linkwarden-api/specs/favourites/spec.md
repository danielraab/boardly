## ADDED Requirements

### Requirement: Integer IDs for apps
Every folder, link and tag SHALL have an integer ID next to its ID, for apps whose clients expect numbers (see the Linkwarden API). The integer ID is a positive number, unique among all nodes or among all tags, assigned when the node or tag is created and never changed or reused, also after the node or tag is deleted. Existing nodes and tags get one when the server is upgraded. Moving, renaming or editing a node keeps its integer ID. The web app's own API doesn't show integer IDs.

#### Scenario: Stable after a move
- **WHEN** a link with integer ID 7 is moved to another folder and renamed
- **THEN** it still has integer ID 7

#### Scenario: Not reused
- **WHEN** the link with the highest integer ID is deleted and a new link is created
- **THEN** the new link gets a higher integer ID than the deleted one

#### Scenario: Existing data
- **WHEN** the server is upgraded and a user already has folders, links and tags
- **THEN** each of them has a distinct integer ID
