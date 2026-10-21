# Spec Delta

## Purpose

What a dashboard contains and how it renders: sections as the structural widget, the column grid with its desktop and mobile settings, and the layout data delivered with the dashboard.

## ADDED Requirements

### Requirement: Sections at the top level
A dashboard's content SHALL be an ordered list of widgets. At the top level, only sections are allowed. In this version, sections are also the only widget type allowed inside a section. A dashboard without any sections is empty.

#### Scenario: Top-level widget that isn't a section
- **WHEN** a layout is saved whose top level contains a widget of a kind other than `section`
- **THEN** the server responds 400 with `invalid_layout` and the stored layout is unchanged

#### Scenario: New dashboard
- **WHEN** a user creates a dashboard
- **THEN** it has no sections

### Requirement: Section settings
A section SHALL have a width in columns (an integer from 1 to 12), an optional title (trimmed; at most 100 characters; empty means no title), a border flag (default on) and a height. The height is either `auto` (default), which fits the content, or a whole number of rows from 1 to 50. Values outside these ranges are rejected with 400 and `invalid_layout`.

#### Scenario: Defaults
- **WHEN** a section is stored without a title and with border on and height `auto`
- **THEN** it renders with a border, no heading, and a height that fits its content

#### Scenario: Title too long
- **WHEN** a layout is saved with a section title of 101 characters
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Width out of range
- **WHEN** a layout is saved with a section width of 0 or 13
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Section rendering
A section SHALL render its title as a heading when it has one, and a visible border only when its border flag is on. With a fixed height of N rows, a section is N row units tall. One row unit is `4rem`, and the gaps between rows are included. Content that is taller than a fixed height scrolls vertically inside the section.

#### Scenario: Borderless section
- **WHEN** a section has the border turned off and the dashboard is not in edit mode
- **THEN** no border is drawn around it

#### Scenario: Fixed height with overflow
- **WHEN** a section has a height of 3 rows and its content is taller than that
- **THEN** the section is 3 rows tall and shows a vertical scrollbar

### Requirement: Nesting limit
Sections SHALL nest at most 5 levels deep. A top-level section is level 1. A layout with a section at level 6 or deeper is rejected with 400 and `invalid_layout`.

#### Scenario: Five levels
- **WHEN** a layout is saved with sections nested exactly 5 levels deep
- **THEN** it is saved

#### Scenario: Six levels
- **WHEN** a layout is saved with a section at level 6
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Column grid
Each dashboard SHALL lay out its top-level sections on a grid. The column count is the dashboard's desktop column count when the viewport is at least as wide as its mobile breakpoint, and the mobile column count otherwise. Sections are placed in order, left to right, and wrap to a new line when the next one doesn't fit.

#### Scenario: Wrapping
- **WHEN** a dashboard with 6 desktop columns has three top-level sections of width 3 on a desktop-width viewport
- **THEN** the first two share the first line and the third starts the second line

#### Scenario: Mobile count
- **WHEN** a dashboard with 6 desktop columns, 2 mobile columns and a 768 px breakpoint is viewed 600 px wide
- **THEN** the grid has 2 columns

### Requirement: Widths count against the page
A section's width SHALL count in page columns at every nesting level. A section's rendered width is the smaller of its own width and the number of columns its parent has: the grid's column count for a top-level section, or the parent section's rendered width for a nested one. A section lays out its children on a grid with as many columns as its own rendered width. The stored width is never changed by this.

#### Scenario: Nested widths
- **WHEN** a width-4 section on a 6-column grid contains two width-2 sections
- **THEN** the inner sections sit side by side, and each is as wide as 2 page columns

#### Scenario: Wider than the parent
- **WHEN** a width-6 section is inside a width-3 section
- **THEN** it renders 3 columns wide, and its stored width stays 6

#### Scenario: Mobile clamp
- **WHEN** a width-4 section is viewed on a grid with 2 mobile columns
- **THEN** it renders 2 columns wide (full width)

### Requirement: Layout in the dashboard lookup
`GET /api/u/{userSlug}/{dashboardSlug}` SHALL include the dashboard's desktop and mobile column counts, its mobile breakpoint, its layout version and its sections as a tree. Each section has its `id`, `width`, `title`, `border`, `height` and `children`, in their stored order. The same visibility rules as for the dashboard itself apply, so visitors get the layout of public dashboards only.

#### Scenario: Visitor loads a public dashboard
- **WHEN** a visitor opens a public dashboard that has two top-level sections, the second with one child
- **THEN** the response contains both sections in order, with the child under the second one

#### Scenario: Empty dashboard
- **WHEN** a dashboard has no sections
- **THEN** the lookup returns an empty section list
