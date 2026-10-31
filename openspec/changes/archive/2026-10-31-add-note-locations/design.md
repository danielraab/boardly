# Design

## Context

- `notes.Block` (`backend/internal/notes/doc.go`) is one flat struct with optional fields for every
  block type. `checkBlock` rejects fields that don't belong to the type, for example `ID` on
  anything but an attachment. `ParseDoc`, `FromMarkdown` and `ToMarkdown` are pure.
- The attachment block is the closest existing model: a Lexical `DecoratorNode`
  (`editor/attachments.tsx`) that renders a React view through a context
  (`EditorAttachmentsContext`, which provides `openImage`), mapped in `document/lexical.ts` and
  rendered read-only in `NoteRenderer.tsx`. `AttachmentsPlugin` keeps the last range selection in a
  ref, so a toolbar button that took the focus can still insert at the cursor with
  `$insertNodeToNearestRoot`.
- Modals use Base UI `Dialog` (see `features/attachments/ImageViewer.tsx`).
- `internal/web` already rewrites `index.html` at startup (app title, analytics snippet), which is
  how runtime configuration reaches the SPA without an API call.
- The app sends no Content-Security-Policy for its own pages, so loading map tiles and running
  MapLibre's blob workers needs no header changes.

## Goals / Non-Goals

**Goals:**
- No map code and no request to the map provider until a map is actually shown.
- One map component shared by the map modal and the location form.
- The editor, cards, widgets and Markdown agree on how coordinates are written.

**Non-Goals:**
- Geocoding or address search, routing, several pins on one map, offline tiles, static map images.
- Localized number display: coordinates are always written with `.` (they're copied into other
  apps). The form only *accepts* `,` as the decimal separator.

## Decisions

### 1. Document shape: `lat`, `lng`, optional `label` on the flat `Block`

```go
Lat   *float64 `json:"lat,omitempty"`   // pointers so that 0 (equator / Greenwich) isn't "missing"
Lng   *float64 `json:"lng,omitempty"`
Label *string  `json:"label,omitempty"`
```

`checkBlock` gets a general rule like the existing one for `ID`: location fields on any other
block type are rejected. The `location` case requires both coordinates in range, rounds them to 6
decimals (`math.Round(x*1e6)/1e6`), and rejects an empty label, one longer than 200 characters
(counted in runes) or one with a line break. Pointers, unlike plain floats with `omitempty`, keep
`lat: 0` from being dropped when the block is encoded.

Alternatives: a nested `{"point": {...}}` object or GeoJSON `[lng, lat]`. Rejected: the flat
shape matches the other blocks, and `lat`/`lng` names avoid GeoJSON's easy-to-confuse order.

### 2. Label isn't text

Like an attachment, a location contributes nothing to the display title or the plain text. That
keeps a note that starts with a location from being titled after the place name, and keeps
`text.ts`/`doc.go` plain-text code simple. Trade-off: a label isn't found by text search, if it is
added later. That can be revisited without changing the format.

### 3. Markdown: `geo:` URI (RFC 5870)

`[label](geo:lat,lng)` in a paragraph of its own. `geo:` opens a maps app on most phones, so the
Markdown stays useful in Memos and Karakeep clients. `FromMarkdown` turns it back into a block only
when the link is alone in its paragraph (the same rule as attachment file links), and otherwise
keeps the text, because `geo:` stays outside the allowed inline link schemes. Coordinates are
written by one shared formatter: `strconv.FormatFloat(x, 'f', -1, 64)` after rounding, and in the
frontend `String(Number(x.toFixed(6)))`. Both give `48.2` and `16.373819`. A link text equal to
`<lat>, <lng>` is read as "no label".

### 4. Frontend structure

