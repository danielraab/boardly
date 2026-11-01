# media-library Specification

## Purpose
Gives a signed-in user a library of media sources on the `/media` page, starting with internet radio stations that they find through radio-browser.info or paste by hand, and that the media player and the media widget play.

## Requirements

### Requirement: Media page
`/media` SHALL require a signed-in user and show the heading "Media" / "Medien", an "Add station" / "Sender hinzufügen" button and the user's media sources in their order. Signed-out visitors SHALL be sent to `/login`. When the user has no media sources, the page SHALL show "No stations yet" / "Noch keine Sender" with a hint to add one. The document title is "Media - <app title>".

#### Scenario: Open the page
- **WHEN** Ann has the stations "FM4" and "Jazz Radio" in that order and opens `/media`
- **THEN** the page shows "Media", the "Add station" button, and "FM4" followed by "Jazz Radio"

#### Scenario: Empty library
- **WHEN** Ann has no media sources and opens `/media`
- **THEN** the page shows "No stations yet"

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/media`
- **THEN** the browser shows `/login`

### Requirement: Media sources
Each media source SHALL belong to exactly one user and have an ID, a kind, a title (trimmed, 1–200 characters), a URL, an optional icon (a symbol or an image, never both) and a position in the user's list. The only kind in this version is `stream`, an internet radio station whose URL is an `https` URL of an audio stream. The server SHALL reject any other kind with 400 `invalid_kind`, and a title outside 1–200 characters with 400 `invalid_title`. A user SHALL NOT see or change another user's media sources. Requests for another user's source, or for an unknown one, get 404 `not_found`. Deleting a user deletes their media sources.

#### Scenario: Another user's station
- **WHEN** Bob requests `PATCH /api/media/<ID of Ann's station>`
- **THEN** the server responds 404 with `not_found` and the station is unchanged

#### Scenario: Unknown kind
- **WHEN** Ann creates a media source with kind `embed`
- **THEN** the server responds 400 with `invalid_kind`

#### Scenario: Empty title
- **WHEN** Ann renames a station to "   "
- **THEN** the server responds 400 with `invalid_title`

### Requirement: Media API
The web app SHALL manage media sources through these endpoints, all of which require a session and respond 401 without one:
- `GET /api/media` lists the user's sources in their order, each with `id`, `kind`, `title`, `url`, `icon` (as for notes: `{ "type": "symbol", "name": … }`, `{ "type": "image", "imageId": … }` or `null`) and `position`.
- `POST /api/media` creates a source from `kind`, `title`, `url` and optionally `icon` or `logoUrl`, adds it at the end of the list after checking its URL (see "Stream URL check"), and responds 201 with the source.
- `PATCH /api/media/{id}` changes `title`, `url` (checked again only when it changed) or `icon` (`null` removes it), and responds with the source.
- `PUT /api/media/order` takes the IDs of all the user's sources in their new order. A list that isn't exactly the user's source IDs gets 400 `invalid_order`.
- `DELETE /api/media/{id}` deletes the source and responds 204.

#### Scenario: Create
- **WHEN** Ann posts a stream "FM4" with the URL `https://orf-live.ors-shoutcast.at/fm4-q2a`
- **THEN** the server responds 201 with the new station, and it is last in `GET /api/media`

#### Scenario: Reorder
- **WHEN** Ann has "FM4" and "Jazz Radio" and puts `[<Jazz Radio>, <FM4>]` to `/api/media/order`
- **THEN** `GET /api/media` lists "Jazz Radio" first

#### Scenario: Incomplete order
- **WHEN** Ann puts a list to `/api/media/order` that leaves out one of her stations
- **THEN** the server responds 400 with `invalid_order` and the order is unchanged

#### Scenario: No session
- **WHEN** a request to `GET /api/media` has no session
- **THEN** the server responds 401

