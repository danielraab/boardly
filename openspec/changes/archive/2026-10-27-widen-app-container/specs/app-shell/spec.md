# Spec Delta

## ADDED Requirements

### Requirement: Content width
The header and the page content SHALL share one horizontally centred container that is as wide as the viewport minus the side padding, up to at most 1536 px. On wider viewports the container stays 1536 px wide and the remaining space is split evenly to its left and right. The left edge of the header content and the left edge of the page content line up at every viewport width. Pages may limit their own content to a narrower width inside the container.

#### Scenario: Wide viewport
- **WHEN** a page is viewed 1920 px wide
- **THEN** the header and the page content are each 1536 px wide and centred, with 192 px of empty space on each side

#### Scenario: Viewport narrower than the cap
- **WHEN** a page is viewed 1280 px wide
- **THEN** the header and the page content span the whole viewport except for the side padding

#### Scenario: Header and content aligned
- **WHEN** a dashboard is viewed at any width
- **THEN** the app icon in the header and the dashboard title start at the same horizontal position
