# Spec Delta

## ADDED Requirements

### Requirement: Analytics snippet
The server SHALL read an optional analytics snippet from `BOARDLY_ANALYTICS_SNIPPET` when it starts. The value is HTML, such as a `<script>` tag of an analytics tool. Leading and trailing whitespace is removed, and an unset or empty value means there is no snippet. The server SHALL insert the snippet unchanged, without escaping, directly before the closing `</head>` tag of the web app's HTML document on every response that serves the document, including the fallback for client-side routes. Without a snippet, the document SHALL be served without any addition. Changing the snippet SHALL need only a restart of the server, not a new build of the web app.

#### Scenario: No snippet by default
- **WHEN** the server starts without `BOARDLY_ANALYTICS_SNIPPET` and a client requests `GET /`
- **THEN** the HTML document contains no content besides that of the built web app before `</head>`

#### Scenario: Snippet on the start page
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET='<script defer src="https://stats.example.org/script.js" data-website-id="abc"></script>'` and a client requests `GET /`
- **THEN** the HTML document contains that script tag unchanged, directly before `</head>`

#### Scenario: Snippet on a deep link
- **WHEN** the server starts with an analytics snippet and a client requests `GET /settings`
- **THEN** the HTML document contains the snippet directly before `</head>`

#### Scenario: Snippet is not escaped
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET='<script>window.a = 1 && 2;</script>'` and a client requests `GET /`
- **THEN** the HTML document contains `<script>window.a = 1 && 2;</script>` as written, with no `&amp;` or `&lt;`

#### Scenario: Blank value
- **WHEN** the server starts with `BOARDLY_ANALYTICS_SNIPPET="   "`
- **THEN** there is no snippet, and the HTML document is served without any addition

#### Scenario: Other files are unchanged
- **WHEN** the server starts with an analytics snippet and a client requests a hashed JavaScript bundle or a web app manifest
- **THEN** the response doesn't contain the snippet
