# media-player Specification

## Purpose
A mini player in the app shell that plays one internet radio stream at a time and keeps playing while the user moves between Boardly's pages, for the owner and for visitors of public dashboards.

## Requirements

### Requirement: One player for the whole app
The app SHALL have exactly one media player, which the app shell owns. Starting a stream from anywhere, such as the media page or a media widget, SHALL play it in this player. Starting another stream replaces the current one, so at most one stream plays at a time. The player plays the stream URL directly in the browser. Audio never goes through Boardly's server. The player and the embed dock SHALL exclude each other: opening an embed stops whatever the player plays and closes the mini player bar, and starting a stream or an episode closes the embed dock.

#### Scenario: Switch station
- **WHEN** "FM4" is playing and Ann starts "Jazz Radio" from a widget
- **THEN** "FM4" stops and "Jazz Radio" plays

#### Scenario: Direct playback
- **WHEN** Ann plays "FM4"
- **THEN** the browser loads the station's stream URL and Boardly's server receives no audio request

#### Scenario: Embed replaces a station
- **WHEN** "FM4" is playing and Ann opens a Spotify album
- **THEN** "FM4" stops, the mini player bar closes and the embed dock shows the album

#### Scenario: Station replaces an embed
- **WHEN** a Spotify album is open and Ann plays "FM4"
- **THEN** the embed dock closes, which stops the album, and "FM4" plays

#### Scenario: Episode replaces an embed
- **WHEN** a Spotify album is open and Ann plays a podcast episode
- **THEN** the embed dock closes and the episode plays

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
While a stream is loaded, the app shell SHALL show a bar fixed to the bottom of the viewport with the station's icon (or a radio icon), its title, a play/pause button ("Play" / "Abspielen", "Pause" / "Pause"), a volume control ("Volume" / "Lautstärke") with a mute button ("Mute" / "Stumm", "Unmute" / "Ton an"), and a close button ("Close player" / "Player schließen") that stops playback and hides the bar. When no stream is loaded, the bar is not shown. While the bar is shown, the page content and the global footer (see the `app-shell` capability) SHALL get enough space at the bottom that the bar covers none of them. On viewports narrower than 480 px, the bar SHALL leave out the volume control and keep the icon, title, play/pause and close. All controls work with the keyboard and have accessible names.

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

#### Scenario: Footer not covered
- **WHEN** the bar is shown and Ann scrolls to the end of a page
- **THEN** the global footer with its links is fully visible above the bar

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

### Requirement: On-demand episodes
The player SHALL also play podcast episodes from the `feeds` capability, in the same single player, so that starting an episode replaces whatever plays and starting a stream replaces an episode. The audio loads directly from the episode's URL in the browser and never goes through Boardly's server. For an episode, pausing SHALL keep the position, and resuming SHALL continue from it without loading the audio again. Live streams keep resuming at the live position.

#### Scenario: Pause and resume an episode
- **WHEN** an episode is at 10:00 and Ann activates "Pause" and then "Play"
- **THEN** playback continues at 10:00

#### Scenario: Episode replaces a station
- **WHEN** "FM4" is playing and Ann plays an episode
- **THEN** "FM4" stops and the episode plays

#### Scenario: Direct playback of an episode
- **WHEN** Ann plays an episode
- **THEN** the browser loads the audio from the episode's URL and Boardly's server receives no audio request

### Requirement: Seeking in an episode
While an episode is loaded, the mini player bar SHALL show the elapsed time, the total time and a position slider ("Position" / "Position") that works with the keyboard and has an accessible value. On viewports narrower than 480 px, the bar SHALL keep the slider and leave out the time labels. While a stream is loaded, the bar SHALL NOT show any of these. Where the browser supports it, the player SHALL also react to the operating system's seek controls and report the position to it.

#### Scenario: Seek
- **WHEN** Ann moves the slider of an episode to 30:00
- **THEN** playback continues at 30:00

#### Scenario: Stream has no slider
- **WHEN** Ann plays "FM4"
- **THEN** the bar shows no position slider

