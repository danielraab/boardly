## MODIFIED Requirements

### Requirement: Browser offers installation
The web app SHALL meet the installability criteria of Chromium-based browsers, so that they offer to install it with their own install button or prompt. The web app SHALL NOT show an install button, banner or prompt of its own.

#### Scenario: Chromium browser
- **WHEN** a visitor opens Boardly over HTTPS in a current Chrome or Edge
- **THEN** the browser shows its install option for Boardly

#### Scenario: No own prompt
- **WHEN** a visitor opens any page in a browser that can install the app
- **THEN** the page itself shows no install button or banner

### Requirement: Manifest language
The manifest's name, short name and description SHALL be in the app language (English or German), determined like the app language: the stored choice first, then the browser languages, then English. Both language versions SHALL declare the same id.

#### Scenario: German browser
- **WHEN** a visitor without a stored language choice opens Boardly in a browser that prefers German
- **THEN** the linked manifest has the German description

#### Scenario: Language switched
- **WHEN** a visitor switches the app language from English to German
- **THEN** the page links the German manifest

### Requirement: Share target
The manifest SHALL register the app as a share target that accepts a title, a text and a URL, and opens `/share` with them as the query parameters `title`, `text` and `url`.

#### Scenario: Share from another app
- **WHEN** a user on Android, with the app installed, shares a web page from another app and picks Boardly
- **THEN** the installed app opens `/share` with the shared title, text and URL
