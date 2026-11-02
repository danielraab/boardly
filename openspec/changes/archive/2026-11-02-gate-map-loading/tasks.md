# Tasks

## 1. Consent state

- [x] 1.1 Add `frontend/src/features/notes/locations/mapConsent.ts`. It provides `mapStyleUrl()`/`mapHost()` (the fallback moves out of `LocationMap.tsx`), `needsConsent()` (false when the style host equals `location.hostname`), the page-lifetime set of allowed hosts, the stored consent in localStorage `boardly.maps.allowedHosts` with `try/catch`, `allow(host, always)`, `withdraw(host)` and a `useMapConsent()` hook based on `useSyncExternalStore`. Verify with a unit test `mapConsent.test.ts` covering: same host needs no consent; consent for one host doesn't apply to another; allow without always isn't stored; withdraw removes the host; storage that throws still allows maps for the page.
- [x] 1.2 Make `LocationMap.tsx` use `mapStyleUrl()` from the new module, and verify that the existing Locations tests still pass.

## 2. Gate in LazyMap

- [x] 2.1 In `LazyMap` (`LocationDialogs.tsx`), render the placeholder when consent is needed, and run the dynamic import only after consent. The placeholder has the text with the host, a "Load map" button and an unchecked "Always load maps in this browser" checkbox, in the same box as the map. Verify in `Locations.test.tsx` that opening the map modal shows the placeholder and doesn't import `LocationMap` or request the style.
- [x] 2.2 Below a map that loaded because of stored consent (or right after "Always" was checked), show the "Don't load maps automatically" link, which withdraws the consent and is then replaced by "Maps will ask again.". Verify with a test: store consent, open the modal, activate the link, and check that storage no longer has the host and the text changed.
- [x] 2.3 Verify with tests that the location form can add a location from its fields alone without consent, and that "Load map" once lets a second map on the same page (modal → Edit → form) load without asking again.

## 3. Texts and docs

- [x] 3.1 Add `notes.locations.consent.{text,load,always,withdraw,withdrawn}` to `frontend/src/locales/en.json` and `de.json` with the spec's wording, and verify that the i18n key check / `make check` passes.
- [x] 3.2 Update the README "Maps" section: viewers are asked before a map loads, consent is per browser and per host and can be withdrawn below a map, there is no prompt for a style on Boardly's own host, and such a style must not reference tiles on other hosts. Add a CHANGELOG entry. Verify by reading the rendered sections.

## 4. Verification

- [x] 4.1 Run `make check` and `make test` and make sure both pass. In the running app, check in the network tab that opening a map modal makes no request to `tiles.openfreemap.org` until "Load map" is pressed, also as a visitor of a public dashboard.