### Requirement: Remembered episode position
For a signed-in user, the player SHALL save the position of an episode to the user's listen record on the server (see the `feeds` capability) when playback starts, at least every 15 seconds while it plays, when it pauses, after seeking, when another source replaces it, when the player is closed and when the page is hidden or left. It SHALL continue from the record's position when the episode is started again, also after a reload and on another device, unless the record is finished, in which case it starts at the beginning. When an episode plays to its end, the player SHALL mark the record as finished. A save that fails SHALL NOT interrupt playback and isn't shown. For a visitor who isn't signed in, and for an episode the user can't see, the player SHALL store nothing, neither on the server nor in the browser: the episode starts at the beginning when it is started again after a reload. Positions that earlier versions stored in the browser SHALL be deleted from the browser and not used.

#### Scenario: Continue after a reload
- **WHEN** Ann stops an episode at 25:00, reloads the page and plays it again
- **THEN** it continues at 25:00

#### Scenario: Continue on another device
- **WHEN** Ann stops an episode at 25:00 on her laptop and plays it on her phone
- **THEN** it continues at 25:00

#### Scenario: Reload doesn't start playback
- **WHEN** an episode is playing and Ann reloads the page
- **THEN** nothing plays and the player is closed

#### Scenario: Finished episode starts over
- **WHEN** Ann finished an episode and plays it again
- **THEN** it starts at the beginning and its record isn't finished any more

#### Scenario: Visitor stores nothing
- **WHEN** a signed-out visitor plays an episode on a public dashboard to 10:00, reloads and plays it again
- **THEN** it starts at the beginning, and the browser's storage holds no position

#### Scenario: Offline save
- **WHEN** the network is down while Ann listens to an episode
- **THEN** playback goes on without an error message

### Requirement: Finished episode
When an episode reaches its end, the player SHALL stop and the bar SHALL stay with a "Play" / "Abspielen" button that starts the episode from the beginning. This SHALL NOT be shown as an error. When an episode can't be played, the bar SHALL show "This episode can't be played right now" / "Diese Episode kann gerade nicht abgespielt werden" and offer "Try again" / "Erneut versuchen".

#### Scenario: Episode ends
- **WHEN** an episode plays to its end
- **THEN** the bar shows "Play" and no error, and activating it starts the episode from the beginning

#### Scenario: Audio gone
- **WHEN** Ann plays an episode whose file answers 404
- **THEN** the bar shows "This episode can't be played right now" and "Try again"

### Requirement: Episode details for the operating system
For an episode, the player SHALL tell the operating system the episode's title and the feed's title as the artist, and otherwise behave as described in "Operating system media controls". The player SHALL work for an episode started by a visitor from a feed widget on a public dashboard.

#### Scenario: Lock screen
- **WHEN** Ann plays an episode of "Podcast Eins" in the installed app and locks the screen
- **THEN** the lock screen shows the episode's title and "Podcast Eins" with a pause control

### Requirement: Embed dock
The app shell SHALL have one embed dock, which shows at most one embed, in a plain iframe with the source's `embedUrl`. The iframe SHALL be created only after the user opened the embed, and Boardly's page SHALL NOT load any script of the provider. The dock shows the source's title and a close button ("Close player" / "Player schließen"), which removes the iframe and hides the dock. Apart from the link to withdraw a stored consent (see "Consent for embeds"), the dock has no other Boardly controls: play, pause, seeking and volume are the provider's, inside its player. The dock SHALL NOT be collapsed or hidden while an embed is open.

On screens from the `sm` breakpoint up, the dock SHALL float in the bottom-right corner, about 360 px wide, with a player 352 px high for the shape `audio`. Below the `sm` breakpoint, the dock SHALL span the full width at the bottom, with a player 152 px high for the shape `audio`. While the dock is shown, the page content and the global footer SHALL get enough space at the bottom that, scrolled to the end, the dock covers none of them.

The embed SHALL stay open, without its iframe being reloaded, when the user moves to another page of the app and when a dashboard enters or leaves edit mode. Reloading the page or opening Boardly in a new tab SHALL NOT open an embed. Signing out SHALL close the dock. A renamed embed SHALL show its new title without the iframe being reloaded, and an embed whose link changed SHALL load the new player.

#### Scenario: Open an embed
- **WHEN** Ann opens the album "Global Warming" and has already confirmed the notice for Spotify
- **THEN** the dock appears with the title "Global Warming" and the player of the album