### Requirement: Stream URL check
Before it saves a stream URL, the server SHALL check it as follows:
- Only `http` and `https` URLs are accepted. Any other scheme, or a malformed URL, gets 400 `invalid_url`.
- An `http` URL is tried once as `https` with the same host, port and path. If that works, the `https` URL is saved. Otherwise the server responds 400 `stream_not_https`.
- When the response is a playlist (`.pls`, `.m3u` or `.m3u8` by content type or extension, and not an HLS media playlist), the first `https` stream URL in it is used, following the same rules. When it has no usable entry, the server responds 400 `stream_unreachable`.
- The final URL must answer with an audio content type or an HLS playlist within 10 seconds. Otherwise the server responds 400 `stream_unreachable`. The server reads only the response headers and at most 64 KB of a playlist, and never relays audio.
- At most 5 redirects are followed, and a redirect to an `http` URL counts as `stream_not_https`.

The check SHALL apply the private address protection of the `images` capability. The resolved URL is the one that is saved and returned.

#### Scenario: HTTPS stream
- **WHEN** Ann adds `https://stream.example.org/live.mp3`, which answers with `audio/mpeg`
- **THEN** the station is saved with that URL

#### Scenario: HTTP with an HTTPS twin
- **WHEN** Ann pastes `http://stream.example.org/live.mp3` and `https://stream.example.org/live.mp3` answers with `audio/mpeg`
- **THEN** the station is saved with the `https` URL

#### Scenario: HTTP only
- **WHEN** Ann pastes `http://old.example.org:8000/stream` and the `https` version can't be reached
- **THEN** the server responds 400 with `stream_not_https` and nothing is saved

#### Scenario: Playlist file
- **WHEN** Ann pastes `https://radio.example.org/listen.pls`, which lists `https://edge.example.org/radio.aac`
- **THEN** the station is saved with `https://edge.example.org/radio.aac`

#### Scenario: Not audio
- **WHEN** Ann pastes `https://example.org/`, which answers with `text/html`
- **THEN** the server responds 400 with `stream_unreachable`

#### Scenario: Private address
- **WHEN** private addresses are not allowed and Ann pastes a URL whose host resolves to `192.168.1.20`
- **THEN** the server responds 400 with `stream_unreachable` and makes no connection to that address

### Requirement: Station logo
When a stream is created with a `logoUrl` and without an `icon`, the server SHALL fetch that image once with the rules of "Fetching favicons" in the `images` capability (types, size, time limit, private address protection) and store it as an image of the user, which becomes the station's icon. An `http` logo URL is allowed, because the server, not the browser, loads it. When the logo can't be fetched or isn't an accepted image, the station SHALL be saved without an icon. The response never fails because of the logo.

#### Scenario: Logo from search
- **WHEN** Ann adds "FM4" from the search and radio-browser.info gives a PNG logo
- **THEN** the station's icon is that PNG, served from `/api/images/<id>`

#### Scenario: Broken logo
- **WHEN** the logo URL answers 404
- **THEN** the station is saved without an icon

### Requirement: Station search
The "Add station" dialog SHALL have the tabs "Search" / "Suchen" (default) and "Enter URL" / "URL eingeben". The search tab has a search field. After the user stops typing for a moment, at least 2 characters are sent **from the browser directly** to the public radio-browser.info API, without going through Boardly's server. It shows at most 30 results, ordered by popularity. Each result shows the station's logo (or a radio icon), name, country, tags and bitrate, if known. A station without an `https` stream URL SHALL be shown dimmed with "Not playable (no HTTPS stream)" / "Nicht abspielbar (kein HTTPS-Stream)" and can't be added. Choosing a playable result SHALL create the station with its name as title, its stream URL and its logo URL. A result whose station is already in the library, by URL, SHALL show "Added" / "Hinzugefügt" and can't be added again. While searching, the dialog shows a loading state. When the directory can't be reached, it shows "The station directory isn't reachable right now. You can still enter a URL." / "Das Senderverzeichnis ist gerade nicht erreichbar. Du kannst trotzdem eine URL eingeben." Nothing is sent to radio-browser.info until the user types in the search field. Below the `sm` breakpoint, the add button of a result SHALL show only its icon; its accessible name still says "Add <station>" / "<station> hinzufügen".

