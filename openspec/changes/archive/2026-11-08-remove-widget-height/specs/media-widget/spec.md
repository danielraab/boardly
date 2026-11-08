# Spec Delta

## MODIFIED Requirements

### Requirement: Media widget settings
The widget kind `media` ("Media" / "Medien") SHALL have, besides the common width, an ordered list of media sources (1–50), an optional title (at most 200 characters after trimming; empty means none), a layout (`list` or `tiles`, shown as "List" / "Liste" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). The settings dialog SHALL offer the owner's media sources as a search field over their titles with the placeholder "Add a station" / "Sender hinzufügen". Each result shows the source's icon and title. Chosen sources are listed below it in the widget's order, can be reordered by dragging (also with the keyboard), and can be removed. When the owner has no media sources, the dialog SHALL say "Your media library is empty." / "Deine Mediathek ist leer." with a link to `/media`. The dialog can't be confirmed without at least one source. Next to "Show on public dashboard", the dialog SHALL say that visitors can then play these stations. A layout with a media widget without sources, with more than 50 sources, with a source ID longer than an ID can be, with a title longer than 200 characters, or with a layout or size outside these values is rejected with 400 and `invalid_layout`. The sources don't have to exist when the layout is saved.

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

### Requirement: Media widget rendering
A media widget SHALL show its title, if any, and its sources in the widget's order. With the layout `list`, each source is a row with its icon (or a radio icon) and its title. With `tiles`, each source is a square tile with its icon large and its title below. Each row or tile is a button. Activating it plays that source in the media player (see the `media-player` capability). Activating the source that is playing pauses it. The source that is loaded in the player SHALL be marked as playing (or paused), with an accessible state. The size sets the text and icon size on the same S/M/L scale as the other widgets. In edit mode, the buttons SHALL NOT start playback.

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

#### Scenario: Station changed
- **WHEN** Ann renames "FM4" to "FM4 Wien" on `/media` and opens the dashboard
- **THEN** the widget shows "FM4 Wien"
