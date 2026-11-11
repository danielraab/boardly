## ADDED Requirements

### Requirement: Order of the continue listening widget
With "Continue listening", a feed widget SHALL show the owner's episodes in progress (see the `feeds` capability) of its feeds, sorted by when their listen record was last changed, newest first. "All my feeds" includes episodes of feeds the owner no longer subscribes to. An episode that is finished, marked as unplayed or removed from history SHALL leave the widget without a reload of the page, for the owner. For the owner, its rows show the progress described in "Listen progress on rows" of the `feeds` capability.

#### Scenario: Finished episode leaves the widget
- **WHEN** a "Continue listening" widget shows an episode and Ann plays it to its end
- **THEN** the episode disappears from the widget

#### Scenario: Order
- **WHEN** Ann listened to episode A yesterday and to episode B today
- **THEN** the widget lists B before A

## MODIFIED Requirements

### Requirement: Feed widget settings
The widget kind `feed` ("Feed" / "Feed") SHALL have, besides the common width, a choice of feeds, "Show" / "Anzeigen" with the values "Latest" / "Neueste" (`latest`, default), "Read later" / "Später lesen" (`read_later`) and "Liked" / "Gefällt mir" (`liked`) and "Continue listening" / "Weiterhören" (`in_progress`), a number of items ("Number of items" / "Anzahl Einträge", an integer from 1 to 100, default 10), "Show summary" / "Zusammenfassung zeigen" (default off), "Only items with audio" / "Nur Einträge mit Audio" (default off), an optional title (at most 200 characters after trimming), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). The choice of feeds SHALL be "All my feeds" / "Alle meine Feeds" (default) or a list of 1–50 chosen feeds, in a search field over the owner's feed titles. When the owner has no feeds, the dialog SHALL say "You don't have any feeds yet." / "Du hast noch keine Feeds." with a link to `/feeds`. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles and links of these items, and with "All my feeds", those of feeds added later too. A layout with a feed widget with more than 50 feeds, a feed ID longer than an ID can be, a repeated feed ID, a show value other than these, a number of items outside 1–100, a title longer than 200 characters or a size other than `S`, `M` or `L` is rejected with 400 and `invalid_layout`. The feeds don't have to exist when the layout is saved.

#### Scenario: Add a feed widget
- **WHEN** Ann picks "Feed" in the widget picker and confirms the dialog without changes
- **THEN** a widget with "All my feeds", "Latest", 10 items, no summary, not limited to items with audio, size M and "Show on public dashboard" off is added to the draft

#### Scenario: Widget saved before the setting existed
- **WHEN** a feed widget has no value for "Only items with audio"
- **THEN** it shows all of its items and its settings dialog shows the setting off

#### Scenario: Invalid count
- **WHEN** a layout is saved with a feed widget whose number of items is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: No feeds yet
- **WHEN** Ann has no feeds and opens the settings of a new feed widget
- **THEN** the dialog says "You don't have any feeds yet." with a link to `/feeds`

#### Scenario: Continue listening
- **WHEN** Ann sets "Show" to "Continue listening" and has three episodes in progress
- **THEN** the widget shows the three episodes, the one she listened to last first

### Requirement: Feed widget on public dashboards
For anyone other than the owner, a feed widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely. With the flag on, visitors get the widget with the items it shows, each with its ID, title, link, time, audio, summary when the widget shows summaries, and its feed's display title and icon, and nothing else: no feed URL, no subscription ID, no flag or flag time and no listen record. Visitors SHALL get no flag buttons and no listen progress, and can open links and play episodes like the owner. A widget with "Continue listening" shows visitors the titles of the owner's episodes in progress, without positions or times.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a feed widget whose "Show on public dashboard" is off
- **THEN** the response contains no trace of the widget or its items

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor plays an episode in the widget
- **THEN** the player plays it for the visitor, and the row has no flag buttons

#### Scenario: Feed URL stays private
- **WHEN** a visitor's response contains a feed widget whose feed URL has a token in its query
- **THEN** the response doesn't contain the feed URL

#### Scenario: Saved items shown on purpose
- **WHEN** the widget shows "Liked" and its "Show on public dashboard" is on
- **THEN** visitors see those items without their flag times

#### Scenario: No progress for visitors
- **WHEN** the widget shows "Continue listening", its "Show on public dashboard" is on, and a visitor opens the dashboard
- **THEN** the visitor sees the episodes without progress bars, and the response contains no listen record
