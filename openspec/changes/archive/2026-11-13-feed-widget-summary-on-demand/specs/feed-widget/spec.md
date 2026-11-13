## MODIFIED Requirements

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
