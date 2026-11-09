# Spec Delta

## MODIFIED Requirements

### Requirement: Media widget settings
The widget kind `media` ("Media" / "Medien") SHALL have, besides the common width, an ordered list of media sources (1–50), an optional title (at most 200 characters after trimming; empty means none), a layout (`list` or `tiles`, shown as "List" / "Liste" and "Tiles" / "Kacheln"; default `list`), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off). The settings dialog SHALL offer the owner's media sources as a search field over their titles with the placeholder "Add a station" / "Sender hinzufügen". Each result shows the source's icon and title. Chosen sources are listed below it in the widget's order, can be reordered by dragging (also with the keyboard), and can be removed. When the owner has no media sources, the dialog SHALL say "Your media library is empty." / "Deine Mediathek ist leer." with a link to `/media`. The dialog can't be confirmed without at least one source. Next to "Show to other viewers", the dialog SHALL say that visitors can then play these stations. A layout with a media widget without sources, with more than 50 sources, with a source ID longer than an ID can be, with a title longer than 200 characters, or with a layout or size outside these values is rejected with 400 and `invalid_layout`. The sources don't have to exist when the layout is saved.

#### Scenario: Add a media widget
- **WHEN** Ann picks "Media" in the widget picker, adds "FM4" and "Jazz Radio" and confirms
- **THEN** a media widget with these two stations, in this order, layout "List", size M and "Show to other viewers" off is added to the draft

#### Scenario: No source chosen
- **WHEN** Ann opens the settings of a new media widget and hasn't chosen a station
- **THEN** the dialog can't be confirmed

#### Scenario: Empty library
- **WHEN** Ann has no media sources and opens the settings of a new media widget
- **THEN** the dialog says "Your media library is empty." with a link to `/media`

#### Scenario: Invalid layout value
- **WHEN** a layout is saved with a media widget whose layout is `grid`
- **THEN** the server responds 400 with `invalid_layout`

### Requirement: Media widget on public dashboards
For anyone other than the owner, a media widget with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely, together with its sources. With the flag on, visitors get the widget with its sources and can play them like the owner. A visitor's response contains, for each existing source, only its ID, kind, title, stream URL and icon, and nothing else from the owner's media library: no other sources, positions or times.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a media widget whose "Show to other viewers" is off
- **THEN** the response contains no trace of the widget or its stations

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor activates "FM4" in the widget
- **THEN** the player plays "FM4" for the visitor

#### Scenario: Only what is shown
- **WHEN** Ann has five stations and the widget shows two of them
- **THEN** the visitor's response contains those two stations and nothing about the other three
