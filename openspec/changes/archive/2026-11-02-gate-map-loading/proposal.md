# Proposal

## Why

Opening a map of a note location makes the browser load the style, tiles, fonts and sprites from
the map provider, by default `tiles.openfreemap.org`. The provider then sees the viewer's IP
address and the area viewed. For an instance open to others, and especially for anonymous visitors
of public dashboards, there is no legal basis for this transfer under GDPR Art. 6 unless the viewer
agrees first. The map is a convenience: the label and coordinates are shown without it, and the
location form works with its fields alone.

## What Changes

- Before a map is loaded (the map modal and the location form, for signed-in users and visitors of
  public dashboards alike), a placeholder SHALL say which host the map is loaded from and that it
  sees the viewer's IP address and the area shown, with a "Load map" button and an "Always load
  maps in this browser" checkbox. Nothing is requested from the provider, and no map code is
  loaded, until the viewer agrees.
- "Load map" without the checkbox allows maps until the page is reloaded. With the checkbox, the
  consent is stored in the browser's localStorage for that provider host, so a changed
  `BOARDLY_MAP_STYLE_URL` on another host asks again.
- A map loaded because of stored consent SHALL show a small "Don't load maps automatically" link
  that removes the stored consent (GDPR Art. 7(3): withdrawing must be as easy as giving it).
- No placeholder when the map style is on the same host as the page, which means a self-hosted
  style. No new environment variable.
- README "Maps" section and `note-locations` spec updated; English and German texts.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `note-locations`: "Map configuration" gains the consent gate, its storage per provider host, the
  withdrawal link and the self-hosted exception. "Map modal" and "Location form" show the
  placeholder in place of the map until the viewer agrees, and the form stays usable without it.

## Impact

- Frontend: `frontend/src/features/notes/locations/LocationDialogs.tsx` (`LazyMap`), a small
  consent module next to it, i18n keys under `notes.locations.consent.*` (en, de), tests.
- Backend: none. The style URL is already in `index.html`'s `boardly-map-style` meta tag.
- Docs: README "Maps" section, CHANGELOG.