#### Scenario: Search and add
- **WHEN** Ann types "fm4" and chooses the result "FM4" with an `https` stream
- **THEN** "FM4" is added at the end of her library with its stream URL and logo

#### Scenario: Station without HTTPS
- **WHEN** a result's only stream URL starts with `http://`
- **THEN** it is dimmed, shows "Not playable (no HTTPS stream)", and can't be chosen

#### Scenario: Already added
- **WHEN** Ann already has "FM4" with the same stream URL and searches "fm4"
- **THEN** the result "FM4" shows "Added" and can't be chosen

#### Scenario: Directory down
- **WHEN** radio-browser.info doesn't answer
- **THEN** the dialog shows that the directory isn't reachable and the "Enter URL" tab still works

#### Scenario: No request before typing
- **WHEN** Ann opens the dialog and doesn't type
- **THEN** the browser sends no request to radio-browser.info

#### Scenario: Add button on a phone
- **WHEN** Ann searches "fm4" on a screen narrower than the `sm` breakpoint
- **THEN** the add button of "FM4" shows only a plus icon and is announced as "Add FM4"

### Requirement: External directory notice
The search tab SHALL always show a notice that the search is sent to radio-browser.info, a free, community-run station directory that isn't part of Boardly: "Search by radio-browser.info, a free community directory. Your search terms are sent to it." / "Suche über radio-browser.info, ein freies Community-Verzeichnis. Deine Suchbegriffe werden dorthin gesendet." The name SHALL link to `https://www.radio-browser.info` in a new tab, without giving the new page access to the opener, and with an accessible name that says it opens in a new tab.

#### Scenario: Notice shown
- **WHEN** Ann opens the "Search" tab of the "Add station" dialog
- **THEN** the notice about radio-browser.info is visible above the results, with a link to `https://www.radio-browser.info`

### Requirement: Add a station by URL
The "Enter URL" tab SHALL have a "Name" / "Name" field and a "Stream URL" / "Stream-URL" field, both required, and a hint that only HTTPS streams can be played. Confirming creates the station. While the server checks the URL, the button shows a busy state. A failed check SHALL be shown at the URL field with a translated message for `invalid_url`, `stream_not_https` and `stream_unreachable`, and the dialog stays open with the entered values.

#### Scenario: Paste a URL
- **WHEN** Ann enters "My station" and `https://stream.example.org/live.mp3` and confirms
- **THEN** "My station" is added at the end of her library and the dialog closes

#### Scenario: Not HTTPS
- **WHEN** the server responds `stream_not_https`
- **THEN** the URL field says that this station only offers an unencrypted stream, which Boardly can't play, and the dialog stays open

### Requirement: Manage stations
Each station on `/media` SHALL show its icon (or a radio icon), its title and its URL's host, a play button that plays it in the media player (see the `media-player` capability) and shows a pause button while that station is playing, and a menu with "Edit" / "Bearbeiten" and "Delete" / "Löschen". "Edit" opens a dialog with the title, the stream URL and the icon, chosen with the same icon chooser as notes (image or symbol, or none). Deleting asks for confirmation. Stations SHALL be reorderable by dragging a handle, also with the keyboard as on the favourites page. Deleting the station that is playing stops the player.

#### Scenario: Play from the page
- **WHEN** Ann activates the play button of "FM4"
- **THEN** the media player plays "FM4" and the button becomes a pause button

#### Scenario: Rename
- **WHEN** Ann edits "FM4", changes the title to "FM4 Wien" and confirms
- **THEN** the list shows "FM4 Wien"

#### Scenario: Delete the playing station
- **WHEN** "FM4" is playing and Ann deletes it and confirms
- **THEN** playback stops, the player closes and "FM4" is gone from the list

#### Scenario: Reorder with the keyboard
- **WHEN** Ann focuses the handle of "Jazz Radio", presses Space, ArrowUp and Space
- **THEN** "Jazz Radio" is above "FM4", and the order is saved
