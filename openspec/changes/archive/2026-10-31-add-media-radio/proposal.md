# Proposal

## Why

Boardly is the page people keep open all day, but it can't play anything. Issue #25 asks for internet radio and, later, other media such as Spotify playlists. Internet radio is plain HTTPS audio that browsers play natively, so it is a cheap first step. A media library with a persistent player gives later source kinds (embeds) a home without another redesign.

## What Changes

- New **Media** page (`/media`, linked from the user menu) where a signed-in user keeps an ordered list of media sources. This change adds one source kind, `stream` (an internet radio station): a title, an HTTPS stream URL and an optional icon.
- Adding a station works in two ways:
  - **Search**: the browser queries the public radio-browser.info directory directly. The dialog states that the search is sent to radio-browser.info, an external community service, and links to it. Stations without an HTTPS stream are shown as not playable.
  - **Paste**: the user enters a URL by hand.
- When a station is saved, the server checks its URL. It resolves `.pls` and `.m3u` playlist files to the stream inside them, tries the `https://` version once for an `http://` URL, and rejects URLs that aren't playable over HTTPS. The station's logo is fetched once and stored in Boardly's image storage. All fetches use the existing private-address protection.
- **Only HTTPS streams.** Boardly never proxies or relays audio, and shows no "now playing" titles.
- A **persistent mini player** at the bottom of every page plays one stream at a time. It keeps playing across navigation, has play/pause, volume (remembered per browser) and close, shows an error when a stream fails, and integrates with the operating system's media controls (Media Session).
- A new **Media** dashboard widget shows chosen stations as play buttons that start the mini player. It has a "Show on public dashboard" flag, so visitors of a public dashboard can play the stations too.
- The data model stores a source `kind`, so a later change can add `embed` sources (Spotify, YouTube, SoundCloud) without a migration redesign.

## Capabilities

### New Capabilities
- `media-library`: the `/media` page and its API. Media sources of a user, the `stream` kind, adding stations by radio-browser.info search or by pasting a URL, URL checks and logo fetch on save, renaming, choosing an icon, reordering and deleting.
- `media-player`: the persistent mini player in the app shell. One active stream, play/pause, volume, close, error state, Media Session integration, behaviour across navigation and sign-out.
- `media-widget`: the `media` dashboard widget. Its settings, rendering for the owner and for visitors of public dashboards, and deleted sources.

### Modified Capabilities
- `app-shell`: the signed-in user menu gets a "Media" link, and the page widths table gets the media page.
- `dashboard-editing`: the widget picker gets a third group, "Media", with the media widget.
- `images`: the upload cleanup keeps images that a media source uses as its icon, and the private-address protection also covers station URL checks and logo fetches.

## Impact

- **Backend**: a new migration (`media_sources` table), a new `internal/media` package (store, URL check, playlist parsing), new REST endpoints under `/api/media`, a new widget kind in `internal/widgets`, updated image cleanup query, and reuse of the guarded HTTP client from `internal/images`.
- **Frontend**: a new route `/media` and feature folder, a `MediaPlayerProvider` and mini player mounted in the root layout, a radio-browser.info client, a new widget kind and widget group, a user menu entry, en/de translations.
- **External dependency**: radio-browser.info, called only from the browser and only when a user searches. Boardly keeps working without it, and pasting URLs still works.
- **No new configuration.** The existing `BOARDLY_FETCH_ALLOW_PRIVATE` also applies to station checks.
- **Docs**: README gets a "Media" section, and CHANGELOG gets an entry.
