# Design

## Context

See `proposal.md` for the motivation and the `media-library`, `media-player`, `media-widget` and `data-import` deltas for the behaviour.

Today every source of the library plays through one `<audio>` element that `MediaPlayerProvider` owns in the app shell. `media_sources.kind` is free text with no constraint, and the store rejects anything but `stream` in `Create` and `InsertTx`. The app sends no Content-Security-Policy on its pages, so an iframe needs no header change. The media widget decides visibility per widget in its `Resolve` function.

## Goals / Non-Goals

**Goals:**
- A new provider only adds a registry entry, plus a display name in the frontend.
- Nothing from a provider reaches the browser before the user asks for it, and never for visitors of public dashboards.

**Non-Goals:**
- Showing what an embed plays, or controlling it from Boardly. That needs a provider script, which was ruled out because it would run with the rights of Boardly's page.
- Tracks, podcasts and artists of Spotify. They are later additions to the provider's accepted types.
- YouTube itself. The design leaves room for it, but this change adds only the `audio` shape.
- Consent management through the `analytics-consent` banner. The first-use notice is separate, and is per provider.

## Decisions

### Provider registry in the backend `media` package

A new `embed.go` holds a list of providers. Each has a name, a shape, a function that parses a raw link into a type and ID, and functions that give the canonical URL, the player address and the oEmbed address. `Parse` loops over the list.

The server is the only place that builds a player address. The browser never derives one from a pasted link, so an import file or a crafted request can't put an arbitrary page into the iframe.

*Alternative*: a registry in the frontend. Rejected, because link validation has to happen on the server for the API and for import anyway.

### Derive provider, player address and shape when reading

Only the canonical URL is stored, in the existing `url` column. `scanSource` adds `provider`, `embedUrl` and `shape` to sources of kind `embed`, with `omitempty` so that streams look as before. This needs no migration, and the export stays as it is. The existing "same URL" duplicate check of the import works because the URL is canonical.

*Alternative*: new columns. Rejected, because they would repeat what the URL already says and would have to be kept in step in export and import.

### One kind, `embed`, rather than `spotify`

The radio change planned for `embed`, and a provider-specific kind would need another kind and more store branches for YouTube. The provider follows from the URL.

### oEmbed behind a store function

`Store` gets a third injected function next to `Check` and `FetchLogo`, which fetches an item's title and cover. The real one uses the guarded transport that the radio directory already uses (`images.GuardedTransport`). Tests inject a fake. The cover goes through the existing `FetchLogo`, so it lands in the user's images and the browser loads it from Boardly.

The oEmbed call happens only when a title or an icon is missing. Spotify's docs don't list playlists for oEmbed, but a request for a playlist returned a title and an iframe URL, and the failure rule ("a source with a title is saved anyway") limits the damage if that changes.

### The dock is state of the player, rendered in the app shell

`PlayableSource.kind` gains `embed`, with optional `provider`, `embedUrl` and `shape`. `play(source)` branches on the kind: an embed stops the audio and becomes `current`, and a stream or an episode replaces it, which unmounts the iframe. `PlayerStatus` gains `open`.

Exclusion needs no knowledge of what plays inside the iframe, because removing the iframe stops it. `MiniPlayer` renders nothing for an embed, and a new `EmbedDock` next to it renders only for an embed. Both live inside `MediaPlayerProvider`, so the existing behaviour across routes, edit mode and sign-out applies. `toggle` and the Media Session code ignore embeds.

*Alternative*: an iframe inside the widget. Rejected: it stops on navigation, widgets can be narrower than the 280 px the player needs, a widget can list 50 sources, and the media page would need its own place for it. Rendering once in the shell is also less code.

The iframe gets a stable React key from `embedUrl`, so that a renamed source doesn't reload it and a changed link does.

### Plain iframe, no sandbox attribute

The iframe has `allow="autoplay; clipboard-write; encrypted-media; fullscreen; picture-in-picture"`, the list that Spotify's own oEmbed snippet uses. `encrypted-media` is needed for more than previews.

A cross-origin iframe is already isolated from Boardly's page by the same-origin policy. A `sandbox` attribute would add little and risks breaking the provider's login and playback, so it isn't set. The referrer policy stays at the browser default, which YouTube's embeds will need.

### Dock layout

A fixed element with `sm:` variants: from `sm` up, bottom-right and 360 px wide, and below that full width at the bottom. The player's height comes from `shape`, 352 or 152 px. `EmbedDock` reserves its height at the bottom of the page the way the mini player does, so the end of a page can be scrolled clear of the panel even on desktop. Changing the viewport width changes only the iframe's height, so it doesn't reload.

A later `video` shape would use a 16:9 box. The dock has no collapsed state, because YouTube requires its player to stay visible.

### First-use notice in the frontend

The notice is a dialog shown by a small `useOpenEmbed` helper that widgets and the media page share. The answer is stored per provider under one `localStorage` key, wrapped so that a failure just means asking again. Provider display names ("Spotify") live in a small frontend map, used with translated sentences that take the name as a parameter.

### Visitor filter in `Resolve`

`Resolve` drops sources of kind `embed` when `!viewer.IsOwner`, before the existing "no sources left" check, so a widget of only embeds is left out through the same path. `mediaItem` gets the three embed fields for owners. It's a filter on the server, so a visitor can't get an embed by changing the frontend.

The lookup also returns each widget's stored config, whose `sourceIds` would still name the embeds. A `VisitorConfig` empties that list for visitors, as the feed widget does with its feed IDs. The view plays from the resolved data, so visitors don't need the IDs.

## Risks / Trade-offs

- [Spotify plays only previews on phones or with third-party cookies blocked] → The notice says that a login is needed, and the proposal records this as a known limitation. Desktop Chrome and Firefox played full tracks in a manual test.
- [Boardly can't tell whether an embed is playing] → The UI says "open", never "playing", and offers only close.
- [Spotify changes its player address or oEmbed behaviour] → Both are confined to one registry entry. Saved sources keep working when oEmbed fails, since it's only used when adding.
- [The floating panel covers a corner of the page] → Space is reserved at the end of the page so everything can be scrolled clear of it. The panel closes with one click.
- [Compact height of 152 px isn't documented in what we read] → It's checked by hand on a phone during implementation, and it's a one-line constant if the player needs a different height.
- [Spotify sets cookies once the player opens] → Only after the notice is confirmed, never for visitors. The README tells operators to cover it in their privacy policy.

## Migration Plan

No migration. Rolling back leaves `embed` rows in `media_sources` that an older server still lists, but can't play, and that a user can delete from `/media`. Older servers' imports count them as invalid.
