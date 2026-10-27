# Spec Delta

## ADDED Requirements

### Requirement: Collapsible sections
Outside edit mode, a section with a title SHALL have a toggle button in its heading that collapses or expands it. Sections start expanded. A collapsed section only shows its heading: its children are hidden, it keeps its width, and it is as tall as its heading, also when it has a fixed height. The toggle's accessible name says whether it expands or collapses the section and names the section, and the button reports its state with `aria-expanded`. A section without a title has no toggle and is always expanded. Which sections are collapsed is remembered per browser in local storage and applies again after a reload. It is not sent to the server and doesn't change the layout or its version. When local storage is unavailable, collapsing still works until the page is left. In edit mode every section is shown expanded and has no toggle; the remembered state is kept and applies again after leaving edit mode.

#### Scenario: Collapse a section
- **WHEN** a viewer clicks the toggle of the expanded section "News"
- **THEN** the section's children are hidden, its heading stays visible, and the toggle has `aria-expanded="false"`

#### Scenario: Expand a section
- **WHEN** a viewer clicks the toggle of the collapsed section "News"
- **THEN** the section's children are shown again

#### Scenario: Remembered after reload
- **WHEN** a viewer collapses the section "News" and reloads the page
- **THEN** "News" is still collapsed and the other sections are expanded

#### Scenario: Fixed height
- **WHEN** a section with a height of 3 rows is collapsed
- **THEN** it is only as tall as its heading

#### Scenario: Visitor of a public dashboard
- **WHEN** a visitor who isn't signed in collapses a section on a public dashboard
- **THEN** the section collapses and the dashboard's stored layout is unchanged

#### Scenario: Untitled section
- **WHEN** a section has no title and the dashboard is not in edit mode
- **THEN** it has no toggle and its children are shown

#### Scenario: Edit mode
- **WHEN** the owner collapses the section "News" and then opens edit mode
- **THEN** "News" is shown expanded without a toggle, and after leaving edit mode it is collapsed again
