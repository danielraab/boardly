## MODIFIED Requirements

### Requirement: Item content
For each feed entry, the server SHALL store a title, an optional link, an optional summary, a time and optional audio. Only the newest `BOARDLY_FEED_LATEST_ITEMS` entries of a fetched document SHALL be stored. An entry with neither a title nor a summary is left out, and an entry without a title takes the first 100 characters of its summary. The title and the summary SHALL be plain text, with HTML removed and entities decoded. The title SHALL be cut to 500 characters and the summary to 1000 characters, also for items created by an import. A link that isn't an absolute `http` or `https` URL SHALL be dropped. The time is the entry's published or updated time, or the time it was first seen when it has none, and never later than the time it was first seen. An entry has audio when it has an enclosure of an audio type, or of a generic type with a common audio file extension, with an `http` or `https` URL. An `http` audio URL SHALL be stored as `https`. Images in entries SHALL NOT be stored or shown.

#### Scenario: HTML in the summary
- **WHEN** an entry's description is `<p>Hello <b>world</b></p><script>x()</script>`
- **THEN** its summary is "Hello world"

#### Scenario: Long summary
- **WHEN** an entry's description is 1500 characters of plain text
- **THEN** its summary is its first 1000 characters

#### Scenario: Long title
- **WHEN** an entry's title is 800 characters long
- **THEN** its title is its first 500 characters

#### Scenario: Summary of an older item gets longer
- **WHEN** an item was stored with a summary cut to 500 characters and a refresh returns the same entry
- **THEN** the item's summary has up to 1000 characters

#### Scenario: Dangerous link
- **WHEN** an entry's link is `javascript:alert(1)`
- **THEN** the item has no link

#### Scenario: Episode
- **WHEN** an entry has an enclosure with the type `audio/mpeg` and the URL `http://cdn.example.org/1.mp3`
- **THEN** the item's audio URL is `https://cdn.example.org/1.mp3`

#### Scenario: Date in the future
- **WHEN** an entry claims a date next year
- **THEN** its time is the time it was first seen

### Requirement: Item rows
Each item on `/feeds` SHALL show the icon of its feed, its title, its feed's title, its time as a relative time with the full date and time as a tooltip, and its summary, if any, cut off after two lines. A row of an item with a link and without audio SHALL open the link in a new tab, without giving the new page access to the opener. A row of an item with audio SHALL have a play button that plays the episode in the media player (see the `media-player` capability) and shows a pause button while it is playing, and its title links to the item's link, if any. An item without a link and without audio can't be activated.

A row narrower than 480 px SHALL use the narrow layout: the first line shows the feed's icon, the feed's title and the time; the item's title follows over the full width of the row on up to two lines; then the summary; and the buttons of the row (play, flags) are on a line below the text, aligned to the end. Text that doesn't fit is cut off with "…". A wider row keeps the buttons to the right of the text. The layout depends on the width of the row, not of the viewport.

A summary that is cut off SHALL have a toggle "Show more" / "Mehr anzeigen" that shows the whole summary in the row, and then "Show less" / "Weniger anzeigen" that cuts it off again. The toggle states whether the summary is expanded to assistive technology, doesn't open the item's link and doesn't start playback. A summary that fits in two lines has no toggle. Expanding is not remembered.

#### Scenario: News item
- **WHEN** Ann activates a news item
- **THEN** its link opens in a new tab and `/feeds` stays open

#### Scenario: Episode
- **WHEN** Ann activates the play button of an episode
- **THEN** the media player plays it and the button becomes a pause button

#### Scenario: Narrow row
- **WHEN** Ann opens `/feeds` on a phone 375 px wide
- **THEN** each row shows the feed's title in its first line, the item's title below it over the full width of the row, and the play and flag buttons below the text

#### Scenario: Wide row
- **WHEN** Ann opens `/feeds` in a window 1280 px wide
- **THEN** the play and flag buttons are to the right of the text

#### Scenario: Expand a summary
- **WHEN** an item's summary is longer than two lines and Ann activates "Show more"
- **THEN** the whole summary is shown, the toggle says "Show less", and no link opens

#### Scenario: Short summary
- **WHEN** an item's summary fits in two lines
- **THEN** the row has no "Show more" toggle
