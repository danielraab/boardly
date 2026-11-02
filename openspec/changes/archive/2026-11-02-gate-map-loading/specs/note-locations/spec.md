# Spec Delta

## ADDED Requirements

### Requirement: Map consent
When the map style is on another host than the page, no map SHALL be loaded until the viewer
agrees. This applies to the map modal and the location form, for signed-in users and for visitors
of public dashboards alike. Until then, the place of the map SHALL show a placeholder that says
"The map is loaded from <host>, which sees your IP address and the area shown." / "Die Karte wird
von <host> geladen, das deine IP-Adresse und den gezeigten Ausschnitt sieht.", where `<host>` is
the host name of the map style URL. Below the text are a "Load map" / "Karte laden" button and a
checkbox "Always load maps in this browser" / "Karten in diesem Browser immer laden", which is
unchecked at first. While the placeholder is shown, the browser SHALL request nothing from the map
provider and SHALL load no map code. The modal's label and coordinates, and the location form's
fields, Save and "Use my location", SHALL work without loading the map.

"Load map" without the checkbox SHALL load the map and allow maps from that host until the page is
reloaded, so later maps on the same page load without asking. With the checkbox, it SHALL also
store the consent for that host in the browser, so maps from that host load without asking after a
reload too. Consent is kept per host: consent for one host SHALL NOT allow maps from another. When
the browser can't store the consent (for example, storage is blocked), "Load map" SHALL still load
the map for the page.

A map that loaded because of stored consent SHALL show a "Don't load maps automatically" / "Karten
nicht automatisch laden" link below it. Activating the link SHALL remove the stored consent for
that host. The map shown stays, the link is replaced by "Maps will ask again." / "Karten werden
wieder nachfragen.", and the next map after a reload shows the placeholder again.

When the host name of the map style URL is the host name of the page, maps SHALL load without the
placeholder and without the link.

#### Scenario: Placeholder first
- **WHEN** a visitor of Ann's public dashboard opens the map modal of "Stephansplatz" for the first time, with the default map style
- **THEN** the modal shows "Stephansplatz", the coordinates and the placeholder naming `tiles.openfreemap.org`, and no request is made to `tiles.openfreemap.org`

#### Scenario: Load once
- **WHEN** Ann activates "Load map" without the checkbox, closes the modal and opens the map modal of another location
- **THEN** both maps load, and after a reload the placeholder is shown again

#### Scenario: Always load
- **WHEN** Ann checks "Always load maps in this browser", activates "Load map" and reloads the page
- **THEN** the next map modal shows the map without the placeholder, with the link "Don't load maps automatically" below it

#### Scenario: Withdraw
- **WHEN** Ann activates "Don't load maps automatically" below a map
- **THEN** the map stays, the link is replaced by "Maps will ask again." and after a reload the next map shows the placeholder

#### Scenario: Other provider
- **WHEN** Ann stored consent for `tiles.openfreemap.org` and the operator changes `BOARDLY_MAP_STYLE_URL` to `https://maps.example.org/style.json`
- **THEN** the next map shows the placeholder naming `maps.example.org`

#### Scenario: Form without the map
- **WHEN** Ann opens "Add location" without agreeing, enters 48.208176 and 16.373819 and saves
- **THEN** the location is added and nothing is requested from the map provider

#### Scenario: Self-hosted style
- **WHEN** Boardly runs at `https://boardly.example.org` with `BOARDLY_MAP_STYLE_URL=https://boardly.example.org/maps/style.json`
- **THEN** maps load without the placeholder and without the link

#### Scenario: Storage blocked
- **WHEN** the browser blocks storage and Ann checks the checkbox and activates "Load map"
- **THEN** the map loads, and after a reload the placeholder is shown again

## MODIFIED Requirements

### Requirement: Map configuration
Maps SHALL be vector maps drawn from the MapLibre style at `BOARDLY_MAP_STYLE_URL`, by default
`https://tiles.openfreemap.org/styles/liberty`. The map code and the style SHALL only be loaded when
a map is first shown (the map modal or the location form) and, unless the style is on the page's
host, the viewer has agreed (see "Map consent"), so a page that only shows location chips, or a
map the viewer hasn't agreed to, makes no request to the map provider. The variable SHALL be listed
in `.env.example`, and the README SHALL say that opening a map loads data from the configured
provider, which sees the viewer's IP address and the area viewed, that viewers are asked first
unless the style is on Boardly's own host, and that a self-hosted style on another host still asks.
A value that isn't an absolute `http` or `https` URL SHALL stop the server from starting with an
error naming the variable.

#### Scenario: Default style
- **WHEN** `BOARDLY_MAP_STYLE_URL` is unset and Ann opens a map modal and activates "Load map"
- **THEN** the map is drawn from `https://tiles.openfreemap.org/styles/liberty`

#### Scenario: Own style
- **WHEN** `BOARDLY_MAP_STYLE_URL` is `https://maps.example.org/style.json` and Ann agrees to load maps
- **THEN** maps are drawn from that style

#### Scenario: No map without opening one
- **WHEN** Ann opens `/notes` with cards that show location chips
- **THEN** no map code is loaded and no request is made to the map provider

#### Scenario: Invalid style URL
- **WHEN** the server starts with `BOARDLY_MAP_STYLE_URL=ftp://example.org/style`
- **THEN** it doesn't start and reports an error naming `BOARDLY_MAP_STYLE_URL`
