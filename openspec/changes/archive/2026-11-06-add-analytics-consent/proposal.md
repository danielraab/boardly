# Proposal

## Why

`BOARDLY_ANALYTICS_SNIPPET` runs the operator's analytics script as soon as the page loads. That's
fine for cookieless tools, but tools such as Google Analytics or Matomo with cookies need the
visitor's consent first under the GDPR and the ePrivacy rules (in Austria § 165 (3) TKG 2021, in
Germany § 25 TDDDG). Today an operator has to paste a third-party consent manager into the snippet,
which loads from a CDN, looks and speaks unlike Boardly, and can't be checked at startup.

## What Changes

- Boardly bundles the consent manager [vanilla-cookieconsent](https://github.com/orestbida/cookieconsent)
  (v3) and turns it on only when the operator asks for it. Instances without consent don't load it.
- New optional setting `BOARDLY_CONSENT_ANALYTICS`: the name of the analytics service, for example
  `Google Analytics`. Setting it turns on the consent banner with one optional category,
  "Analytics", that names the service. Unset or empty keeps today's behaviour.
- New optional setting `BOARDLY_CONSENT_ANALYTICS_COOKIES`: comma-separated cookie names, with `*`
  as a wildcard (for example `_ga,_ga_*`). They're listed in the banner's settings and deleted
  when the visitor withdraws consent.
- The operator marks the snippet's scripts with `type="text/plain" data-category="analytics"`. The
  server still inserts the snippet unchanged. The browser doesn't run these scripts until the
  visitor accepts, and then the consent manager runs them.
- The server refuses to start, with an error naming the problem, when consent is on and:
  - `BOARDLY_ANALYTICS_SNIPPET` is empty,
  - a `<script>` in the snippet lacks `type="text/plain"` and `data-category="analytics"` (it would
    run without consent),
  - no privacy policy is configured (the banner must link to it).
- The banner asks again when the snippet or the consent settings change. Boardly derives the
  consent revision from a hash of them, so the operator doesn't have to count.
- The banner follows Boardly's look and language: en/de texts from the app's translations, the
  current theme (light and dark), a link to the privacy policy, and "Accept" and "Reject" with
  equal weight. It doesn't block the page.
- The global footer gets a "Cookie settings" / "Cookie-Einstellungen" button when consent is on.
  It opens the settings, where the visitor can change or withdraw their choice.
- When consent is off, every page shows a small, closable notice fixed at the bottom of the
  viewport, which says that Boardly uses only necessary cookies and browser storage. When an
  analytics snippet runs without consent (for example cookieless Umami or Plausible), the notice
  adds that usage statistics are collected without cookies. It links to the privacy policy when
  one is configured. Closing it is remembered in the browser, and it comes back when its text
  changes. The server tells the web app which text applies.
- The visitor's choice is stored only in the browser, in a first-party cookie set by the consent
  manager. The server keeps no consent records.
- Documentation: README (new settings, how to mark the snippet, what stays the operator's job),
  `.env.example`, and the analytics section of `docs/legal-example/privacy.md`.
- Not part of this change: a JSON config file for several services or categories, Google Consent
  Mode, and moving the map consent into the banner.

## Capabilities

### New Capabilities
- `analytics-consent`: The opt-in consent banner for the analytics snippet. Covers the settings,
  the startup checks, the consent settings and revision in the HTML document, the banner and its
  settings dialog, the stored choice, withdrawal with cookie clearing, theme and language, and the
  "only necessary cookies" notice shown when consent is off.

### Modified Capabilities
- `app-shell`: The global footer gets a "Cookie settings" button when consent is on.

## Impact

- Backend: `internal/config` (new variables, cookie name checks), a startup check of the snippet's
  scripts with `golang.org/x/net/html` (already a dependency) and of the privacy page after
  `legal.Load`, `internal/web` (new options and meta tags in `index.html`, including the notice variant), `main.go` wiring.
- Frontend: a small cookie notice component in the app shell, new dependency `vanilla-cookieconsent`, a lazily imported consent module, a theme and
  language bridge, the footer button, en/de translation keys, and CSS that maps the banner's
  colours to Boardly's theme tokens.
- API: no route is added, removed or changed, so the OpenAPI documents are unaffected.
- Deployment: optional new env vars. Operators who turn consent on must mark their snippet's
  scripts and configure a privacy policy.
- Export and import are unaffected: the consent choice lives only in the visitor's browser and
  isn't user data.
