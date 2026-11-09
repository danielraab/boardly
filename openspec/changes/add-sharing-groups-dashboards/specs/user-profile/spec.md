# Spec Delta

## MODIFIED Requirements

### Requirement: Reserved user slugs
A user slug SHALL NOT be one of the reserved words: paths that the app uses or may use as its own first path segment (at least `api`, `assets`, `karakeep`, `login`, `logout`, `settings`, `favourites`, `dashboards`, `shared`, `groups`, `invitation`, `admin`, `help`, `about`). The server rejects them with 400 and `slug_reserved`. The web app checks the same list, shows a hint at the field and blocks saving. If a later version adds a fixed route that equals an existing user slug, the route takes precedence and that user has to choose a new slug.

#### Scenario: Reserved word in the form
- **WHEN** the user enters `login` as their slug
- **THEN** the field shows that this address is reserved and the save action is disabled

#### Scenario: Reserved word over the API
- **WHEN** a client sends `PATCH /api/me` with `{"slug": "API"}`
- **THEN** the server responds 400 with `slug_reserved` and the slug is unchanged

#### Scenario: Reserved page address
- **WHEN** a client sends `PATCH /api/me` with `{"slug": "groups"}`
- **THEN** the server responds 400 with `slug_reserved` and the slug is unchanged
