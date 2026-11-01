# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-11-01

### Added

- Attachments in notes: files can be added with the "Add attachment" button in the editor toolbar,
  or pasted or dropped into the content. Images, audio and video are shown inline, other files as a
  file card, and the paperclip button lists the note's files. Clicking an image opens an image
  viewer, also on the note cards of `/notes`.
- An Attachments page (`/attachments`, linked from the user menu) with every attachment and the
  notes that use them, filters, Keep and Delete. Attachments that no note uses are marked for
  deletion after 24 hours and deleted after `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` (default 30)
  days, unless they are kept or used again.
- `BOARDLY_ATTACHMENT_MAX_SIZE_MB` (default 5), `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` and
  `BOARDLY_ATTACHMENT_TYPES` (default: images except SVG, and PDF) configure attachments.
  `.env.example` lists every known type.
- Locations in notes: the "Add location" button in the editor toolbar adds a place, set by
  coordinates, the device's position or a click on the map, with an optional label. It is shown as
  a chip that opens a map. Maps load only when one is opened, from the style at the new
  `BOARDLY_MAP_STYLE_URL` (default: OpenFreeMap).
- Attachments in the Memos API: photos and files added in MoeMemos are uploaded as attachments and
  added at the end of the note.
- Locations in the Memos API: a memo's location is the location at the end of the note, before its
  attachments.
- Attachments in the Karakeep API: the Karakeep apps can upload images and PDFs, which become notes
  with the file as their content, and a note that holds only one image or PDF is shown in the apps
  as that file.
- Floccus bookmark sync, in Linkwarden or Karakeep mode. The Linkwarden API gets
  `GET` and `PUT /api/v1/collections/{id}` (rename and move a collection), and the Karakeep API gets
  the REST endpoints under `/api/karakeep/api/v1/` that Floccus uses. See "Floccus bookmark sync"
  in the README.
- A Media page (`/media`, linked from the user menu) with a library of internet radio stations.
  Stations are found through a search of radio-browser.info, which the browser calls directly, or
  added by stream URL. Only https streams can be played; the server checks a station's URL when it
  is saved, resolves `.pls` and `.m3u` playlists, and tries `https://` for an `http://` URL. Audio
  never goes through the server.
- A player bar at the bottom of every page that keeps playing while you move around the app, with
  play/pause, volume and mute, and support for the device's media controls.
- A "Media" dashboard widget, in a new "Media" group of the widget picker, that plays stations of the
  library. It can be shown on public dashboards, where visitors can play the stations.
- A link can be deleted from its edit dialog, with the trash button next to Cancel and Save.
- The note pickers of the note and selected notes widgets show each note's category path, and mark
  archived notes. The note widget's settings search notes by title or category.

### Changed

- **Breaking:** deleting a collection through the Linkwarden API (`DELETE /api/v1/collections/{id}`,
  also used by the Linkwarden extension and app) now deletes its links and subfolders too, as
  Linkwarden does. Before, they were moved to the parent folder. Deleted bookmarks can't be
  restored, except from a backup.
- **Breaking:** the Linkwarden API lists the folders directly in the key's folder as top-level
  collections (`parentId` `null`), next to the key's own collection, as Linkwarden lists
  "Unorganized". The Linkwarden apps show them at the top level instead of inside the key's
  collection.
- **Breaking:** files at the end of a note are the memo's attachments in the Memos API and are no
  longer part of the memo's `content`. The same applies to a location at the end of a note.
- The landing page describes Boardly as an everyday helper, with a sketch of a dashboard and
  feature cards for dashboards, favourites, notes, connected apps and radio.

## [0.3.0] - 2026-10-29

### Added

- Notes on a new Notes page (`/notes`, linked from the user menu): a card grid with an optional
  title, icon and colour per note, and rich content written with Markdown shortcuts, including
  checkable todos. Notes are saved automatically and sorted into nested categories, managed in a
  side panel.
- Notes can be edited inline on their cards, and todos ticked off there. "Open note" opens the full
  editor.
- Dashboard widgets for notes, grouped under "Notes" in the widget picker: "Note" shows one note
  with its todos, "Recent notes" the newest notes, "Note list" the notes of a category and
  "Selected notes" a hand-picked list. Each can be shown on public dashboards. Rows look like note
  cards in the note's colour.
- Opening a note from a dashboard widget shows its editor in a modal and adds `?note=<id>` to the
  address, so Back closes it and the address can be reloaded or shared.
