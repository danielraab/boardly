# feed-widget Specification

## Purpose
A dashboard widget that shows a glance list of items from the owner's feeds, either the latest ones or those flagged to read later or as liked, for the owner and, if allowed, for visitors of public dashboards.
## Requirements
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

### Requirement: Feed widget rendering
A feed widget SHALL show its title, if any, and at most its number of items, merged from its feeds. With "Only items with audio", the widget shows only items that have audio, for every "Show" value and for visitors too, and its number of items counts only those. With "Latest", items are sorted newest first. With "Read later" and "Liked", items are sorted by when the flag was set, newest first. "All my feeds" with "Read later" or "Liked" includes items of feeds the owner no longer subscribes to. Each item is a row with the feed's icon, the item's title, the feed's title as a second line and its time as a relative time with the full date and time as a tooltip. With "Show summary", the summary is shown below, cut off after two lines. Text that doesn't fit is cut off with "…". A row of an item with a link and without audio SHALL open the link in a new tab, without giving the new page access to the opener. A row of an item with audio SHALL show a play button that plays it in the media player and pauses it while it is playing, and the widget SHALL mark the episode that is loaded in the player. The size sets the text, icon and padding on the same S/M/L scale as the other widgets. In edit mode, rows SHALL NOT open links or start playback, and flags can't be changed. A widget without items to show SHALL display "No items" / "Keine Einträge".

Rows narrower than 480 px SHALL use the narrow layout of item rows of the `feeds` capability: the feed's icon, the feed's title and the time first, the item's title over the full width of the row below, and the buttons on a line below the text. With "Show summary", a summary that is cut off SHALL have the "Show more" / "Mehr anzeigen" and "Show less" / "Weniger anzeigen" toggle of the `feeds` capability, for the owner and for visitors. In edit mode, the toggle SHALL NOT be shown.

Without "Show summary", the row of an item with a summary SHALL have an info button "Show summary" / "Zusammenfassung zeigen" among its buttons, before the play button, to the right of the text when the row is wide and on the line of buttons when it is narrow. Activating it SHALL show the whole summary in the row, below the title, without cutting it off; the button then says "Hide summary" / "Zusammenfassung verbergen" and hides it again. The button states whether the summary is shown to assistive technology, doesn't open the item's link and doesn't start playback. Showing a summary is not remembered. A row of an item without a summary, and every row in edit mode, SHALL have no info button. Visitors get no summaries without "Show summary" (see "Feed widget on public dashboards"), so their rows have no info button.

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

#### Scenario: Narrow widget column
- **WHEN** a feed widget is in a column 300 px wide on a desktop
- **THEN** its rows use the narrow layout, with the buttons below the text

#### Scenario: Expand a summary in the widget
- **WHEN** a widget with "Show summary" shows an item whose summary is longer than two lines, and a visitor activates "Show more"
- **THEN** the whole summary is shown in the row and no link opens

#### Scenario: No toggle in edit mode
- **WHEN** Ann edits the dashboard
- **THEN** the rows of the feed widget have no "Show more" toggle

#### Scenario: Open a summary on demand
- **WHEN** a widget without "Show summary" shows an item with a summary, and Ann activates its info button
- **THEN** the whole summary is shown below the title, the button says "Hide summary", and no link opens

#### Scenario: Hide a summary again
- **WHEN** Ann activates "Hide summary" on a row whose summary she opened
- **THEN** the summary is no longer shown and the button says "Show summary"

#### Scenario: No summary to show
- **WHEN** a widget without "Show summary" shows an item without a summary
- **THEN** its row has no info button

#### Scenario: Info button in a narrow row
- **WHEN** a widget without "Show summary" is in a column 300 px wide and Ann views it
- **THEN** the info button is on the line of buttons with the flag buttons, and the row has no extra line for it

#### Scenario: No info button for visitors
- **WHEN** a visitor views a public dashboard with a feed widget without "Show summary"
- **THEN** its rows have no info button and show no summary

#### Scenario: No info button in edit mode
- **WHEN** Ann edits a dashboard with a feed widget without "Show summary"
- **THEN** its rows have no info button

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

### Requirement: Order of the continue listening widget
With "Continue listening", a feed widget SHALL show the owner's episodes in progress (see the `feeds` capability) of its feeds, sorted by when their listen record was last changed, newest first. "All my feeds" includes episodes of feeds the owner no longer subscribes to. An episode that is finished, marked as unplayed or removed from history SHALL leave the widget without a reload of the page, for the owner. For the owner, its rows show the progress described in "Listen progress on rows" of the `feeds` capability.

#### Scenario: Finished episode leaves the widget
- **WHEN** a "Continue listening" widget shows an episode and Ann plays it to its end
- **THEN** the episode disappears from the widget

#### Scenario: Order
- **WHEN** Ann listened to episode A yesterday and to episode B today
- **THEN** the widget lists B before A

