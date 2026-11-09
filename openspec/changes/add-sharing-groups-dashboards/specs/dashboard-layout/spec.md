# Spec Delta

## MODIFIED Requirements

### Requirement: Layout in the dashboard lookup
`GET /api/u/{userSlug}/{dashboardSlug}` SHALL include the dashboard's desktop and mobile column counts, its mobile breakpoint, its layout version and its sections as a tree. Each widget has its `id`, `kind`, `width` and, in their stored order, its `children` if it is a section. A section also has its `height`, `title` and `border`. A content widget also has its `config`, and it can have read-only `data` that the server resolves for its kind. The same visibility rules as for the dashboard itself apply, so anyone but the owner gets the layout only of dashboards they can see. A widget kind can leave its widgets out of a visitor's response. The layout version and the positions of the remaining widgets don't change because of that.

#### Scenario: Visitor loads a public dashboard
- **WHEN** a visitor opens a public dashboard that has two top-level sections, the second with one child
- **THEN** the response contains both sections in order, with the child under the second one

#### Scenario: Empty dashboard
- **WHEN** a dashboard has no sections
- **THEN** the lookup returns an empty section list

#### Scenario: Data is not stored
- **WHEN** the owner saves a layout in which a content widget has a `data` value
- **THEN** the `data` value is ignored, and the stored widget only has its kind, width and config

#### Scenario: Viewer loads a shared dashboard
- **WHEN** Bob opens a dashboard Ann shared with Bob that has two top-level sections
- **THEN** the response contains both sections in order
