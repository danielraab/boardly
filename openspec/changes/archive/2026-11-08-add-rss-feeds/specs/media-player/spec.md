# Spec Delta

## ADDED Requirements

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
The player SHALL remember the position of an episode in the browser at least every 10 seconds while it plays and when it pauses, and SHALL continue from there when the episode is started again, also after a reload. The position SHALL be forgotten when the episode finishes. The player SHALL keep the positions of at most the 200 most recently played episodes. When the browser doesn't allow storing them, the player SHALL start at the beginning and still work.

#### Scenario: Continue after a reload
- **WHEN** Ann stops an episode at 25:00, reloads the page and plays it again
- **THEN** it continues at 25:00

#### Scenario: Reload doesn't start playback
- **WHEN** an episode is playing and Ann reloads the page
- **THEN** nothing plays and the player is closed

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
