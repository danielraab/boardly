# Design

## Context

- `internal/web` writes instance settings into `index.html` at startup: it replaces markers (title,
  map style) and inserts meta tags (`boardly-imprint`, `boardly-privacy`) and the raw analytics
  snippet before `</head>`. The frontend reads those meta tags (`lib/legalLinks`,
  `mapConsent.mapStyleUrl`).
- `config.Load` checks single variables. Checks that need another subsystem, such as whether a
  privacy policy exists, can only run after `legal.Load` in `main.go`.
- Themes set `data-theme` on `<html>`, and each theme has a `mode` (`light`/`dark`) in
  `src/theme`. Colours are CSS variables (`--color-paper`, `--color-surface`, `--color-ink`,
  `--color-muted`, `--color-line`, `--color-accent`).
- The map consent (`features/notes/locations/mapConsent.ts`) is a separate, contextual consent and
  stays as it is.

## Goals / Non-Goals

**Goals:**
- Instances without consent download no consent library. They only get the small cookie notice.
- Misconfigurations that would run analytics without consent stop the server instead of shipping.
- The banner can't drift from Boardly's look, language or privacy link through operator config.

**Non-Goals:**
- Passing the library's own config through. The env vars cover one service, and a JSON file for
  more is a later change.
- Consent records on the server, Google Consent Mode, and a "maps" category.
- Showing the banner in the Vite dev server, which doesn't insert the snippet either.

## Decisions

### vanilla-cookieconsent v3, bundled and imported lazily
The library already blocks `<script type="text/plain" data-category="…">` and runs these scripts
when the category is accepted, supports revisions, `autoClear` of cookies with an optional reload,
translations with `setLanguage`, a `cc--darkmode` class, and `equalWeightButtons`. Its markup lives
under `#cc-main`, and its colours are `--cc-*` variables.

It's added to `frontend/package.json` and loaded with `import()` from a small `lib/consent` entry,
so Vite puts the library and its CSS into a separate chunk that is only fetched when the
`boardly-consent-analytics` meta tag exists.

- *Alternative: our own gate with `<template>` and script recreation.* Less code to depend on, but
  it would rebuild what the library already does well (settings dialog, revisions, clearing,
  accessibility).
- *Alternative: load from a CDN.* Rejected: loading the consent manager from a third party already
  sends the visitor's IP address there before any consent.

### Settings travel as meta tags
`web.Options` gets `ConsentService`, `ConsentCookies []string` and `ConsentRevision int`. When the
service is set, `internal/web` inserts three escaped meta tags next to the legal ones, as the specs
describe. This matches how the map style and the legal links travel, so there's no new mechanism
and no API route.

- *Alternative: an inline `<script type="application/json">`.* Needs separate escaping rules for the
  script context, with no gain for three flat values.

### Checks live in a small `internal/consent` package
- `config.Load` reads and trims `BOARDLY_CONSENT_ANALYTICS` and `BOARDLY_CONSENT_ANALYTICS_COOKIES`,
  checks the cookie names (`^[A-Za-z0-9_.*-]+$`, not only `*`) and the cookies-without-service case.
  These need no other subsystem.
- `consent.CheckSnippet(snippet string) error` parses the snippet with
  `golang.org/x/net/html.ParseFragment` in a `<body>` context, so `<img>`, `<noscript>` and
  `<iframe>` come back as elements instead of being moved around as they would be in a `<head>`
  context. Every top-level node must be a comment, whitespace text, or a `<script>` with
  `type="text/plain"` and `data-category="analytics"`. Matching is case-insensitive on the
  attribute values, as in HTML. Errors name `BOARDLY_ANALYTICS_SNIPPET` and the offending element.
- `main.go` runs `CheckSnippet` and the "snippet not empty" and "privacy policy configured"
  checks (`legalPages.Href(legal.Privacy) != ""`) after `legal.Load` and before the server listens.

- *Alternative: a regex over `<script` tags.* Rejected: it misses `<img>` pixels, attribute
  quoting variants and comments.
- *Alternative: rewriting unmarked scripts to `text/plain`.* Rejected as too clever: the operator
  should see what is served.

### Revision derived from a hash
`consent.Revision(snippet, service, cookies)` takes SHA-256 over the three values, joined with a
separator that can't appear in them, uses the first four bytes as an unsigned number masked to 31
bits, and adds 1. The library treats revision `0` as "revisions off", so the result is never 0, and
31 bits keep it a safe JavaScript integer. The same input gives the same revision on every restart,
and a collision only means a visitor isn't asked again, with a chance of about 1 in 2³¹.

- *Alternative: a `BOARDLY_CONSENT_REVISION` the operator raises.* Easy to forget, and forgetting
  it means old consent covers a new tool.

### Boardly builds the library config
`lib/consent` builds the config from the meta tags:

- `cookie`: name `boardly_consent`, `expiresAfterDays: 182`, `sameSite: "Lax"`,
  `secure: location.protocol === "https:"`.
- `revision` from the meta tag.
- `categories.necessary` with `enabled: true, readOnly: true`, and `categories.analytics` with one
  service labelled with the service name and
  `autoClear: { cookies: [...names as RegExp] }`. The reload after a withdrawal is done in
  `onChange`, not with the library's `reloadPage`, so the parent-domain clearing (below) runs
  before it. A name such as `_ga_*` becomes
  `/^_ga_.*$/` after escaping every other regex character.
