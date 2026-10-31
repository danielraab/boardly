# Spec Delta

## Purpose

A mini player in the app shell that plays one internet radio stream at a time and keeps playing while the user moves between Boardly's pages, for the owner and for visitors of public dashboards.

## ADDED Requirements

### Requirement: One player for the whole app
The app SHALL have exactly one media player, which the app shell owns. Starting a stream from anywhere, such as the media page or a media widget, SHALL play it in this player. Starting another stream replaces the current one, so at most one stream plays at a time. The player plays the stream URL directly in the browser. Audio never goes through Boardly's server.

#### Scenario: Switch station
- **WHEN** "FM4" is playing and Ann starts "Jazz Radio" from a widget
- **THEN** "FM4" stops and "Jazz Radio" plays

#### Scenario: Direct playback
- **WHEN** Ann plays "FM4"
- **THEN** the browser loads the station's stream URL and Boardly's server receives no audio request

### Requirement: Playback across navigation
Playback SHALL continue without interruption when the user moves to another page of the app, including dashboards, the media page, notes, favourites and settings, and when a dashboard enters or leaves edit mode. Reloading the page or opening Boardly in a new tab SHALL NOT start playback by itself. Signing out SHALL stop playback and close the player.

#### Scenario: Navigate while playing
- **WHEN** "FM4" is playing on `/ann/home` and Ann opens `/notes`
- **THEN** "FM4" keeps playing and the player is still shown

#### Scenario: Reload
- **WHEN** "FM4" is playing and Ann reloads the page
- **THEN** nothing plays and the player is closed

#### Scenario: Sign out
- **WHEN** "FM4" is playing and Ann signs out
- **THEN** playback stops and the player is closed

### Requirement: Mini player bar
While a stream is loaded, the app shell SHALL show a bar fixed to the bottom of the viewport with the station's icon (or a radio icon), its title, a play/pause button ("Play" / "Abspielen", "Pause" / "Pause"), a volume control ("Volume" / "Lautstärke") with a mute button ("Mute" / "Stumm", "Unmute" / "Ton an"), and a close button ("Close player" / "Player schließen") that stops playback and hides the bar. When no stream is loaded, the bar is not shown. While the bar is shown, the page content SHALL get enough space at the bottom that the bar covers none of it. On viewports narrower than 480 px, the bar SHALL leave out the volume control and keep the icon, title, play/pause and close. All controls work with the keyboard and have accessible names.

#### Scenario: Bar while playing
- **WHEN** Ann plays "FM4"
- **THEN** the bar at the bottom shows the FM4 icon, "FM4", a pause button, the volume control and the close button

#### Scenario: Pause and resume
- **WHEN** Ann activates "Pause" and then "Play"
- **THEN** the stream stops and then plays again from the live position

#### Scenario: Close
- **WHEN** Ann activates "Close player"
- **THEN** playback stops and the bar disappears

#### Scenario: Content not covered
- **WHEN** the bar is shown and Ann scrolls to the end of a long dashboard
- **THEN** the last widget is fully visible above the bar

### Requirement: Volume memory
The player SHALL remember the volume and the muted state in the browser and use them for every stream, also after a reload. When the browser doesn't allow storing them, the player SHALL use full volume, not muted, and still work.

#### Scenario: Volume kept
- **WHEN** Ann sets the volume to 40 %, reloads the page and plays a station
- **THEN** the volume is 40 %

### Requirement: Loading and errors
While a stream is connecting or buffering, the bar SHALL show a loading state instead of the play/pause icon. When the stream can't be played (network error, unsupported format, stream ended), the player SHALL stop, the bar SHALL show "This station can't be played right now" / "Dieser Sender kann gerade nicht abgespielt werden" and offer "Try again" / "Erneut versuchen", which starts the same stream again. The bar stays until the user closes it or starts another station.

#### Scenario: Dead stream
- **WHEN** Ann plays a station whose server answers 404
- **THEN** the bar shows "This station can't be played right now" and "Try again"

#### Scenario: Retry
- **WHEN** the stream is reachable again and Ann activates "Try again"
- **THEN** the station plays

### Requirement: HLS streams
The player SHALL play HLS streams (`.m3u8`) in every browser the app supports, natively where the browser can and otherwise through a script that is loaded only when an HLS stream is first played.

#### Scenario: HLS in a browser without native support
- **WHEN** Ann plays a station whose URL is an HLS playlist in a browser without native HLS support
- **THEN** the station plays

### Requirement: Operating system media controls
Where the browser supports it, the player SHALL tell the operating system the station's title, "Boardly" (or the custom app title) as the artist, and the station's icon as artwork, and SHALL react to the system's play, pause and stop controls, such as headset buttons, the lock screen and media keys. Stop SHALL work like the close button.

#### Scenario: Lock screen
- **WHEN** Ann plays "FM4" in the installed app on her phone and locks the screen
- **THEN** the lock screen shows "FM4" with its icon and a pause control that pauses the stream

### Requirement: Player for visitors
The player SHALL work for visitors who aren't signed in, when they start a station from a media widget on a public dashboard. It behaves as for the owner, except that there is nothing to sign out of.

#### Scenario: Visitor plays a station
- **WHEN** a signed-out visitor activates "FM4" in a media widget on Ann's public dashboard
- **THEN** the player bar appears and "FM4" plays
