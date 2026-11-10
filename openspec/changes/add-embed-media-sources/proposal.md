# Proposal

## Why

Issue #25 asks for internet radio and, later, other media such as Spotify playlists. Radio shipped with a library and a media widget that left room for embeddable sources. Spotify, and YouTube after it, can't play through Boardly's `<audio>` element, but both offer a player that a plain iframe can show, without an SDK, an npm dependency or a provider script in Boardly's page.

## What Changes

- New media source kind `embed`, starting with **Spotify playlists and albums**. Embeds live in the existing library and media widget, and are added by pasting a link in a new "Embed" tab of the add dialog.
- A **provider registry** on the server turns a pasted link into a canonical URL and an iframe URL, and says how to fetch the title and cover (oEmbed) and which shape the player has. YouTube can follow as one more entry.
- A new **embed dock** in the app shell shows the provider's player in a plain iframe, created only after the user activates the source, and keeps playing across pages. It is a floating panel of about 360 px in the bottom-right corner on larger screens, and full width at the bottom in the provider's shorter layout on phones. Its only Boardly control is close.
- The dock and the audio player exclude each other: starting radio or a podcast removes the dock, and opening an embed stops the audio.
- Boardly can't see whether an embed is playing, so widgets and the media page mark it as **open** rather than playing or paused, and activating an open source leaves it open.
- **Embeds are never shown on public dashboards.** The server leaves them out for everyone but the owner, whatever "Show on public dashboard" says. A widget with only embeds disappears for visitors.
- A **consent notice** per provider says that opening the player connects to the provider, which can set cookies. It works like the map consent: confirming holds until the page is reloaded, and only a checkbox stores it in the browser, with a link in the dock to withdraw it. Nothing is loaded from the provider before the user confirms it.
- Texts that speak of stations where embeds now appear become general ("Add media" / "Medien hinzufügen").
- No Spotify library, no provider script and no new dependency. Boardly doesn't use Spotify's Web Playback SDK or iFrame API.
- Known limitation: full tracks need a Spotify login in the same browser, otherwise Spotify plays 30-second previews. This worked on Chrome and Firefox on desktop. Phones are untested and may only play previews.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `media-library`: the `embed` kind, the embed link check and provider registry, title and cover through oEmbed, the "Embed" tab, general wording, and managing embeds on `/media`.
- `media-player`: the embed dock, its exclusion with the audio player, and the consent notice.
- `media-widget`: embeds in the widget, the "open" mark, and leaving embeds out for visitors.
- `data-import`: imported embed sources are validated through the registry.

## Impact

- **Backend**: `backend/internal/media` (registry, oEmbed fetch, store), `backend/internal/widgets/media.go` (per-source filter for visitors), `backend/internal/httpapi/media.go` (new error codes), `backend/internal/dataimport` (invalid embeds counted as invalid).
- **Frontend**: `frontend/src/features/media` (types, player context, dock, notice, add and edit dialogs), `frontend/src/features/dashboards/widgets/media`, `AppShell`, en and de translations.
- **API**: `POST /api/media` and `PATCH /api/media/{id}` accept `embed` sources, and `title` is optional for them. Sources of kind `embed` also have `provider`, `embedUrl` and `shape`. New errors `embed_unsupported` and `embed_unreachable`. `backend/internal/httpapi/openapi/boardly.yaml` is updated.
- **Database**: no migration. `kind` is free text, and the provider follows from the stored URL.
- **Export and import**: the export already carries each source's kind, title, URL and icon, so the format and its version stay the same. Import checks embed links through the registry and counts ones it can't use as invalid. Older servers skip the unknown kind and count it as invalid, as they already do.
- **Privacy**: signed-in users who confirm the notice connect to Spotify, which sets cookies such as a one-year `sp_t`. Visitors of public dashboards never do. The README tells operators to mention this in their privacy policy.
- **External services**: the server calls `open.spotify.com/oembed` when an embed is added without a title or cover, sending only the link.
