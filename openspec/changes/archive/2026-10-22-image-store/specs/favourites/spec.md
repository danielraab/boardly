# Spec Delta

## MODIFIED Requirements

### Requirement: Link nodes
A link SHALL have an absolute `http` or `https` URL of at most 2048 characters and an optional title of at most 1000 characters after trimming. An empty title SHALL be stored as no title. The tree SHALL show a link by its title, or by its URL when it has no title. Activating a link in the tree SHALL open its URL in a new tab. A link SHALL show the favicon of its URL's origin when one is known, and a globe icon otherwise. Creating a link or changing its URL SHALL start fetching that origin's favicon in the background, without delaying the response. In the API, a link node has `iconImageId` set to its favicon's image ID when one is known, and leaves it out otherwise.

#### Scenario: Create a link with a title
- **WHEN** the user adds a link with URL `https://example.org/docs` and title "Docs" to the folder "Work"
- **THEN** "Work" shows a link labelled "Docs"

#### Scenario: Link without title
- **WHEN** the user adds a link with URL `https://example.org` and no title
- **THEN** the tree shows the link labelled `https://example.org`

#### Scenario: Invalid URL
- **WHEN** the user tries to save a link with the URL `example` or `javascript:alert(1)`
- **THEN** the link isn't saved and the form shows a translated validation message

#### Scenario: Open a link
- **WHEN** the user activates the link "Docs" in the tree
- **THEN** `https://example.org/docs` opens in a new tab

#### Scenario: Edit a link
- **WHEN** the user changes the title of "Docs" to "Documentation" and its URL to `https://example.org/doc`
- **THEN** the tree shows "Documentation", which opens the new URL

#### Scenario: Favicon appears
- **WHEN** the user adds a link to `https://github.com/` and the favicon fetch succeeds
- **THEN** the link shows GitHub's favicon once the tree is loaded again

#### Scenario: No favicon
- **WHEN** a link's origin has no favicon
- **THEN** the link shows a globe icon
