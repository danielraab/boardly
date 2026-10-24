# Spec Delta

## ADDED Requirements

### Requirement: Link list details buttons
For the owner, outside edit mode, every link of a link list SHALL show an info button that opens the link's details (see the favourite-details spec): in the layouts "List" and "Compact" at the right end of the link's row, in the layout "Tiles" in the top-right corner of the tile. Activating it SHALL NOT follow the link.

#### Scenario: List
- **WHEN** Ann views a link list with the layout "List" and activates the info button of "Mail"
- **THEN** the details of "Mail" open, and no new tab opens

#### Scenario: Compact
- **WHEN** the layout is "Compact"
- **THEN** each row shows an info button at its right end

#### Scenario: Tiles
- **WHEN** the layout is "Tiles"
- **THEN** each tile shows an info button in its top-right corner, and activating it opens the details instead of the link