```
Toolbar ── [📍 Add location] ── dispatch OPEN_LOCATION_FORM_COMMAND ──┐
                                                                     ▼
LocationNode (DecoratorNode)                 LocationsPlugin (inside LexicalComposer)
 decorate() → <LocationChip …/>                ├─ tracks lastSelection (shared helper, see 5)
     │ onActivate(nodeKey)                     ├─ state: { view: key } | { form: key | "new" }
     └──── EditorLocationsContext ───────────▶ ├─ <LocationMapDialog editable onEdit=…/>
                                               └─ <LocationFormDialog onSave/onRemove/…/>
                                                      save → editor.update: node.setPoint(…)
                                                             or insert $createLocationNode(…)
NoteRenderer (read-only)
 location block → <LocationChip/> → <LocationMapDialog editable={false}/>
```

- `LocationChip`, `LocationMapDialog` and `LocationFormDialog` live in
  `features/notes/locations/` and contain no map code. Both dialogs render
  `const LocationMap = lazy(() => import("./LocationMap"))` inside `Suspense`. `LocationMap.tsx`
  is the only module that imports `maplibre-gl` and its CSS, so Vite puts both in a separate chunk.
- `LocationMap` props: `point | null`, `onPick?(point)` (click handler, only in the form) and
  `onError()` (so the dialogs show "The map couldn't be loaded" and the form stays usable with its
  fields). It creates one `maplibregl.Map` per mount and moves one `Marker`. The form centres on
  the point when it comes from GPS or from the fields, but not when it comes from a click on the
  map (the user is looking there already).
- Opening "Edit" from the map modal closes the map modal and opens the form. When the form closes,
  focus returns to the chip. Stacked dialogs are avoided because Base UI focus handling with two
  modal dialogs is fiddly, and the map modal adds nothing behind the form.
- The chip on cards stops click propagation, as images do, so the map opens instead of inline
  editing.

### 5. Shared "insert at the cursor or at the end"

`AttachmentsPlugin` already does this: it restores the last range selection when the current one is
gone, `$insertNodeToNearestRoot` when there is a selection, otherwise `$getRoot().append`. Extract
it into `editor/insertBlock.ts` (`useLastSelection(editor)` +
`$insertBlocksAtCursor(nodes, last)`), and use it from both plugins. This is a small, behaviour-
preserving refactor covered by the existing attachment insertion tests.

### 6. Map style URL via `index.html`

`BOARDLY_MAP_STYLE_URL` (validated as an absolute http(s) URL in `internal/config`) is injected by
`internal/web` as `<meta name="boardly-map-style" content="…">`, replacing a marker meta in
`frontend/index.html` that carries the OpenFreeMap default. This works the same way in `vite dev`
and in the embedded build. The frontend reads it once in `LocationMap`. An API endpoint was
considered, but the map would then wait for one more request, and visitors of public dashboards
have no session.

### 7. Device position

`navigator.geolocation.getCurrentPosition` with `enableHighAccuracy: true`, `timeout: 10000` and
`maximumAge: 60000`. `PERMISSION_DENIED` → "denied", anything else (including a missing
`navigator.geolocation`) → "couldn't be found". There's no secure-context check: browsers that
refuse on plain HTTP report it as an error, and the user sees "couldn't be found". While a request
is running, the button is disabled and shows a spinner.

## Risks / Trade-offs

- [Bundle size: `maplibre-gl` is about 800 KB minified] → only in the lazy chunk, and the chip
  never loads it. A test checks that rendering a chip doesn't import the map module.
- [OpenFreeMap availability or policy changes] → the style is configurable, and the dialogs degrade
  to label and coordinates.
- [Privacy: the tile provider sees IP and viewed area, also for visitors of public dashboards] →
  only after an explicit click. Documented in README.
- [jsdom has no WebGL, so MapLibre can't run in Vitest] → tests mock `./LocationMap` with a stub
  that exposes `onPick` as a button. Real map behaviour is checked manually in the browser (task 7).
- [Floating-point equality in round-trip tests] → coordinates are rounded to 6 decimals on both
  sides before comparison, and formatting is shared.
