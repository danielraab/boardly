## MODIFIED Requirements

### Requirement: Landing page for signed-out visitors
When no user is signed in, `/` SHALL show the landing page: a hero area with the product name, a one-line translated tagline and a call to action that goes to `/login`.

#### Scenario: Visitor opens root
- **WHEN** a signed-out visitor opens `/`
- **THEN** the page shows "Boardly", the tagline and a sign-in call to action

#### Scenario: Call to action
- **WHEN** the visitor activates the call to action
- **THEN** the browser shows `/login`
