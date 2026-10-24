# Spec Delta

## MODIFIED Requirements

### Requirement: Favourites page
`/favourites` SHALL require a signed-in user. It SHALL show the user's tree with folders that can be collapsed and expanded, and actions to add a folder or link to a folder and to edit and delete each node. Each link SHALL also have a "Refresh icon" / "Icon neu laden" action that refreshes the favicon of the link's site (see "Manual favicon refresh" in the images spec) and then shows the new icon. While the refresh runs, the action is disabled. When no favicon is found, the address is blocked or too many refreshes were made, the page shows a translated message and the link keeps its icon. All of its text SHALL be translated (English and German).

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/favourites`
- **THEN** the browser shows `/login`

#### Scenario: Collapse a folder
- **WHEN** the user collapses the folder "Work"
- **THEN** the children of "Work" are hidden, and expanding it shows them again

#### Scenario: Refresh an icon
- **WHEN** the user chooses "Refresh icon" on the link "GitHub" and the site's favicon is found
- **THEN** the link shows the refreshed favicon

#### Scenario: Refresh finds no icon
- **WHEN** the user chooses "Refresh icon" on a link whose site has no favicon
- **THEN** the page shows that no icon was found, and the link keeps its icon
