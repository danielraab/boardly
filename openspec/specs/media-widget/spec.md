# media-widget Specification

## Purpose
A dashboard widget that shows chosen stations of the owner's media library as play buttons, which start the app's media player, for the owner and, if allowed, for visitors of public dashboards.

## Requirements

### Requirement: Media widget settings
The widget kind `media` ("Media" / "Medien") SHALL have, besides the common width, an ordered list of media sources (1–50), an optional title (at most 200 characters after trimming; empty means none), a layout (`list` or `tiles`, shown as "List" / "Liste" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). The settings dialog SHALL offer the owner's media sources as a search field over their titles with the placeholder "Add media" / "Medien hinzufügen". Each result shows the source's icon and title. Chosen sources are listed below it in the widget's order, can be reordered by dragging (also with the keyboard), and can be removed. When the owner has no media sources, the dialog SHALL say "Your media library is empty." / "Deine Mediathek ist leer." with a link to `/media`. The dialog can't be confirmed without at least one source. Next to "Show on public dashboard", the dialog SHALL say that visitors can then play these stations, and that Spotify and other embedded players are never shown on public dashboards: "Spotify and other embedded players are never shown on public dashboards." / "Spotify und andere eingebettete Player werden auf öffentlichen Dashboards nie angezeigt." A layout with a media widget without sources, with more than 50 sources, with a source ID longer than an ID can be, with a title longer than 200 characters, or with a layout or size outside these values is rejected with 400 and `invalid_layout`. The sources don't have to exist when the layout is saved.

#### Scenario: Add a media widget
- **WHEN** Ann picks "Media" in the widget picker, adds "FM4" and "Jazz Radio" and confirms
- **THEN** a media widget with these two stations, in this order, layout "List", size M and "Show on public dashboard" off is added to the draft

#### Scenario: No source chosen
- **WHEN** Ann opens the settings of a new media widget and hasn't chosen a station
- **THEN** the dialog can't be confirmed

#### Scenario: Empty library
- **WHEN** Ann has no media sources and opens the settings of a new media widget
- **THEN** the dialog says "Your media library is empty." with a link to `/media`

#### Scenario: Invalid layout value
- **WHEN** a layout is saved with a media widget whose layout is `grid`
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: Note about public dashboards
- **WHEN** Ann opens the settings of a media widget
- **THEN** next to "Show on public dashboard" the dialog says that embedded players are never shown on public dashboards

### Requirement: Media widget rendering
A media widget SHALL show its title, if any, and its sources in the widget's order. With the layout `list`, each source is a row with its icon (or a radio icon, for an embed a music icon) and its title. With `tiles`, each source is a square tile with its icon large and its title below. Each row or tile is a button. Activating a stream plays it in the media player (see the `media-player` capability), and activating the source that is playing pauses it. The source that is loaded in the player SHALL be marked as playing (or paused), with an accessible state. Activating an embed opens it in the embed dock, after the consent notice if that is due, and its accessible name is "Open <title>" / "<title> öffnen". The embed that is open SHALL be marked as open, with an accessible state, and activating it leaves the dock as it is. The size sets the text and icon size on the same S/M/L scale as the other widgets. In edit mode, the buttons SHALL NOT start playback or open an embed.

#### Scenario: Play from the widget
- **WHEN** Ann activates "FM4" in a media widget
- **THEN** the player plays "FM4" and "FM4" is marked as playing in the widget

#### Scenario: Pause from the widget
- **WHEN** "FM4" is playing and Ann activates "FM4" in the widget
- **THEN** playback pauses and "FM4" is marked as paused

#### Scenario: Same station in two widgets
- **WHEN** two widgets on the dashboard contain "FM4" and "FM4" is playing
- **THEN** both widgets mark "FM4" as playing

#### Scenario: Edit mode
- **WHEN** Ann activates "FM4" in a media widget while editing the dashboard
- **THEN** nothing starts playing

#### Scenario: Edit mode with an embed
- **WHEN** Ann activates a Spotify album in a media widget while editing the dashboard
- **THEN** no dock opens and no notice is shown

#### Scenario: Station changed
- **WHEN** Ann renames "FM4" to "FM4 Wien" on `/media` and opens the dashboard
- **THEN** the widget shows "FM4 Wien"

#### Scenario: Open an embed from the widget
- **WHEN** Ann activates the album "Global Warming" in a media widget
- **THEN** the embed dock shows the album and the album is marked as open in the widget, not as playing

#### Scenario: Activate an open embed
- **WHEN** "Global Warming" is open and Ann activates it in the widget
- **THEN** the dock stays open and its player isn't reloaded

### Requirement: Deleted media sources
A source in a media widget that no longer exists SHALL be left out of the widget for everyone. When none of a widget's sources exist, the widget SHALL show "These media sources no longer exist" / "Diese Medien existieren nicht mehr" to the owner, and its settings can still be opened. For visitors, the widget is left out. The layout can still be saved with sources that no longer exist.

#### Scenario: One station deleted
- **WHEN** a widget has "FM4" and "Jazz Radio" and Ann deletes "Jazz Radio" on `/media`
- **THEN** the widget shows only "FM4"

#### Scenario: All deleted
- **WHEN** all of a widget's stations are deleted
- **THEN** the owner sees "These media sources no longer exist", and visitors don't see the widget

### Requirement: Media widget on public dashboards
For anyone other than the owner, a media widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely, together with its sources. With the flag on, visitors get the widget with its streams and can play them like the owner. Sources of kind `embed` SHALL NOT be part of the response for anyone other than the owner, whatever the flag says. A widget with no stream left for a visitor SHALL be left out of the response entirely. A visitor's response contains, for each existing stream, only its ID, kind, title, stream URL and icon, and nothing else from the owner's media library: no other sources, positions or times.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a media widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its stations

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor activates "FM4" in the widget
- **THEN** the player plays "FM4" for the visitor

#### Scenario: Only what is shown
- **WHEN** Ann has five stations and the widget shows two of them
- **THEN** the visitor's response contains those two stations and nothing about the other three

#### Scenario: Embed left out for visitors
- **WHEN** the flag is on and a widget has the station "FM4" and a Spotify album
- **THEN** the visitor's response contains "FM4" and no trace of the album

#### Scenario: Only embeds
- **WHEN** the flag is on and a widget has only Spotify albums
- **THEN** the visitor's response contains no trace of the widget

#### Scenario: Another signed-in user
- **WHEN** Bob is signed in and opens Ann's public dashboard, whose widget has a Spotify album and the flag on
- **THEN** the response contains no trace of the album

#### Scenario: Owner keeps embeds
- **WHEN** Ann opens her own public dashboard, whose widget has a Spotify album
- **THEN** the response contains the album with its `provider`, `embedUrl` and `shape`
