# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- A default folder for new links, chosen in Settings → Profile and stored on the account. The
  "Save link" page (share target and "Add link" shortcut) starts in it on every device.

### Changed

- The "Save link" page no longer remembers the last used folder in the browser; it starts in the
  default folder, or the root folder when none is set.

## [0.1.0] - 2026-10-25

### Added

- Favourites: a tree of folders and links with tags, managed on the Favourites page. Links get
  their favicon fetched automatically, and an image store keeps favicons and uploaded images.
- Links have a description and a preview image, taken from the linked page (`og:description`,
  `og:image`) and shown by the Karakeep and Linkwarden apps. The apps can edit the description, and
  the Linkwarden extension's screenshot upload now becomes the link's preview.
- The edit dialog of a link shows and edits its description, shows its preview image with a
  "Remove preview" action, and has a button that loads the site's favicon again.
- A details dialog for links on dashboards shows the preview image, icon, URL, description and tags,
  with Open and Edit actions. The dashboard owner opens it from an info button on link, link list
  and favourite explorer widgets.
- Preview images are converted into JPEGs of at most 800 pixels wide. New settings:
  `BOARDLY_PREVIEW_MAX_SIZE_MB`, `BOARDLY_PREVIEW_MAX_PIXELS` and `BOARDLY_PREVIEW_WIDTH`.
- Dashboards: create, rename and delete dashboards, address them by user slug, pick one as the
  start page and make it public. An actions menu on each dashboard opens its settings.
- Dashboard editor with sections and widgets: favourite explorer (with an "Add favourite" button),
  link list for a favourites folder, and single link tiles with an icon chooser.
- App tokens, bound to one favourites folder, for connected apps. The "Connected apps" settings tab
  shows each app's logo.
- Karakeep-compatible API at `/api/karakeep` and Linkwarden-compatible API at `/api/linkwarden`, so
  their browser extensions and mobile apps can save and browse links.
- Installable web app (PWA) with an app icon, a share target, launcher shortcuts and a "Save link"
  page, also reachable from an "Add link" button in the header.
- One-time sign-in codes in the magic link email, for signing in from another browser or the
  installed app.
- Theme switcher with seven color themes.
- The browser tab shows the current page's title.
- SMTP login and sender settings: `BOARDLY_SMTP_USERNAME`, `BOARDLY_SMTP_PASSWORD` and
  `BOARDLY_SMTP_FROM`.

### Changed

- **Breaking:** the project is renamed from Famodash to Boardly. Environment variables, the database
  file name, the session cookie and the app token prefix change. See "Upgrading from Famodash" in
  the README.
- Buttons are more compact and share one style.
- Favicons are no longer refreshed after 30 days, and missing favicons are no longer fetched when
  the server starts. Use the refresh button in the link's edit dialog instead.
- Favicon fetches and other background work are kept in the database and continue after a restart.

## [0.0.1] - 2026-10-20

First release.

### Added

- Project foundation: Go backend serving a Vite frontend, app shell, landing page, home page and i18n.
- Passwordless sign-in with magic links.
- Passkey (WebAuthn) sign-in, plus registering and managing passkeys on the user profile.
- Container image with a built-in health check command.
- GitHub workflows for CI, i18n checks and publishing the image to GHCR on version tags.

[Unreleased]: https://github.com/danielraab/boardly/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/danielraab/boardly/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/danielraab/boardly/releases/tag/v0.0.1