- `guiOptions`: `consentModal` with layout `box`, position `bottom right`,
  `equalWeightButtons: true`, and `preferencesModal` with `equalWeightButtons: true`.
  `disablePageInteraction: false`, and `hideFromBots: false` so the banner also shows in
  automated browsers (they don't run the scripts either before consent).
- `language.translations.en` / `.de`, built from new i18n keys under `consent.*` in
  `locales/en.json` and `de.json`. The library takes HTML in descriptions, so the service name,
  cookie names and privacy href are escaped before they're put in. The privacy link is an
  `<a href>` from `legalLink("privacy")`. External links get `target="_blank" rel="noopener noreferrer"`.
  For a page served from files, a click handler on `#cc-main` routes the link through TanStack
  Router, so the app doesn't reload.

### Bridges to language and theme
- Language: after `run()`, an `i18next.on("languageChanged")` listener calls
  `CookieConsent.setLanguage(lng, true)`.
- Theme: a subscription to the theme module toggles `cc--darkmode` on `<html>` from the active
  theme's mode, including the "system" choice.
- Colours: a CSS file imported by the lazy module maps `#cc-main`'s `--cc-*` variables to Boardly's
  tokens (`--cc-bg: var(--color-surface)`, `--cc-primary-color: var(--color-ink)`,
  `--cc-secondary-color: var(--color-muted)`, `--cc-separator-border-color: var(--color-line)`,
  buttons from `--color-accent`, and so on). Accept and reject use the same button variables.
  Because the tokens change with `data-theme`, all seven themes follow without per-theme CSS.

### Cookie notice without consent is Boardly's own component
When consent is off, the vanilla-cookieconsent chunk isn't loaded at all. A small React component,
`CookieNotice`, is rendered by `AppShell` next to the footer. It reads the `boardly-cookie-notice`
meta tag through `lib/consent/settings.ts`, so the server decides the variant: it knows whether a
snippet is set, while the web app can't tell reliably.

- Position: `fixed` bottom-left, with a max width of about 24rem and side gutters at narrow widths.
  It's shifted above the mini player bar with the same offset the footer already uses for it.
  Surface, line and ink tokens give it the theme's colours.
- It's a `<section aria-label>` with a plain close `<button>` (lucide `X`), not a
  dialog, so it doesn't trap focus.
- Dismissal is stored in `localStorage` under `boardly.cookieNotice.dismissed` as the variant
  string, like the theme and language settings. A stored value for another variant shows the
  notice again. With blocked storage, a module-level flag hides it until reload, as in the map
  consent.

- *Alternative: use vanilla-cookieconsent's banner with only a "necessary" category.* It would load
  the library on every instance and show a consent dialog where there is nothing to consent to.
- *Alternative: storing the dismissal in a cookie.* Not needed: the server never reads it, and
  localStorage is already used for comparable settings.

### Footer button
`AppFooter` reads the meta tag through a `consentEnabled()` helper and renders a `<button>`
"Cookie settings" after the legal links. Activating it calls `openConsentSettings()`, which awaits
the lazy module and calls `CookieConsent.showPreferences()`.

### Clearing on parent domains
The library's `autoClear` erases cookies on the current host. Analytics tools such as GA set them
on the registrable domain, so `.example.org` for `dash.example.org`. In `onChange`, when analytics
was withdrawn, `lib/consent` also expires each matching cookie on every parent domain of
`location.hostname` (stopping before the last label) before the reload. Setting an expired cookie
on a domain that isn't allowed is a no-op in browsers, so trying all levels is safe.

## Risks / Trade-offs

- [The operator forgets to mark a script] → The startup check refuses to start, and the README
  shows a marked example for GA and Matomo.
- [Analytics loaded by a marked script injects further scripts or pixels] → Those run only after
  consent, because their loader only runs after consent. That's correct.
- [The library changes its API or markup in a major version] → Pin `^3`. The Boardly-owned config
  layer is the only place that touches the library, and the tests cover banner texts, script
  activation and clearing.
- [Bundle size] → The library chunk is only fetched when consent is on, about 20 kB gzipped.
- [jsdom doesn't run scripts, so tests can't prove execution] → The tests check that the library
  replaced the `text/plain` script with an executable one, and that nothing is replaced before
  consent.
- [Banner over the footer or the media mini player] → Box layout in the bottom-right corner, not a
  full-width bar. The visual check covers the narrow viewport and the mini player.
- [An operator runs a cookie-setting tool without consent, so the cookieless-analytics notice is
  wrong] → The README says that a snippet without consent must not set cookies or similar
  identifiers, and that tools such as GA or Matomo with cookies need `BOARDLY_CONSENT_ANALYTICS`.
- [Legal adequacy of the default texts] → The texts name the service and link to the privacy policy,
  but the content of the privacy policy stays the operator's job. The README says so.

## Migration Plan

Existing instances show the cookie notice after the update, which visitors can close. Otherwise,
nothing changes until `BOARDLY_CONSENT_ANALYTICS` is set. An operator who turns it on marks
the snippet's scripts, makes sure a privacy policy is configured, and restarts. Rollback: unset the
variable and restart. Stored `boardly_consent` cookies are then ignored and expire on their own.
