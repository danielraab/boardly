# note-locations Specification

## Purpose
Lets users put places into their notes as location blocks, see them on a map, and create or edit
them by entering coordinates, using the device's position or picking a point on a map.

## Requirements

### Requirement: Location chip
Wherever a note's content is shown (the editor page and the editor modal, inline editing on cards,
note cards and note widgets, also for visitors of public dashboards), a location block SHALL be
shown as its own block: a chip with a pin icon and the location's label, or, without a label, its
coordinates as `<lat>, <lng>` with the latitude and longitude written with up to 6 decimals and no
trailing zeros. The chip SHALL have the accessible name "Location: <label or coordinates>" /
"Ort: <label or coordinates>". Activating the chip SHALL open the map modal of the location, also
on note cards, where it doesn't start inline editing. Showing a chip SHALL load no map.

#### Scenario: Chip with label
- **WHEN** "Trip" contains a location with the label "Stephansplatz" at 48.208176, 16.373819
- **THEN** the card, the editor and a note widget of "Trip" show a chip with a pin and "Stephansplatz"

#### Scenario: Chip without label
- **WHEN** "Trip" contains a location without a label at 48.2, 16.37
- **THEN** the chip shows "48.2, 16.37"

#### Scenario: Several locations
- **WHEN** "Trip" contains a paragraph, a location, another paragraph and two more locations
- **THEN** the three chips are shown in that order between the paragraphs

#### Scenario: Chip on a card
- **WHEN** Ann clicks the chip "Stephansplatz" on the card "Trip" on `/notes`
- **THEN** the map modal of "Stephansplatz" opens and the card doesn't start inline editing

### Requirement: Map modal
The map modal SHALL show a map centred on a pin at the location, at a street-level zoom, with the
location's label as its title (the coordinates when it has none), the coordinates, and the map
data's attribution. The map SHALL be movable and zoomable. Escape, the close button or a click
outside the modal SHALL close it and return focus to the chip. In the editor page, the editor
modal and inline editing on cards, the map modal SHALL have an "Edit" / "Bearbeiten" button that
opens the location form for the location; elsewhere (note cards that aren't being edited, note
widgets, public dashboards) it has no Edit button. When the map can't be loaded, the modal SHALL
still show the label and the coordinates and the message "The map couldn't be loaded." / "Die
Karte konnte nicht geladen werden."

#### Scenario: Open the map
- **WHEN** Ann activates the chip "Stephansplatz" in the editor of "Trip"
- **THEN** a modal shows a map with a pin at 48.208176, 16.373819, the title "Stephansplatz", the coordinates and an Edit button

#### Scenario: No edit for visitors
- **WHEN** a visitor of Ann's public dashboard activates the chip "Stephansplatz" in a note widget
- **THEN** the map modal shows the location without an Edit button

#### Scenario: Map unavailable
- **WHEN** the map style can't be loaded and Ann opens the map modal of "Stephansplatz"
- **THEN** the modal shows "Stephansplatz", the coordinates and "The map couldn't be loaded."

### Requirement: Location form
The location form SHALL be a modal titled "Add location" / "Ort hinzufügen" when creating and
"Edit location" / "Ort bearbeiten" when editing, with:
- the fields "Latitude" / "Breitengrad" and "Longitude" / "Längengrad", which accept decimal numbers
  with `.` or `,` as the decimal separator;
- a "Use my location" / "Meinen Standort verwenden" button;
- a map with a pin at the current point;
- the optional field "Label" / "Bezeichnung", at most 200 characters;
- "Remove" / "Entfernen" (only when editing), "Cancel" / "Abbrechen" and "Save" / "Speichern".

The three ways to set the point SHALL stay in step: entering valid coordinates SHALL move the pin
to them, clicking or tapping the map SHALL move the pin there and fill both fields, and "Use my
location" SHALL ask the device for its position and, when it gets one, fill both fields, move the
pin and centre the map on it. Coordinates SHALL be rounded to 6 decimals. When creating, the form
starts without a point and with a map of the whole world, and the pin appears once a point is set;
when editing, it starts with the location's point, label and pin.

Save SHALL be possible only with a latitude from -90 to 90 and a longitude from -180 to 180; an
invalid field SHALL show the translated message "Enter a latitude between -90 and 90." / "Gib
einen Breitengrad zwischen -90 und 90 ein." or "Enter a longitude between -180 and 180." / "Gib
einen Längengrad zwischen -180 und 180 ein.". When the device's position can't be found, the form
SHALL say why: "Access to your location was denied." / "Der Zugriff auf deinen Standort wurde
verweigert.", or "Your location couldn't be found." / "Dein Standort konnte nicht gefunden
werden." for any other reason, including a timeout or a browser without location support. The
form SHALL stay usable with the fields alone when the map can't be loaded.

Saving when creating SHALL insert the location (see "Adding locations"); saving when editing SHALL
replace the point and label of the location; Remove SHALL delete the location block from the
content. Cancel, Escape and a click outside the form SHALL close it without changes. Each of these
changes counts as a change and is saved automatically.

#### Scenario: Enter coordinates
- **WHEN** Ann enters 48,208176 as the latitude and 16.373819 as the longitude
- **THEN** the pin moves to 48.208176, 16.373819 and Save becomes possible

#### Scenario: Pick on the map
- **WHEN** Ann clicks a point on the map of the location form
- **THEN** the pin moves there and the fields show its latitude and longitude with at most 6 decimals

#### Scenario: Use my location
- **WHEN** Ann taps "Use my location" and allows the browser to share her position
- **THEN** the fields show her position, the pin moves there and the map is centred on it

#### Scenario: Location denied
- **WHEN** Ann taps "Use my location" and denies the browser's request
- **THEN** the form shows "Access to your location was denied." and the fields keep their values

#### Scenario: Invalid latitude
- **WHEN** Ann enters 95 as the latitude
- **THEN** the form shows "Enter a latitude between -90 and 90." and can't be saved

#### Scenario: Edit a location
- **WHEN** Ann opens the map modal of "Stephansplatz", activates Edit, changes the label to "Dom" and saves
- **THEN** the chip shows "Dom" and after the autosave the note's location block has the label "Dom" and the same coordinates

#### Scenario: Remove a location
- **WHEN** Ann opens the location form of "Stephansplatz" from its map modal and activates Remove
- **THEN** the chip is gone from the content and the change is saved

### Requirement: Adding locations
The content editor's toolbar SHALL offer an "Add location" / "Ort hinzufügen" button with a pin
icon, right after "Add attachment". It SHALL open the location form to create a location. Saving
the form SHALL insert a location block at the cursor when the content editor has or last had one,
and at the end of the note when it never had one; a location inserted inside a paragraph splits
it. The cursor SHALL then be placed after the location. Cancelling the form inserts nothing.

#### Scenario: Insert at the cursor
- **WHEN** Ann puts the cursor between two paragraphs of "Trip", activates "Add location", picks a point on the map and saves
- **THEN** a location chip appears between the two paragraphs and the change is saved

#### Scenario: Insert at the end
- **WHEN** Ann opens "Trip" and adds a location with "Add location" without placing the cursor
- **THEN** the location chip appears at the end of the note

#### Scenario: Cancel
- **WHEN** Ann activates "Add location" and then Cancel
- **THEN** the content doesn't change

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
