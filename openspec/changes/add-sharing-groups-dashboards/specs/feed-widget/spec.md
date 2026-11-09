# Spec Delta

## MODIFIED Requirements

### Requirement: Feed widget settings
The widget kind `feed` ("Feed" / "Feed") SHALL have, besides the common width, a choice of feeds, "Show" / "Anzeigen" with the values "Latest" / "Neueste" (`latest`, default), "Read later" / "Später lesen" (`read_later`) and "Liked" / "Gefällt mir" (`liked`), a number of items ("Number of items" / "Anzahl Einträge", an integer from 1 to 100, default 10), "Show summary" / "Zusammenfassung zeigen" (default off), "Only items with audio" / "Nur Einträge mit Audio" (default off), an optional title (at most 200 characters after trimming), a size (`S`, `M` or `L`, default `M`) and "Show to other viewers" / "Anderen Betrachtern zeigen" (default off). The choice of feeds SHALL be "All my feeds" / "Alle meine Feeds" (default) or a list of 1–50 chosen feeds, in a search field over the owner's feed titles. When the owner has no feeds, the dialog SHALL say "You don't have any feeds yet." / "Du hast noch keine Feeds." with a link to `/feeds`. Next to "Show to other viewers", the dialog SHALL say that visitors then see the titles and links of these items, and with "All my feeds", those of feeds added later too. A layout with a feed widget with more than 50 feeds, a feed ID longer than an ID can be, a repeated feed ID, a show value other than these, a number of items outside 1–100, a title longer than 200 characters or a size other than `S`, `M` or `L` is rejected with 400 and `invalid_layout`. The feeds don't have to exist when the layout is saved.

#### Scenario: Add a feed widget
- **WHEN** Ann picks "Feed" in the widget picker and confirms the dialog without changes
- **THEN** a widget with "All my feeds", "Latest", 10 items, no summary, not limited to items with audio, size M and "Show to other viewers" off is added to the draft

#### Scenario: Widget saved before the setting existed
- **WHEN** a feed widget has no value for "Only items with audio"
- **THEN** it shows all of its items and its settings dialog shows the setting off

#### Scenario: Invalid count
- **WHEN** a layout is saved with a feed widget whose number of items is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: No feeds yet
- **WHEN** Ann has no feeds and opens the settings of a new feed widget
- **THEN** the dialog says "You don't have any feeds yet." with a link to `/feeds`

### Requirement: Feed widget on public dashboards
For anyone other than the owner, a feed widget with "Show to other viewers" off SHALL be left out of the dashboard lookup response entirely. With the flag on, visitors get the widget with the items it shows, each with its ID, title, link, time, audio, summary when the widget shows summaries, and its feed's display title and icon, and nothing else: no feed URL, no subscription ID and no flag or flag time. Visitors SHALL get no flag buttons and can open links and play episodes like the owner.

#### Scenario: Flag off
- **WHEN** a visitor opens Ann's public dashboard with a feed widget whose "Show to other viewers" is off
- **THEN** the response contains no trace of the widget or its items

#### Scenario: Flag on
- **WHEN** the flag is on and a visitor plays an episode in the widget
- **THEN** the player plays it for the visitor, and the row has no flag buttons

#### Scenario: Feed URL stays private
- **WHEN** a visitor's response contains a feed widget whose feed URL has a token in its query
- **THEN** the response doesn't contain the feed URL

#### Scenario: Saved items shown on purpose
- **WHEN** the widget shows "Liked" and its "Show to other viewers" is on
- **THEN** visitors see those items without their flag times
