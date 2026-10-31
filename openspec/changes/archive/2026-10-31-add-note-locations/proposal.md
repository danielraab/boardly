# Proposal

## Why

Notes about trips, meeting points, gardens or anything else tied to a place can only hold the place
as text or as a link to a map site. Users want to put a location into a note, see where it is on a
map, and set it by typing coordinates, using the device's position or picking a point on a map.

## What Changes

- New **location block** in the note document: `{"type": "location", "lat": <number>, "lng":
  <number>, "label": "<optional text>"}`. A note can hold any number of them, anywhere between
  other blocks. The document version stays `1`. In Markdown, a location is a paragraph of its own
  with a `geo:` link, `[<label or coordinates>](geo:<lat>,<lng>)`, and such a paragraph is read back
  as a location block.
- In the content, a location is shown as a **location chip**: a pin icon followed by its label, or
  by its coordinates when it has none. Activating the chip opens a **map modal** with a map centred
  on a pin at the location, the label and the coordinates. In the editors, the map modal has an
  **Edit** button that opens the location form.
- A **location form** (modal) to create or edit a location, with three ways to set the point:
  entering latitude and longitude, **"Use my location"** (the device's position) and **clicking on
  the map**. It also has the optional label, and Remove (when editing), Cancel and Save.
- A new **"Add location"** button in the editor toolbar, right after "Add attachment", opens the
  location form and inserts the new location at the cursor, or at the end of the note.
- Maps are vector maps loaded only when a map is first shown, so notes without locations, and
  showing a location chip, load no map code and no map data. The map style is configurable with
  the new `BOARDLY_MAP_STYLE_URL` (default: OpenFreeMap's "liberty" style).
- Not in this change: the Memos API's `location` field (separate change `memos-location`), map
  previews on cards, address search (geocoding), and showing several locations of a note on one map.

## Capabilities

### New Capabilities
- `note-locations`: how locations are shown in notes (chip, map modal), created and edited
  (location form with coordinates, device position and map), inserted from the toolbar, and the
  map configuration.

### Modified Capabilities
- `note-document`: the document format gains the location block, validation checks it, the content
  editor's toolbar gains "Add location", and Markdown in and out handles locations as `geo:` links.

## Impact

- Backend: `internal/notes` document type and `ParseDoc` validation, `ToMarkdown` and
  `FromMarkdown`; `internal/config` reads `BOARDLY_MAP_STYLE_URL`; `internal/web` injects it into
  `index.html`. No migration: locations are part of the content.
- Frontend: new dependency `maplibre-gl` (lazy-loaded chunk with its CSS); Lexical `LocationNode`
  and mapping in `document/lexical.ts`; location chip, map modal and location form; toolbar button;
  rendering in `NoteRenderer` (cards and widgets); en + de strings.
- Configuration and docs: `.env.example`, README (configuration, privacy note that map tiles are
  loaded from the configured provider).
- Privacy: opening a map loads tiles from the map style's provider, which sees the visitor's IP
  address and the area viewed. Nothing is loaded until a map is opened.
