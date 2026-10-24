# pwa Specification

## Purpose
Makes the Boardly web app installable as a progressive web app: a web app manifest, home-screen icons, a standalone window that opens the home page, and a title bar coloured like the active theme, without offline support.

## Requirements

### Requirement: Web app manifest
Every page SHALL link a web app manifest served by the app. The manifest SHALL declare the id `/`, the start URL `/`, the scope `/`, the display mode `standalone`, a name, a short name, a description, a background colour, a theme colour and the home-screen icons. The server SHALL serve manifest files with the content type `application/manifest+json`.

#### Scenario: Manifest is linked
- **WHEN** a visitor loads any page
- **THEN** the document head links a manifest

#### Scenario: Manifest is served
- **WHEN** a client requests the linked manifest
- **THEN** the server responds with status 200, content type `application/manifest+json` and a manifest whose `start_url` is `/` and `display` is `standalone`

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
`/share` SHALL require a signed-in user and show a "Save link" form with the title, the URL, tags and a folder choice, in the order of the link form (see "Link form" in the favourites spec) and with its hint about the description and preview image, prefilled from the query parameters:
- The URL is `url` if it is an absolute http or https URL. Otherwise it is the first http or https URL in `text`. Otherwise the field is empty.
- The title is `title` if present. Otherwise it is `text` without the URL taken from it, trimmed. It is cut to the link title limit.
- The folder is the one last used on this page in this browser if it still exists, otherwise the root folder.
Saving SHALL create a link in the chosen folder, with the same validation as the favourites page, and then show a confirmation with ways to open the favourites and to save another link. "Cancel" SHALL return to `/`. All text SHALL be translated (English and German).

#### Scenario: URL in the url parameter
- **WHEN** a signed-in user opens `/share?title=Docs&url=https%3A%2F%2Fexample.org%2Fdocs`
- **THEN** the form shows the URL `https://example.org/docs` and the title "Docs"

#### Scenario: URL inside the text
- **WHEN** a signed-in user opens `/share?text=Read%20this%20https%3A%2F%2Fexample.org%2Fa`
- **THEN** the form shows the URL `https://example.org/a` and the title "Read this"

#### Scenario: Nothing shared
- **WHEN** a signed-in user opens `/share` without parameters
- **THEN** the form is empty and the folder is the last used one, or the root

#### Scenario: Title before URL and hint
- **WHEN** a signed-in user opens `/share`
- **THEN** the title field comes before the URL field, and the page shows the hint that the description and preview image are loaded in the background

#### Scenario: Save
- **WHEN** the user picks the folder "Work", adds the tag "docs" and saves
- **THEN** "Work" contains the link with the tag "docs", and the page shows the confirmation

#### Scenario: Folder remembered
- **WHEN** the user saved a link into "Work" and opens `/share` again
- **THEN** "Work" is preselected

#### Scenario: Invalid URL
- **WHEN** the user tries to save with the URL `javascript:alert(1)`
- **THEN** nothing is saved and the form shows the translated validation message

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
