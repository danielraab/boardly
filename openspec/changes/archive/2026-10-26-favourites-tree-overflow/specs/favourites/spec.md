# Spec Delta

## ADDED Requirements

### Requirement: Deeply nested trees stay readable
On `/favourites`, every name in the tree SHALL keep a minimum width (about 6rem below the `sm` breakpoint, 8rem from it; shorter names keep their own width) and a maximum width of 24rem; a name that doesn't fit SHALL be cut off with "…", and its full text SHALL be shown as a tooltip. The action buttons of a row SHALL come after its name, link count and tags and never overlap them. When a link has more tags than fit, the tags SHALL give way before the name: the name keeps its width (up to its maximum), only the tags that fit completely are shown, and all tags are listed in a tooltip. Each row SHALL be at least as wide as the tree; a row that needs more room SHALL be wider, and then the tree SHALL scroll horizontally on its own, while the page SHALL NOT scroll horizontally. Rows that fit SHALL keep their action buttons at the right edge of the tree. Below the `sm` breakpoint the indentation per level SHALL be half of the indentation on wider screens, and the drop indicator of drag and drop SHALL use the same indentation as the rows. Drag and drop SHALL also work while the tree is scrolled sideways.

#### Scenario: Deep nesting on a phone
- **WHEN** the tree has ten nested folders and is shown 375px wide
- **THEN** every folder name is at least partly visible, the tree can be scrolled sideways, and the page has no horizontal scrollbar

#### Scenario: Buttons don't overlap
- **WHEN** a deep folder row is wider than the tree
- **THEN** its action buttons come after its name and link count, and scrolling the tree sideways shows them

#### Scenario: Drag while scrolled
- **WHEN** the tree is scrolled sideways and the user drags a deep link onto a folder
- **THEN** the link is moved into that folder

#### Scenario: Shallow tree
- **WHEN** all rows fit into the tree's width
- **THEN** the tree has no horizontal scrollbar and looks as before

#### Scenario: Very long name
- **WHEN** a link's title is 300 characters long and the page is wide
- **THEN** the title is shown at most 24rem wide and ends with "…", its tags follow right after it, and hovering it shows the full title

#### Scenario: Many tags
- **WHEN** a link titled "Tagged" has twelve tags and the row is too narrow for all of them
- **THEN** "Tagged" is shown in full, as many whole tags as fit follow it, the rest are hidden, and hovering the tags lists all twelve

#### Scenario: Smaller indentation on phones
- **WHEN** the page is narrower than the `sm` breakpoint
- **THEN** each level is indented half as much as on a wide screen