#### Scenario: Nothing loaded before opening
- **WHEN** Ann opens a dashboard with a media widget that contains a Spotify album and doesn't activate it
- **THEN** the browser makes no request to Spotify

#### Scenario: Navigate with an open embed
- **WHEN** an album is open on `/ann/home` and Ann opens `/notes`
- **THEN** the dock is still shown and its iframe isn't reloaded

#### Scenario: Close
- **WHEN** Ann activates "Close player" in the dock
- **THEN** the iframe is removed and the dock disappears

#### Scenario: Reload
- **WHEN** an album is open and Ann reloads the page
- **THEN** the dock is closed

#### Scenario: Sign out
- **WHEN** an album is open and Ann signs out
- **THEN** the dock is closed

#### Scenario: Floating on larger screens
- **WHEN** Ann opens an album on a screen at least as wide as the `sm` breakpoint
- **THEN** the dock floats in the bottom-right corner, about 360 px wide, with a 352 px high player

#### Scenario: Full width on a phone
- **WHEN** Ann opens an album on a screen narrower than the `sm` breakpoint
- **THEN** the dock spans the full width at the bottom with a 152 px high player

#### Scenario: Content not covered
- **WHEN** the dock is shown and Ann scrolls to the end of a long dashboard
- **THEN** the last widget and the global footer are fully visible above the dock

### Requirement: Consent for embeds
Until the user has agreed for the provider, opening an embed SHALL ask before anything is loaded from the provider. The dialog has the title "Open the Spotify player?" / "Spotify-Player öffnen?", the text "This connects your browser to Spotify, which can set cookies and see your IP address. Without a Spotify login in this browser, it plays previews only." / "Dabei verbindet sich dein Browser mit Spotify. Der Anbieter kann Cookies setzen und deine IP-Adresse sehen. Ohne Spotify-Anmeldung in diesem Browser spielt der Player nur Vorschauen.", a checkbox "Always open Spotify in this browser" / "Spotify in diesem Browser immer öffnen", which is unchecked at first, and the buttons "Open player" / "Player öffnen" and "Cancel" / "Abbrechen".

Confirming without the checkbox SHALL open the embed and allow embeds of that provider until the page is reloaded, so later embeds on the same page open without asking. With the checkbox, it SHALL also store the consent for that provider in the browser, so its embeds open without asking after a reload too. Consent is kept per provider. Cancelling SHALL change nothing, and whatever plays keeps playing. When the browser can't store the consent, confirming SHALL still open the embed and allow the provider for the page.

While an embed is open and consent for its provider is stored, the dock SHALL show a "Don't open Spotify automatically" / "Spotify nicht automatisch öffnen" link under the title. Activating the link SHALL remove the stored consent for that provider. The player stays, the link is replaced by "Spotify will ask again." / "Spotify wird wieder nachfragen.", and the next embed of that provider after a reload asks again.

#### Scenario: First use
- **WHEN** Ann opens a Spotify album for the first time in this browser
- **THEN** the dialog is shown, and the browser has made no request to Spotify

#### Scenario: Confirm for the page
- **WHEN** Ann confirms the dialog without the checkbox and opens another Spotify embed
- **THEN** both open without a second dialog, and after a reload the dialog is shown again

#### Scenario: Always open
- **WHEN** Ann checks "Always open Spotify in this browser", confirms the dialog and reloads the page
- **THEN** the next Spotify embed opens without the dialog, with the link "Don't open Spotify automatically" in the dock

#### Scenario: Withdraw
- **WHEN** Ann activates "Don't open Spotify automatically" in the dock
- **THEN** the player stays, the link is replaced by "Spotify will ask again." and after a reload the next Spotify embed shows the dialog

#### Scenario: Cancel
- **WHEN** "FM4" is playing, the dialog is shown and Ann activates "Cancel"
- **THEN** no dock opens and "FM4" keeps playing

#### Scenario: Storage not available
- **WHEN** the browser doesn't allow storing the consent and Ann checks the checkbox and confirms the dialog
- **THEN** the dock opens, the next Spotify embed on the page opens without the dialog, and after a reload the dialog is shown again
