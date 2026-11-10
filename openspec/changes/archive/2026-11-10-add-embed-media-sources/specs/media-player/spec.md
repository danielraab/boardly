## MODIFIED Requirements

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

## ADDED Requirements

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
