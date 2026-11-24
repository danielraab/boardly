## ADDED Requirements

### Requirement: Paused feeds in the widget
A widget showing "Latest" SHALL leave out the items of feeds the owner has paused, both with "All my feeds" and with chosen feeds, for everyone. "Read later", "Liked" and "Continue listening" keep the items of paused feeds. When some of a "Latest" widget's chosen feeds still exist and all of them are paused, the owner SHALL see "These feeds are paused" / "Diese Feeds sind pausiert", and visitors don't get the widget.

#### Scenario: One chosen feed paused
- **WHEN** a "Latest" widget has "Heise" and "Podcast Eins" and Ann pauses "Heise"
- **THEN** the widget shows only the items of "Podcast Eins"

#### Scenario: All my feeds
- **WHEN** a "Latest" widget shows "All my feeds" and Ann pauses "Heise"
- **THEN** the widget shows the items of her other feeds and none of "Heise"

#### Scenario: All chosen feeds paused
- **WHEN** a "Latest" widget's only chosen feed is "Heise" and Ann pauses it
- **THEN** Ann sees "These feeds are paused", and visitors of the public dashboard don't see the widget

#### Scenario: Removed and paused
- **WHEN** a "Latest" widget has "Heise" and "Podcast Eins", Ann removes "Podcast Eins" and pauses "Heise"
- **THEN** Ann sees "These feeds are paused"

#### Scenario: Saved items of a paused feed
- **WHEN** a "Liked" widget has "Heise", Ann liked two of its items and pauses "Heise"
- **THEN** the widget still shows the two items, for Ann and for visitors when the widget is public

#### Scenario: Resume
- **WHEN** Ann resumes "Heise"
- **THEN** the widget shows its items again without a change to the widget's settings

### Requirement: Paused feeds in the widget settings
The feed widget's settings dialog SHALL list paused feeds in its feed search with the label "Paused" / "Pausiert", and they can still be chosen. Choosing a paused feed doesn't resume it.

#### Scenario: Paused feed in the search
- **WHEN** Ann has paused "Heise" and searches for "Hei" in the settings of a feed widget
- **THEN** "Heise" is listed with "Paused" and can be chosen
