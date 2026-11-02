# Spec Delta

## MODIFIED Requirements

### Requirement: Media API
The web app SHALL manage media sources through these endpoints, all of which require a session and respond 401 without one:
- `GET /api/media` lists the user's sources in their order, each with `id`, `kind`, `title`, `url`, `icon` (as for notes: `{ "type": "symbol", "name": … }`, `{ "type": "image", "imageId": … }` or `null`) and `position`.
- `POST /api/media` creates a source from `kind`, `title`, `url` and optionally `icon` or `logoUrl`, adds it at the end of the list after checking its URL (see "Stream URL check"), and responds 201 with the source.
- `PATCH /api/media/{id}` changes `title`, `url` (checked again only when it changed) or `icon` (`null` removes it), and responds with the source.
- `PUT /api/media/order` takes the IDs of all the user's sources in their new order. A list that isn't exactly the user's source IDs gets 400 `invalid_order`.
- `DELETE /api/media/{id}` deletes the source and responds 204.
- `GET /api/media/search?q=<terms>` searches radio-browser.info by station name on the server and responds with at most 30 results, most popular first. Each result has `id`, `name`, `url` (the stream URL, resolved by the directory where it could), `logoUrl` (the directory's logo URL or `null`), `logo` (a link to the logo through Boardly, see below, or `null`), `country`, `tags` (at most 3), `bitrate` (kbit/s or `null`) and `playable` (whether `url` is `https`). Terms that are shorter than 2 or longer than 100 characters after trimming get 400 `invalid_query`. When the directory can't be reached or answers with an error, the server responds 502 `directory_unreachable`. Searches are limited per user, and too many get 429 `rate_limited` with `Retry-After`. The server sends radio-browser.info only the terms and the search options, never the user's IP address, session or any other data about the user.
- `GET /api/media/search/logo?…` serves a result's logo. The link is only valid for the user it was made for and only for that logo URL, and an invalid link gets 404. The server fetches the logo over guarded connections (no private or internal addresses, unless `BOARDLY_FETCH_ALLOW_PRIVATE` is set), with the time and size limits of favicons, and serves it only when it is a PNG, JPEG, GIF, WebP or ICO image, with `X-Content-Type-Options: nosniff`, `Content-Security-Policy: sandbox` and a private cache lifetime. Anything else, including an unreachable host, gets 404.

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
- **WHEN** a request to `GET /api/media` or `GET /api/media/search?q=fm4` has no session
- **THEN** the server responds 401 and nothing is sent to radio-browser.info

#### Scenario: Search through the server
- **WHEN** Ann requests `GET /api/media/search?q=fm4`
- **THEN** the server asks radio-browser.info for "fm4" and responds with the results, each with a `logo` link to `/api/media/search/logo` when the directory has a logo

#### Scenario: Query too short
- **WHEN** Ann requests `GET /api/media/search?q=f`
- **THEN** the server responds 400 with `invalid_query` and nothing is sent to radio-browser.info

#### Scenario: Directory unreachable
- **WHEN** radio-browser.info doesn't answer and Ann searches "fm4"
- **THEN** the server responds 502 with `directory_unreachable`

#### Scenario: Logo through the server
- **WHEN** Ann's browser requests the `logo` link of the result "FM4", whose logo is a PNG
- **THEN** the server responds with the PNG, `Content-Security-Policy: sandbox` and `X-Content-Type-Options: nosniff`

#### Scenario: Logo link of someone else
- **WHEN** Bob requests the `logo` link that was made for Ann, or a link whose logo URL was changed
- **THEN** the server responds 404 and fetches nothing

#### Scenario: Logo that isn't an image
- **WHEN** a result's logo URL returns HTML or an SVG
- **THEN** the `logo` link responds 404

### Requirement: Station search
The "Add station" dialog SHALL have the tabs "Search" / "Suchen" (default) and "Enter URL" / "URL eingeben". The search tab has a search field. After the user stops typing for a moment, at least 2 characters are sent to Boardly's server (`GET /api/media/search`), which searches the public radio-browser.info directory. While searching, the browser SHALL contact neither radio-browser.info nor the hosts of station logos: logos are loaded through Boardly's server. It shows at most 30 results, ordered by popularity. Each result shows the station's logo (or a radio icon, also when the logo can't be loaded), name, country, tags and bitrate, if known. A station without an `https` stream URL SHALL be shown dimmed with "Not playable (no HTTPS stream)" / "Nicht abspielbar (kein HTTPS-Stream)" and can't be added. Choosing a playable result SHALL create the station with its name as title, its stream URL and its logo URL. A result whose station is already in the library, by URL, SHALL show "Added" / "Hinzugefügt" and can't be added again. While searching, the dialog shows a loading state. When the directory can't be reached, it shows "The station directory isn't reachable right now. You can still enter a URL." / "Das Senderverzeichnis ist gerade nicht erreichbar. Du kannst trotzdem eine URL eingeben." When searches are limited, it shows "Too many searches. Try again in a moment." / "Zu viele Suchen. Versuch es gleich noch einmal." Nothing is sent until the user types in the search field. Below the `sm` breakpoint, the add button of a result SHALL show only its icon; its accessible name still says "Add <station>" / "<station> hinzufügen".

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
- **THEN** the browser sends no search request

#### Scenario: No third party from the browser
- **WHEN** Ann searches "fm4" and the results with logos are shown
- **THEN** every request the browser made went to Boardly's own origin

#### Scenario: Logo can't be loaded
- **WHEN** the logo link of a result responds 404
- **THEN** the result shows the radio icon

#### Scenario: Add button on a phone
- **WHEN** Ann searches "fm4" on a screen narrower than the `sm` breakpoint
- **THEN** the add button of "FM4" shows only a plus icon and is announced as "Add FM4"

### Requirement: External directory notice
The search tab SHALL always show a notice that the search uses radio-browser.info, a free, community-run station directory that isn't part of Boardly, and that the search terms reach it through this server without the user's IP address: "Search by radio-browser.info, a free community directory. Your search terms are sent to it through this server, without your IP address." / "Suche über radio-browser.info, ein freies Community-Verzeichnis. Deine Suchbegriffe werden über diesen Server dorthin gesendet, ohne deine IP-Adresse." The name SHALL link to `https://www.radio-browser.info` in a new tab, without giving the new page access to the opener, and with an accessible name that says it opens in a new tab.

#### Scenario: Notice shown
- **WHEN** Ann opens the "Search" tab of the "Add station" dialog
- **THEN** the notice about radio-browser.info, with the note that her IP address isn't sent, is visible above the results, with a link to `https://www.radio-browser.info`
