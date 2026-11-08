# Spec Delta

## Purpose

A dashboard widget that shows a glance list of items from the owner's feeds, either the latest ones or those flagged to read later or as liked, for the owner and, if allowed, for visitors of public dashboards.

## ADDED Requirements

### Requirement: Feed widget settings
The widget kind `feed` ("Feed" / "Feed") SHALL have, besides the common width, a choice of feeds, "Show" / "Anzeigen" with the values "Latest" / "Neueste" (`latest`, default), "Read later" / "Später lesen" (`read_later`) and "Liked" / "Gefällt mir" (`liked`), a number of items ("Number of items" / "Anzahl Einträge", an integer from 1 to 100, default 10), "Show summary" / "Zusammenfassung zeigen" (default off), an optional title (at most 200 characters after trimming), a size (`S`, `M` or `L`, default `M`) and "Show on public dashboard" / "Auf öffentlichem Dashboard zeigen" (default off). The choice of feeds SHALL be "All my feeds" / "Alle meine Feeds" (default) or a list of 1–50 chosen feeds, in a search field over the owner's feed titles. When the owner has no feeds, the dialog SHALL say "You don't have any feeds yet." / "Du hast noch keine Feeds." with a link to `/feeds`. Next to "Show on public dashboard", the dialog SHALL say that visitors then see the titles and links of these items, and with "All my feeds", those of feeds added later too. A layout with a feed widget with more than 50 feeds, a feed ID longer than an ID can be, a repeated feed ID, a show value other than these, a number of items outside 1–100, a title longer than 200 characters or a size other than `S`, `M` or `L` is rejected with 400 and `invalid_layout`. The feeds don't have to exist when the layout is saved.

#### Scenario: Add a feed widget
- **WHEN** Ann picks "Feed" in the widget picker and confirms the dialog without changes
- **THEN** a widget with "All my feeds", "Latest", 10 items, no summary, size M and "Show on public dashboard" off is added to the draft

#### Scenario: Invalid count
- **WHEN** a layout is saved with a feed widget whose number of items is 0 or 101
- **THEN** the server responds 400 with `invalid_layout`

#### Scenario: No feeds yet
- **WHEN** Ann has no feeds and opens the settings of a new feed widget
- **THEN** the dialog says "You don't have any feeds yet." with a link to `/feeds`

### Requirement: Feed widget rendering
A feed widget SHALL show its title, if any, and at most its number of items, merged from its feeds. With "Latest", items are sorted newest first. With "Read later" and "Liked", items are sorted by when the flag was set, newest first. "All my feeds" with "Read later" or "Liked" includes items of feeds the owner no longer subscribes to. Each item is a row with the feed's icon, the item's title, the feed's title as a second line and its time as a relative time with the full date and time as a tooltip. With "Show summary", the summary is shown below, cut off after two lines. Text that doesn't fit is cut off with "…". A row of an item with a link and without audio SHALL open the link in a new tab, without giving the new page access to the opener. A row of an item with audio SHALL show a play button that plays it in the media player and pauses it while it is playing, and the widget SHALL mark the episode that is loaded in the player. The size sets the text, icon and padding on the same S/M/L scale as the other widgets. In edit mode, rows SHALL NOT open links or start playback, and flags can't be changed. A widget without items to show SHALL display "No items" / "Keine Einträge".

#### Scenario: Merged list
- **WHEN** a widget shows two feeds with items from today and yesterday
- **THEN** the rows of both feeds appear in one list, newest first

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

### Requirement: Flags in the feed widget
For the owner, outside edit mode, each row SHALL have the bookmark and the heart button of the `feeds` capability. Removing the flag that a "Read later" or "Liked" widget shows SHALL remove the row. Flag changes SHALL show in every widget on the dashboard and on `/feeds` without a reload.

#### Scenario: Flag from the widget
- **WHEN** Ann activates the heart button of a row
- **THEN** the item is in her "Liked" view

#### Scenario: Removed from a read later widget
- **WHEN** a widget shows "Read later" and Ann removes the flag of a row
- **THEN** the row disappears

### Requirement: Removed feeds in the widget
A feed in a feed widget that the owner no longer subscribes to SHALL be left out of the widget for everyone. When none of a widget's chosen feeds exist, the widget SHALL show "These feeds no longer exist" / "Diese Feeds existieren nicht mehr" to the owner, and its settings can still be opened. For visitors, the widget is left out.

#### Scenario: One feed removed
- **WHEN** a widget has "Heise" and "Podcast Eins" and Ann removes "Heise"
- **THEN** the widget shows only the items of "Podcast Eins"

#### Scenario: All removed
- **WHEN** all of a widget's chosen feeds are removed
- **THEN** the owner sees "These feeds no longer exist", and visitors don't see the widget

### Requirement: Feed widget on public dashboards
For anyone other than the owner, a feed widget with "Show on public dashboard" off SHALL be left out of the dashboard lookup response entirely. With the flag on, visitors get the widget with the items it shows, each with its ID, title, link, time, audio, summary when the widget shows summaries, and its feed's display title and icon, and nothing else: no feed URL, no subscription ID and no flag or flag time. Visitors SHALL get no flag buttons and can open links and play episodes like the owner.

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
