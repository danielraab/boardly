# Spec Delta

## MODIFIED Requirements

### Requirement: Web app manifest
Every page SHALL link a web app manifest served by the app. The manifest SHALL declare the id `/`, the start URL `/`, the scope `/`, the display mode `standalone`, a name, a short name, a description, a background colour, a theme colour and the home-screen icons. Its name and short name SHALL both be the configured app title (see "Configurable app title" in the `app-hosting` capability, "Boardly" by default) in every language. The server SHALL serve manifest files with the content type `application/manifest+json` and with `Cache-Control: no-cache`, so a changed app title reaches browsers that have the manifest cached.

#### Scenario: Manifest is linked
- **WHEN** a visitor loads any page
- **THEN** the document head links a manifest

#### Scenario: Manifest is served
- **WHEN** a client requests the linked manifest
- **THEN** the server responds with status 200, content type `application/manifest+json`, `Cache-Control: no-cache` and a manifest whose `start_url` is `/` and `display` is `standalone`

#### Scenario: Default name
- **WHEN** the server runs without `BOARDLY_APP_TITLE` and a client requests `/manifest.en.webmanifest`
- **THEN** the manifest's `name` and `short_name` are "Boardly"

#### Scenario: Custom name
- **WHEN** the server runs with `BOARDLY_APP_TITLE="Boardly Staging"` and a client requests `/manifest.de.webmanifest`
- **THEN** the manifest's `name` and `short_name` are "Boardly Staging" and all other fields are unchanged
