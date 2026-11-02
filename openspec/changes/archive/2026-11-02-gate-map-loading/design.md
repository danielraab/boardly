# Design

## Context

Every map goes through `LazyMap` in `frontend/src/features/notes/locations/LocationDialogs.tsx`.
It dynamically imports `./LocationMap`, which is the only module that loads MapLibre, and shows
"Loading map…" or "The map couldn't be loaded." around it. The map modal (`LocationMapDialog`)
and the location form (`LocationForm`) both render `LazyMap`. Public dashboards use the same
components. The style URL reaches the frontend through the `boardly-map-style` meta tag, which
the Go web handler fills in from `BOARDLY_MAP_STYLE_URL`.

## Goals / Non-Goals

**Goals:**
- One gate in `LazyMap`, so every current and future map is covered.
- No request to the provider and no MapLibre download before consent.

**Non-Goals:**
- Proxying or self-hosting tiles.
- An account-level setting. Consent is per browser so that it also works for anonymous visitors.
- Detecting that a style on the page's host points to tiles on other hosts (see Risks).

## Decisions

**Gate inside `LazyMap`, before the dynamic import.** `LazyMap` gets a third state, "needs
consent", which renders the placeholder in the same box (same `className`, so the modal and form
layouts don't change). `import("./LocationMap")` only runs once consent exists. The alternative of
gating in `LocationMap` itself would still download MapLibre, and gating in each dialog would
duplicate the logic.

**Consent module `mapConsent.ts`.** It holds:
- `mapHost()`: the host name of the style URL from the meta tag, with the same fallback as
  `LocationMap`. The fallback moves into this module so that there is one source.
- `needsConsent()`: false when `mapHost() === location.hostname`.
- A page-lifetime `Set<string>` of hosts allowed with "Load map". A module-level variable is enough
  because it is cleared on reload, which is what "until the page is reloaded" means.
- Stored consent: localStorage key `boardly.maps.allowedHosts`, a JSON array of host names, read
  and written in `try/catch` like `useStoredIdSet`. A list rather than one key per host keeps
  withdrawing simple and the storage tidy.
- A small subscription (`useSyncExternalStore`), so that two maps open at once, or the modal
  followed by the form via Edit, see the same state without prop threading.

**Compare against the page's host, not `BOARDLY_BASE_URL`.** The browser is already talking to
the page's host, so loading from that host adds no new recipient. This is exactly the "same as the
base domain" rule in practice, and it needs no backend change or new meta tag. The comparison
uses exact host names: `maps.example.org` and `boardly.example.org` are different hosts and still
ask. Matching the registrable domain would need a public-suffix list and could wrongly treat other
parties' subdomains as first-party.

**Withdrawal link only for stored consent.** A map loaded with "Load map" alone has no stored
consent to withdraw, so the link only appears when the map loaded because of stored consent, or
right after the viewer checked "Always". It sits below the map as small muted text, so it doesn't
compete with the modal's actions.

**Texts** go under `notes.locations.consent.*` in en and de. The host is inserted with
interpolation and shown in `<strong>`/`<code>` style so that it is easy to spot.

## Risks / Trade-offs

- [A style on the page's host can still reference tiles, glyphs or sprites on other hosts] → The
  README says that the exception relies on the whole style being self-hosted. The spec defines the
  rule by the style's host, which the operator controls.
- [Visitors clear their storage and get asked again] → Acceptable; this is the expected result of
  clearing site data.
- [The checkbox is easy to overlook] → It is unchecked by default, as consent requires, and it sits
  directly under the button.

## Migration Plan

Frontend only, no data migration. After an upgrade, every viewer sees the placeholder once.
Rolling back removes the gate, and the stored key is simply ignored.
