# Tasks

## 1. Backend: document format

- [x] 1.1 Add `BlockLocation` and the `Lat`, `Lng` (`*float64`) and `Label` (`*string`) fields to `notes.Block`; reject location fields on other block types; validate range, rounding to 6 decimals and the label rules in `checkBlock`; verify with `doc_test.go` cases for a valid block, `lat: 0`, "Latitude out of range", "Location without longitude", "Too precise", an empty or 201-character label, a label with a line break and `lat` on a paragraph
- [x] 1.2 Skip location blocks in plain text and the display title; verify with a test for "Label isn't the title"
- [x] 1.3 Write locations in `ToMarkdown` as `[label](geo:lat,lng)` or `[lat, lng](geo:lat,lng)`, with the label escaped like text and the shared coordinate formatter; verify with `tomarkdown_test.go` cases for "Locations" and a label with Markdown characters
- [x] 1.4 Read a paragraph that is only a `geo:` link into a location block in `FromMarkdown` (altitude and `;` parameters ignored, label rules, out-of-range or malformed URIs kept as text); verify with `markdown_test.go` cases for "Location", "Location without label", "Location link inside text", `geo:91,0` and `geo:48.2,16.37;u=30`, and that the "Round trip" test covers documents with locations

## 2. Backend: map configuration

- [x] 2.1 Read `BOARDLY_MAP_STYLE_URL` in `internal/config` (default `https://tiles.openfreemap.org/styles/liberty`, absolute http(s) URL required); verify with config tests for the default, "Own style" and "Invalid style URL"
- [x] 2.2 Add the `<meta name="boardly-map-style">` marker with the default to `frontend/index.html` and make `internal/web` replace its content with the configured URL (HTML-escaped); verify with a `web_test.go` case like `TestIndexCarriesAppTitle`
- [x] 2.3 Add `BOARDLY_MAP_STYLE_URL` to `.env.example` and the README configuration table, with the privacy note about the map provider; verify by reading both files

## 3. Frontend: document and editor node

- [x] 3.1 Add `LocationBlock` to `document/types.ts` and a shared `formatCoordinate` / `formatPoint` helper (up to 6 decimals, no trailing zeros); verify with unit tests for `48.2`, `16.373819`, `0` and `-0.0000001`
- [x] 3.2 Add `LocationNode` (`DecoratorNode`, block, no text content, `setPoint`/`getPoint`) and map it both ways in `document/lexical.ts`; verify with `lexical.test.ts` round-trip cases with and without a label
- [x] 3.3 Extract the "insert at the cursor or at the end" logic from `AttachmentsPlugin` into `editor/insertBlock.ts` and use it there; verify the existing attachment insertion tests in `Attachments.test.tsx` still pass unchanged

## 4. Frontend: map and dialogs

- [x] 4.1 Add the `maplibre-gl` dependency (exact version) and `features/notes/locations/LocationMap.tsx` (reads the style from the meta tag, one map and marker, `onPick`, `onError`, attribution), imported only with `React.lazy`; verify with `pnpm build` that `maplibre-gl` ends up in its own chunk, not in the main bundle
- [x] 4.2 Add `LocationChip` (pin icon, label or coordinates, accessible name, stops propagation); verify with tests for "Chip with label", "Chip without label" and that rendering a chip doesn't load the `LocationMap` module
- [x] 4.3 Add `LocationMapDialog` (Base UI Dialog: title, coordinates, lazy map, Edit only when editable, error message, focus return); verify with tests (map module mocked) for "Open the map", "No edit for visitors" and "Map unavailable"
- [x] 4.4 Add `LocationFormDialog` (create or edit titles, lat/lng fields that accept `,`, validation messages, label with 200-character limit, "Use my location" with the error messages, map pick, Remove/Cancel/Save); verify with tests for "Enter coordinates", "Pick on the map" (stub map), "Use my location", "Location denied", "Invalid latitude" and Cancel

## 5. Frontend: editor integration

- [x] 5.1 Add `LocationsPlugin` (context for chips, view and form state, `OPEN_LOCATION_FORM_COMMAND`, insert with the shared helper, update and remove by node key) and register `LocationNode` and the plugin in `NoteEditor`; verify with editor tests for "Insert at the cursor", "Insert at the end", "Edit a location" and "Remove a location", each followed by the autosave of the expected document
- [x] 5.2 Add the "Add location" button (MapPin icon) after "Add attachment" in `Toolbar.tsx`; verify with a test for "Location button in the toolbar" and that it is disabled when the editor is
- [x] 5.3 Render location blocks in `NoteRenderer` with the chip and a read-only map dialog; verify with `NoteRenderer.test.tsx` for several locations in order and "Chip on a card" (no inline editing starts)

## 6. Translations and docs

- [x] 6.1 Add every new string in en and de under `notes.editor.location.*` and `notes.locations.*`; verify with the i18n key-parity test
- [x] 6.2 Mention locations in the README's notes section; verify by reading it

## 7. Verification

- [x] 7.1 Run `make check` and `make test`; verify both pass
- [x] 7.2 In the running app, add a location by typing coordinates, by GPS and by clicking the map, edit and remove one, open a chip from a card and from a public dashboard widget, and check in the network panel that `/notes` makes no request to the map provider before a map is opened
