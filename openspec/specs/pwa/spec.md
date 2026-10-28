# pwa Specification

## Purpose
Makes the Boardly web app installable as a progressive web app: a web app manifest, home-screen icons, a standalone window that opens the home page, and a title bar coloured like the active theme, without offline support.

## Requirements

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

### Requirement: Browser offers installation
The web app SHALL meet the installability criteria of Chromium-based browsers, so that they offer to install it with their own install button or prompt. The web app SHALL NOT show an install button, banner or prompt of its own.

#### Scenario: Chromium browser
- **WHEN** a visitor opens Boardly over HTTPS in a current Chrome or Edge
- **THEN** the browser shows its install option for Boardly

#### Scenario: No own prompt
- **WHEN** a visitor opens any page in a browser that can install the app
- **THEN** the page itself shows no install button or banner

### Requirement: Installed app window
The installed app SHALL open `/` in a standalone window without browser address bar. Pages of the app that are opened inside the installed app SHALL stay in its window.

#### Scenario: Launch
- **WHEN** a signed-in user with a default dashboard launches the installed app
- **THEN** a standalone window shows their default dashboard

### Requirement: Manifest language
The manifest's name, short name and description SHALL be in the app language (English or German), determined like the app language: the stored choice first, then the browser languages, then English. Both language versions SHALL declare the same id.

#### Scenario: German browser
- **WHEN** a visitor without a stored language choice opens Boardly in a browser that prefers German
- **THEN** the linked manifest has the German description

#### Scenario: Language switched
- **WHEN** a visitor switches the app language from English to German
- **THEN** the page links the German manifest

### Requirement: Home-screen icons
The manifest SHALL list the app icon as PNG images of 192×192 and 512×512 pixels for any purpose, and a 512×512 PNG with purpose `maskable` whose drawing lies within the central 80 % circle. These icons SHALL have an opaque background. Every page SHALL also declare a 180×180 PNG Apple touch icon with an opaque background. Each icon SHALL show the same drawing as the favicon, including the rose star.

#### Scenario: Icon files exist
- **WHEN** a client requests any icon listed in the manifest or the Apple touch icon
- **THEN** the server responds with a PNG of the declared size

#### Scenario: Android adaptive icon
- **WHEN** an Android launcher crops the maskable icon to a circle
- **THEN** the whole drawing, including the star, stays visible

### Requirement: Title bar follows the theme
Every page SHALL declare a `theme-color` equal to the active theme's `paper` colour. It SHALL be correct before first paint and SHALL update without reload whenever the active theme changes.

#### Scenario: Theme switched
- **WHEN** the user switches from the White theme to the Fjord theme
- **THEN** the page's `theme-color` changes to Fjord's `paper` colour

#### Scenario: Reload with stored theme
- **WHEN** the stored theme is `ember` and the page is reloaded
- **THEN** the `theme-color` is Ember's `paper` colour before the app's JavaScript bundle runs

### Requirement: No offline support
The web app SHALL NOT register a service worker and SHALL NOT cache pages, assets or data for offline use beyond normal HTTP caching.

#### Scenario: Offline launch
- **WHEN** the installed app is launched without a network connection
- **THEN** it shows the browser's or operating system's usual offline error, as a browser tab would

### Requirement: Share target
The manifest SHALL register the app as a share target that accepts a title, a text and a URL, and opens `/share` with them as the query parameters `title`, `text` and `url`.

#### Scenario: Share from another app
- **WHEN** a user on Android, with the app installed, shares a web page from another app and picks Boardly
- **THEN** the installed app opens `/share` with the shared title, text and URL

### Requirement: Save link page
`/share` SHALL require a signed-in user and show a switch between "Link" / "Link" and "Note" / "Notiz" above the form. The switch starts on "Note" when a title or text was shared but no URL is found in the query parameters (see below), and on "Link" otherwise, so the "Add link" shortcut, which opens `/share` without parameters, starts on "Link". Switching keeps the values entered in both forms.

With "Link", the page SHALL show a "Save link" form with the title, the URL, tags and a folder choice, in the order of the link form (see "Link form" in the favourites spec) and with its hint about the description and preview image, prefilled from the query parameters:
- The URL is `url` if it is an absolute http or https URL. Otherwise it is the first http or https URL in `text`. Otherwise the field is empty.
- The title is `title` if present. Otherwise it is `text` without the URL taken from it, trimmed. It is cut to the link title limit.
- The folder is the user's default folder for new links (see "Default folder for new links" in the user-profile spec) if it is set and still exists, otherwise the root folder. Choosing another folder on the page SHALL apply only to that save and SHALL NOT change the default folder or be remembered.
Saving SHALL create a link in the chosen folder, with the same validation as the favourites page, and then show a confirmation with ways to open the favourites and to save another link.

