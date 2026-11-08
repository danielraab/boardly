# Spec Delta

## ADDED Requirements

### Requirement: Existing widget heights
When the server upgrades to a version without widget heights, a section with height `auto` whose only child is a content widget with a fixed height SHALL take over that widget's height. All other stored widget heights SHALL be dropped. The upgrade SHALL NOT add, move or remove any widget or section, and SHALL NOT change a widget's width.

#### Scenario: Only child with a fixed height
- **WHEN** a dashboard has an `auto`-height section whose only child is a content widget with a height of 6 rows, and the server is upgraded
- **THEN** the section has a height of 6 rows and the widget has no height

#### Scenario: Section already fixed
- **WHEN** a section with a height of 4 rows has one content widget with a height of 6 rows, and the server is upgraded
- **THEN** the section keeps its height of 4 rows and the widget has no height

#### Scenario: Several children
- **WHEN** an `auto`-height section has two content widgets with fixed heights, and the server is upgraded
- **THEN** the section stays `auto` and neither widget has a height

## MODIFIED Requirements

### Requirement: Section rendering
A section SHALL render its title as a heading when it has one, and a visible border only when its border flag is on. With a fixed height of N rows, a section is N row units tall. One row unit is `4rem`, and the gaps between rows are included. Content that is taller than a fixed height scrolls vertically inside the section, and the section is the only part of its content that scrolls vertically. A content widget that is the section's only child fills the height that is left below the heading of a fixed-height section. With several children, each child keeps its natural height.

#### Scenario: Borderless section
- **WHEN** a section has the border turned off and the dashboard is not in edit mode
- **THEN** no border is drawn around it

#### Scenario: Fixed height with overflow
- **WHEN** a section has a height of 3 rows and its content is taller than that
- **THEN** the section is 3 rows tall and shows a vertical scrollbar

#### Scenario: Only child fills the section
- **WHEN** a section with a height of 6 rows has one content widget as its only child, and the widget's content is shorter than the section
- **THEN** the widget is as tall as the section's area below its heading

#### Scenario: Several children overflow
- **WHEN** a section with a height of 4 rows has two content widgets that together need more than 4 rows
- **THEN** each widget has its natural height, the section stays 4 rows tall, and the section shows a vertical scrollbar

#### Scenario: Only child is a section
- **WHEN** a section with a height of 6 rows has one section as its only child
- **THEN** the inner section keeps its own height

### Requirement: Layout in the dashboard lookup
`GET /api/u/{userSlug}/{dashboardSlug}` SHALL include the dashboard's desktop and mobile column counts, its mobile breakpoint, its layout version and its sections as a tree. Each widget has its `id`, `kind`, `width` and, in their stored order, its `children` if it is a section. A section also has its `height`, `title` and `border`. A content widget also has its `config`, and it can have read-only `data` that the server resolves for its kind. The same visibility rules as for the dashboard itself apply, so visitors get the layout of public dashboards only. A widget kind can leave its widgets out of a visitor's response. The layout version and the positions of the remaining widgets don't change because of that.

#### Scenario: Visitor loads a public dashboard
- **WHEN** a visitor opens a public dashboard that has two top-level sections, the second with one child
- **THEN** the response contains both sections in order, with the child under the second one

#### Scenario: Empty dashboard
- **WHEN** a dashboard has no sections
- **THEN** the lookup returns an empty section list

#### Scenario: Data is not stored
- **WHEN** the owner saves a layout in which a content widget has a `data` value
- **THEN** the `data` value is ignored, and the stored widget only has its kind, width and config

### Requirement: Content widget settings
A content widget SHALL have a width in columns (an integer from 1 to 12) and a config whose fields are defined by its kind. The server checks the config with the rules of the kind. Values outside the common ranges or a config that the kind doesn't accept are rejected with 400 and `invalid_layout`. Content widgets count towards the limit of 500 widgets per dashboard. A content widget has no height: a `height` of `auto` or a positive number sent for one is ignored, and it isn't stored or returned.

#### Scenario: Invalid config
- **WHEN** a layout is saved with a content widget whose config the kind doesn't accept
- **THEN** the server responds 400 with `invalid_layout` and the stored layout is unchanged

#### Scenario: Height out of range
- **WHEN** a layout is saved with a content widget of height 51
- **THEN** the layout is saved, and the widget is stored and returned without a height

#### Scenario: Height sent
- **WHEN** a layout is saved with a content widget of height 3
- **THEN** the layout is saved, and the widget is stored and returned without a height

### Requirement: Content widget rendering
A content widget SHALL be placed on its parent section's grid like a section, with the same width rule: its rendered width is the smaller of its own width and its parent's rendered width. It fills the whole grid cell. It is as tall as its content, or fills a fixed-height section as described under section rendering. Content that is too wide scrolls horizontally inside the widget.

#### Scenario: Next to a section
- **WHEN** a width-4 section contains a width-2 section and a width-2 content widget
- **THEN** both sit side by side, each 2 page columns wide

#### Scenario: Overflow
- **WHEN** a content widget's content is wider than the widget
- **THEN** the widget keeps its width and shows a horizontal scrollbar

#### Scenario: Follows its section
- **WHEN** a content widget of width 12 is inside a width-4 section and the section is changed to width 6
- **THEN** the widget renders 4 columns wide before the change and 6 columns wide after it, and its stored width stays 12
