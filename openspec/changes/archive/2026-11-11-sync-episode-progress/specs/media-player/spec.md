## MODIFIED Requirements

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