With "Note", the page SHALL show the form described in "Save note from a share".

"Cancel" SHALL return to `/`. All text SHALL be translated (English and German).

#### Scenario: URL in the url parameter
- **WHEN** a signed-in user opens `/share?title=Docs&url=https%3A%2F%2Fexample.org%2Fdocs`
- **THEN** the switch is on "Link", and the form shows the URL `https://example.org/docs` and the title "Docs"

#### Scenario: URL inside the text
- **WHEN** a signed-in user opens `/share?text=Read%20this%20https%3A%2F%2Fexample.org%2Fa`
- **THEN** the switch is on "Link", and the form shows the URL `https://example.org/a` and the title "Read this"

#### Scenario: Nothing shared
- **WHEN** a signed-in user opens `/share` without parameters
- **THEN** the switch is on "Link", the form is empty and the folder is the default folder, or the root

#### Scenario: Title before URL and hint
- **WHEN** a signed-in user opens `/share`
- **THEN** the title field comes before the URL field, and the page shows the hint that the description and preview image are loaded in the background

#### Scenario: Save
- **WHEN** the user picks the folder "Work", adds the tag "docs" and saves a link
- **THEN** "Work" contains the link with the tag "docs", and the page shows the confirmation

#### Scenario: Default folder preselected
- **WHEN** the user's default folder is "Inbox" and they open `/share` with a URL
- **THEN** "Inbox" is preselected

#### Scenario: Folder remembered
- **WHEN** the default folder is "Inbox", the user saves a link into "Work" and opens `/share` again
- **THEN** "Inbox" is preselected again: the folder last used on this page is no longer remembered

#### Scenario: No default folder
- **WHEN** the user has no default folder set and opens `/share` with a URL
- **THEN** the root folder is preselected

#### Scenario: Invalid URL
- **WHEN** the user tries to save with the URL `javascript:alert(1)`
- **THEN** nothing is saved and the form shows the translated validation message

#### Scenario: Switch keeps values
- **WHEN** the user opens `/share` with a URL, edits the link title, switches to "Note" and back to "Link"
- **THEN** the edited link title is still there

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/share?url=https%3A%2F%2Fexample.org`
- **THEN** the browser shows `/login`, and after signing in on the login page the visitor is back on `/share?url=https%3A%2F%2Fexample.org`

### Requirement: App shortcuts
The manifest SHALL declare the app shortcuts "Add link" (`/share`), "Favourites" (`/favourites`) and "Dashboards" (`/dashboards`), in this order, with names in the manifest's language.

#### Scenario: Shortcut menu
- **WHEN** a user long-presses the installed app's icon on Android
- **THEN** the shortcuts "Add link", "Favourites" and "Dashboards" are offered

#### Scenario: German shortcuts
- **WHEN** the German manifest is linked
- **THEN** its shortcuts are named "Link hinzufügen", "Favoriten" and "Dashboards"

### Requirement: Save note from a share
With "Note" on the share page, the page SHALL show a "Save note" / "Notiz speichern" form with a title, a multi-line text field and a category choice, prefilled from the query parameters:
- The title is `title` if present, cut to the note title limit. Otherwise it is empty.
- The text is `text`. When `url` is an absolute http or https URL that isn't already contained in `text`, it is added as a last line, separated by a line break when `text` isn't empty.
- The category is the user's default category for new notes (see "Default category for new notes" in the user-profile spec) if it is set and still exists, otherwise "No category". Choosing another category SHALL apply only to that save.
Saving SHALL create a note from the title, the text as plain text (see "Content from plain text" in the note-document spec) and the chosen category, and then show a confirmation with ways to open the note and to save something else. A note can be saved with an empty text. A failed save shows a translated error and keeps the form.

#### Scenario: Shared text without a URL
- **WHEN** a signed-in user opens `/share?title=Plan&text=Meet%20at%208`
- **THEN** the switch is on "Note", the title is "Plan" and the text is "Meet at 8"

#### Scenario: URL added to the text
- **WHEN** the user opens `/share?text=Look%20at%20this&url=https%3A%2F%2Fexample.org%2Fa` and switches to "Note"
- **THEN** the text is "Look at this", a line break, and `https://example.org/a`

#### Scenario: Save a note
- **WHEN** the user saves the note "Plan" with the text "Meet at 8" in the category "Inbox"
- **THEN** "Inbox" contains the note "Plan" with a paragraph "Meet at 8", and the page shows the confirmation with a way to open the note

#### Scenario: Default category preselected
- **WHEN** the user's default category is "Inbox" and they open `/share?text=Meet%20at%208`
- **THEN** "Inbox" is preselected in the note form

#### Scenario: No default category
- **WHEN** the user has no default category, opens `/share` and switches to "Note"
- **THEN** "No category" is preselected
