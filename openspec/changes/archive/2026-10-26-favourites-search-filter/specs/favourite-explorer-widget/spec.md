# Spec Delta

## MODIFIED Requirements

### Requirement: Explorer title
An explorer with a title SHALL show it as a heading at its top, cut off with "…" when it doesn't fit, with the expand-all button on the same line. Its heading level is that of a section's title at the same place in the layout. An explorer without a title shows no heading. When the signed-in owner views their dashboard outside edit mode and the root folder exists, the title SHALL be a link to `/favourites?folder=<root folder ID>` (to `/favourites` when the root folder is the tree's root), opening in the same tab. For visitors, in edit mode, or when the root folder no longer exists, the title is plain text.

#### Scenario: With title
- **WHEN** Ann gives an explorer the title "Work links"
- **THEN** the explorer shows the heading "Work links" above its tree

#### Scenario: Without title
- **WHEN** an explorer has no title
- **THEN** the tree starts at the top of the widget

#### Scenario: Owner follows the title
- **WHEN** Ann views her dashboard with an explorer titled "Work links" for the folder "Work" and activates the title
- **THEN** the browser shows `/favourites` with "Work" expanded, scrolled into view and highlighted

#### Scenario: Tree root
- **WHEN** the explorer's root folder is the tree's root and Ann activates its title
- **THEN** the browser shows `/favourites`

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with an explorer titled "Work links"
- **THEN** the title is not a link

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** the explorer's title is not a link