- Dashboard sections with a title can be collapsed. The state is remembered in the browser.
- GitHub links on the landing page and in the user menu.
- Notes can be pinned and archived from the note menu. Pinned notes come first and show a pin;
  archived notes are left out of the categories and shown under a new "Archived" entry. Note
  widgets still show archived notes, marked as archived.
- Memos-compatible API at `/api/memos` for the MoeMemos app, with a new "Memos" connected app whose
  key can be limited to a note category. Memos are notes in Markdown, with the title as a leading
  `# ` heading; pinned and archived map to the note's flags. Attachments aren't supported yet.
- The Karakeep apps see a note's archived flag as `archived` and its pinned flag as `favourited`,
  can set both and filter by them. The Karakeep import keeps archived text bookmarks archived.
- `BOARDLY_LOG_REQUESTS` chooses which HTTP requests are logged: `all` (default), `errors`
  (status 500 and higher) or `none`.
- The Karakeep import creates notes from text bookmarks, in a category chosen on the import form.
  Lists with text bookmarks become categories, the bookmark's tags become a `#tag` line, and notes
  that are already saved are skipped. The report counts notes and categories.
- The "Save link" page can save a note instead: shared text without a web address opens it as
  "Save note", and a switch changes between link and note.
- A default category for new notes, chosen in Settings → Profile, where notes from the share page
  start.
- `POST /api/notes` accepts `text`, plain text that the server turns into the note's content.
- Karakeep text bookmarks: the Karakeep apps save, list, search, edit and delete notes as text
  bookmarks with Markdown content. Each Karakeep key has a category for notes ("All notes" by
  default, also for existing keys); deleting that category revokes the key.
- Connected apps can be edited: the folder and the category of an active key can be changed
  without a new key (`PATCH /api/app-tokens/{id}`).

### Changed

- Karakeep text bookmarks are no longer counted as "not a link" in the import report.
- Each page now has a width that suits its content: dashboards and the notes overview use the full
  1536 px, favourites, the dashboards list and the landing page up to 1152 px, settings, a note and
  the not-found page up to 768 px. All narrower pages are centred.
- Successful health checks (`GET /api/health` answered with 200) are no longer logged.
- The settings tab bar scrolls sideways on narrow screens instead of wrapping its labels.

## [0.2.0] - 2026-10-27

### Added

- Bookmark import on Settings → Import / Export: upload a Karakeep JSON, Linkwarden JSON or browser
  (Netscape HTML) export. Folders are recreated, creation dates and order are kept, and a report
  lists what was added and skipped. Skipping links that are already saved is optional.
- Search on the Favourites page: filter the tree by title, URL, tag and folder path, with
  highlighted matches. A tag filter, expand/collapse all, folder link counts and `?folder=<id>`
  deep links to a folder.
- Bulk edit on the Favourites page: select links and folders, then add, remove or set tags, move
  them to a folder, or delete them with Undo.
- A "Manage tags" dialog to rename, merge and delete tags.
- "Recent favourites" dashboard widget, listing the newest links of a folder and its subfolders,
  limited by count and optionally by age.
- Link list widgets get an optional tag filter and a sort setting (favourites order, newest first,
  alphabetical).
- `BOARDLY_APP_TITLE` renames the instance (e.g. "Boardly Staging") in the web app, its manifest
  and the sign-in emails.
- `BOARDLY_ALLOW_REGISTRATION=false` stops sign-in from creating new accounts; unknown addresses
  get a "registration closed" email instead.
- `BOARDLY_ANALYTICS_SNIPPET` inserts an analytics script (Umami, Plausible, Matomo, …) into the
  web app's HTML without a new build.
- A default folder for new links, chosen in Settings → Profile and stored on the account. The
  "Save link" page (share target and "Add link" shortcut) starts in it on every device.

### Changed

- The "Save link" page no longer remembers the last used folder in the browser; it starts in the
  default folder, or the root folder when none is set.
- Pages are wider on large screens: the header and content now use up to 1536 px instead of
  1152 px, so dashboards get more room.
- Deeply nested folders scroll sideways in the Favourites tree and the favourite explorer widget
  instead of squeezing names.

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

[Unreleased]: https://github.com/danielraab/boardly/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/danielraab/boardly/compare/v0.3.0...v1.0.0
[0.3.0]: https://github.com/danielraab/boardly/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/danielraab/boardly/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/danielraab/boardly/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/danielraab/boardly/releases/tag/v0.0.1
