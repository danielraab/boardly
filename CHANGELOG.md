# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Links have a description and a preview image, taken from the linked page (`og:description`,
  `og:image`) and shown by the Karakeep and Linkwarden apps. The apps can edit the description, and
  the Linkwarden extension's screenshot upload now becomes the link's preview.
- Preview images are converted into JPEGs of at most 800 pixels wide. New settings:
  `BOARDLY_PREVIEW_MAX_SIZE_MB`, `BOARDLY_PREVIEW_MAX_PIXELS` and `BOARDLY_PREVIEW_WIDTH`.
- A refresh button on each link of the Favourites page loads the site's favicon again.

### Changed

- Favicons are no longer refreshed after 30 days, and missing favicons are no longer fetched when
  the server starts. Use the refresh button instead.
- Favicon fetches and other background work are kept in the database and continue after a restart.

## [0.0.1] - 2026-10-20

First release.

### Added

- Project foundation: Go backend serving a Vite frontend, app shell, landing page, home page and i18n.
- Passwordless sign-in with magic links.
- Passkey (WebAuthn) sign-in, plus registering and managing passkeys on the user profile.
- Container image with a built-in health check command.
- GitHub workflows for CI, i18n checks and publishing the image to GHCR on version tags.

[0.0.1]: https://github.com/danielraab/dashnoli/releases/tag/v0.0.1
