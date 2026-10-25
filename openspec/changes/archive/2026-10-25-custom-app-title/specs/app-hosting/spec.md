# Spec Delta

## ADDED Requirements

### Requirement: Configurable app title
The server SHALL read the instance's app title from `BOARDLY_APP_TITLE`. Leading and trailing whitespace is removed, and an unset or empty value gives the default `Boardly`. The app title replaces the product name as a whole wherever the web app, its manifests and the sign-in email show it. The web app's HTML document SHALL carry the app title as its `<title>` and as the content of a `<meta name="application-name">` tag, escaped for HTML, on every response that serves the document, including the fallback for client-side routes.

#### Scenario: Default title
- **WHEN** the server starts without `BOARDLY_APP_TITLE` and a client requests `GET /`
- **THEN** the HTML document's title is "Boardly" and its `application-name` meta tag has the content "Boardly"

#### Scenario: Custom title
- **WHEN** the server starts with `BOARDLY_APP_TITLE="Boardly Staging"` and a client requests `GET /settings`
- **THEN** the HTML document's title is "Boardly Staging" and its `application-name` meta tag has the content "Boardly Staging"

#### Scenario: Blank value
- **WHEN** the server starts with `BOARDLY_APP_TITLE="   "`
- **THEN** the app title is "Boardly"

#### Scenario: Title with markup characters
- **WHEN** the server starts with `BOARDLY_APP_TITLE="A & B <test>"` and a client requests `GET /`
- **THEN** the HTML document's title reads "A & B <test>" as text, and the document contains no `<test>` element
