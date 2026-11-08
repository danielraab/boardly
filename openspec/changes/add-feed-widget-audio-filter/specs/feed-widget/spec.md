# Spec Delta

## MODIFIED Requirements

### Requirement: Feed widget settings
The widget kind `feed` ("Feed" / "Feed") SHALL have, besides the common width, a choice of feeds, "Show" / "Anzeigen" with the values "Latest" / "Neueste" (`latest`, default), "Read later" / "Später lesen" (`read_later`) and "Liked" / "Gefällt mir" (`liked`), a number of items ("Number of items" / "Anzahl Einträge", an integer from 1 to 100, default 10), "Show summary" / "Zusammenfassung zeigen" (default off), "Only items with audio" / "Nur Einträge mit Audio" (default off), an optional title (at most 200 characters after trimming), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). The choice of feeds SHALL be "All my feeds" / "Alle meine Feeds" (default) or a list of 1–50 chosen feeds, in a search field over the owner's feed titles. When the owner has no feeds, the dialog SHALL say "You don't have any feeds yet." / "Du hast noch keine Feeds." with a link to `/feeds`. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles and links of these items, and with "All my feeds", those of feeds added later too. A layout with a feed widget with more than 50 feeds, a feed ID longer than an ID can be, a repeated feed ID, a show value other than these, a number of items outside 1–100, a title longer than 200 characters or a size other than `S`, `M` or `L` is rejected with 400 and `invalid_layout`. The feeds don't have to exist when the layout is saved.

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

### Requirement: Feed widget rendering
A feed widget SHALL show its title, if any, and at most its number of items, merged from its feeds. With "Only items with audio", the widget shows only items that have audio, for every "Show" value and for visitors too, and its number of items counts only those. With "Latest", items are sorted newest first. With "Read later" and "Liked", items are sorted by when the flag was set, newest first. "All my feeds" with "Read later" or "Liked" includes items of feeds the owner no longer subscribes to. Each item is a row with the feed's icon, the item's title, the feed's title as a second line and its time as a relative time with the full date and time as a tooltip. With "Show summary", the summary is shown below, cut off after two lines. Text that doesn't fit is cut off with "…". A row of an item with a link and without audio SHALL open the link in a new tab, without giving the new page access to the opener. A row of an item with audio SHALL show a play button that plays it in the media player and pauses it while it is playing, and the widget SHALL mark the episode that is loaded in the player. The size sets the text, icon and padding on the same S/M/L scale as the other widgets. In edit mode, rows SHALL NOT open links or start playback, and flags can't be changed. A widget without items to show SHALL display "No items" / "Keine Einträge".

#### Scenario: Merged list
- **WHEN** a widget shows two feeds with items from today and yesterday
- **THEN** the rows of both feeds appear in one list, newest first

#### Scenario: Only items with audio
- **WHEN** a widget with "Only items with audio" and 2 items shows a feed whose newest items are an article, an episode, an article and an episode
- **THEN** the widget shows the two episodes, newest first

#### Scenario: News row
- **WHEN** Ann activates a news row
- **THEN** its link opens in a new tab

#### Scenario: Episode row
- **WHEN** Ann activates the play button of an episode row
- **THEN** the media player plays it and the row is marked as playing

#### Scenario: Read later list
- **WHEN** the widget shows "Read later" and Ann flagged items A and then B
- **THEN** B is listed before A

#### Scenario: Edit mode
- **WHEN** Ann activates a row while editing the dashboard
- **THEN** no link opens and nothing plays
