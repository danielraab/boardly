# feeds Specification

## Purpose
Lets signed-in users subscribe to RSS, Atom and JSON feeds of news and podcasts, stores every feed and item once per server, keeps them up to date in the background, and lets each user flag items to read later or as liked.

## Requirements

### Requirement: Feeds page
`/feeds` SHALL require a signed-in user and show the heading "Feeds" / "Feeds", an "Add feed" / "Feed hinzufügen" button, the views "Latest" / "Neueste" (default), "Read later" / "Später lesen", "Liked" / "Gefällt mir" and "Listened" / "Gehört" (see "Listened view"), the items of the chosen view newest first, and a "Manage feeds" / "Feeds verwalten" action that lists the user's feeds in their order. Signed-out visitors SHALL be sent to `/login`. "Latest" shows the items of the user's subscribed feeds, sorted by item time. "Read later" and "Liked" show the user's items with that flag, sorted by when the flag was set, also when the user no longer subscribes to the item's feed. When the user has no feeds, the page SHALL show "No feeds yet" / "Noch keine Feeds" with a hint to add one. An empty "Read later", "Liked" or "Listened" view SHALL say "Nothing here yet" / "Noch nichts hier". The items load in pages of 50 with a "Load more" / "Mehr laden" button. The document title is "Feeds - <app title>".

While the user has feeds, the page SHALL show, above the items, a search field "Search items" / "Einträge durchsuchen" and a feed filter "Feed" / "Feed" with "All feeds" / "Alle Feeds" (default) and the user's feeds in their order. Both apply to the items of the current view, in the order of that view, as described for `q` and `feedId` in "Feed API". The search SHALL start shortly after typing stops, without a button, and a search field that is empty or only has spaces shows all items of the view. Switching the view SHALL keep the search and the filter. A view without items for the search or the filter SHALL say "No matching items" / "Keine passenden Einträge". The search field SHALL have a clear button "Clear search" / "Suche löschen". A filter whose feed the user removes SHALL fall back to "All feeds".

#### Scenario: Open the page
- **WHEN** Ann subscribes to "Heise" and "Podcast Eins" and opens `/feeds`
- **THEN** the page shows the view "Latest" with the items of both feeds, newest first

#### Scenario: Empty
- **WHEN** Ann has no feeds and opens `/feeds`
- **THEN** the page shows "No feeds yet"

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/feeds`
- **THEN** the browser shows `/login`

#### Scenario: Saved item of a removed feed
- **WHEN** Ann liked an item of "Heise" and then removed "Heise" from her feeds
- **THEN** the view "Liked" still shows the item, and "Latest" doesn't

#### Scenario: Search in the current view
- **WHEN** Ann is in "Liked" and types "rust"
- **THEN** the page shows only her liked items with "rust" in their title or summary

#### Scenario: Search kept across views
- **WHEN** Ann searches for "rust" in "Latest" and then switches to "Read later"
- **THEN** the search field still says "rust" and "Read later" shows only matching items

#### Scenario: Filter by feed
- **WHEN** Ann chooses "Podcast Eins" in the feed filter
- **THEN** the page shows only the items of "Podcast Eins"

#### Scenario: Nothing found
- **WHEN** Ann searches for a word that no item of the view contains
- **THEN** the page says "No matching items"

#### Scenario: Clear the search
- **WHEN** Ann activates "Clear search"
- **THEN** the search field is empty and all items of the view are shown

#### Scenario: Search in the history
- **WHEN** Ann is in "Listened" and types "interview"
- **THEN** the page shows only the episodes with a listen record that contain "interview"

### Requirement: Feeds and items are stored once
The server SHALL store each feed once, identified by its resolved URL, and each item of a feed once, identified by the feed and the item's identity. Two users who subscribe to the same feed SHALL share the same feed and item records, and the feed SHALL be fetched once per refresh, whatever the number of subscribers. An item's identity is its GUID (`guid` or `id` in the feed), or its link when it has no GUID, or a hash of its title and date when it has neither. What belongs to one user SHALL be stored per user: the subscription with its position, custom title and icon, and the flags on items. A user SHALL NOT see which other users subscribe to a feed or flag an item. Items of different feeds are different items, even when they have the same GUID or link.

#### Scenario: Two users, one feed
- **WHEN** Ann and Bob both subscribe to `https://example.org/feed.xml`
- **THEN** one feed with one set of items exists, and each refresh fetches the URL once

#### Scenario: Same feed through a redirect
- **WHEN** Ann subscribed to `https://example.org/feed.xml` and Bob adds `http://example.org/feed.xml`, which redirects there
- **THEN** Bob gets a subscription to the same feed, and no second feed is created

#### Scenario: Item seen again
- **WHEN** a refresh returns an item with a GUID that is already stored for this feed, with a new title
- **THEN** the item keeps its ID and its flags and has the new title

#### Scenario: Own titles stay private
- **WHEN** Ann renames her subscription of a feed that Bob also has
- **THEN** Bob's list still shows the feed's title

#### Scenario: Feed moved to a URL that is already known
- **WHEN** a refresh of feed A answers with a permanent redirect to the URL of feed B
- **THEN** A and B become one feed, an item that was in both is kept once, every user's subscription and flags are kept, and a user who subscribed to both has one subscription

### Requirement: Feed API
The web app SHALL manage feeds through these endpoints, all of which require a session and respond 401 without one:
- `GET /api/feeds` lists the user's subscriptions in their order, each with `id`, `feedId`, `url`, `title` (the custom title or else the feed's title), `customTitle` (or `null`), `siteUrl` (or `null`), `icon` (as for notes, or `null`), `faviconId` (or `null`), `position`, `lastFetchedAt` (or `null`) and `failing`.
- `POST /api/feeds` subscribes from `url` (see "Adding a feed"), adds the subscription at the end of the list and responds 201 with it.
- `PATCH /api/feeds/{id}` changes `customTitle` (`null` or empty removes it) or `icon` (`null` removes it) and responds with the subscription. A custom title longer than 200 characters after trimming gets 400 `invalid_title`.
- `PUT /api/feeds/order` takes the IDs of all the user's subscriptions in their new order. A list that isn't exactly the user's subscription IDs gets 400 `invalid_order`.
- `DELETE /api/feeds/{id}` removes the subscription and responds 204.
- `GET /api/feed-items?view=<latest|read_later|liked|listened|in_progress|finished>&feedId=<subscription ID>&q=<search>&limit=<1-100>&cursor=<cursor>` responds with `items` and `nextCursor` (or `null`). `view` defaults to `latest`, `limit` to 50, and `feedId` and `q` are optional. An unknown view, a limit outside 1–100, a `q` longer than 200 characters or an invalid cursor gets 400 `invalid_request`. Each item has `id`, `title`, `link` (or `null`), `summary` (or `null`), `publishedAt`, `audio` (`{ "url", "type", "durationSeconds" }` or `null`), `feed` (`{ "title", "subscriptionId", "icon", "faviconId" }`, with `subscriptionId` `null` when the user doesn't subscribe), `readLaterAt` (or `null`), `likedAt` (or `null`) and `listen` (`{ "positionSeconds", "durationSeconds", "startedAt", "updatedAt", "finishedAt" }` with `durationSeconds` and `finishedAt` possibly `null`, or `null` without a record).
- `PATCH /api/feed-items/{id}` takes `readLater` and `liked`, each a boolean and optional, and responds with the item.
- `PUT` and `DELETE /api/feed-items/{id}/listen` change the listen record (see "Listen API").

The views `listened`, `in_progress` and `finished` show the user's items with a listen record, an unfinished record with a position greater than 0, and a finished record, sorted by the record's `updatedAt`, newest first. `feedId` limits the items of any view to the feed of that subscription of the user. A `feedId` that isn't one of the user's subscriptions gives no items. `q` is split into words at whitespace, and an item matches when every word appears in its title or its summary, ignoring case for all letters, not only ASCII. A `q` that is empty or only has whitespace matches every item. Search doesn't change the order of a view, and paging with `cursor` keeps `q` and `feedId`.

Requests for another user's subscription, or for an unknown one, get 404 `not_found`. An item the user neither subscribes to nor has flagged nor has a listen record of gets 404 `not_found`.

#### Scenario: Another user's subscription
- **WHEN** Bob requests `PATCH /api/feeds/<ID of Ann's subscription>`
- **THEN** the server responds 404 with `not_found`

#### Scenario: Item of a feed the user doesn't have
- **WHEN** Bob, who doesn't subscribe to "Heise", patches an item of "Heise"
- **THEN** the server responds 404 with `not_found` and no flag is set

#### Scenario: No session
- **WHEN** a request to `GET /api/feeds` has no session
- **THEN** the server responds 401

#### Scenario: Reorder
- **WHEN** Ann has "Heise" and "Podcast Eins" and puts `[<Podcast Eins>, <Heise>]` to `/api/feeds/order`
- **THEN** `GET /api/feeds` lists "Podcast Eins" first

#### Scenario: All words must match
- **WHEN** Ann has the items "Rust in production" and "Go in production" and requests `q=production rust`
- **THEN** only "Rust in production" is returned

#### Scenario: Match in the summary
- **WHEN** an item's title doesn't contain "Kubernetes" but its summary does, and Ann requests `q=kubernetes`
- **THEN** the item is returned

#### Scenario: Case of non-ASCII letters
- **WHEN** an item's title is "Ärger im Büro" and Ann requests `q=ärger`
- **THEN** the item is returned

#### Scenario: Search with feed and view
- **WHEN** Ann requests `view=liked&feedId=<Heise>&q=apple`
- **THEN** only her liked items of "Heise" that contain "apple" are returned, sorted by when she liked them

#### Scenario: Another user's feed as filter
- **WHEN** Bob requests `GET /api/feed-items?feedId=<ID of Ann's subscription>`
- **THEN** the response has no items

#### Scenario: Query too long
- **WHEN** Ann requests a `q` of 201 characters
- **THEN** the server responds 400 with `invalid_request`

#### Scenario: In progress
- **WHEN** Ann has one episode at 10:00, one finished and one marked as unplayed, and requests `view=in_progress`
- **THEN** only the episode at 10:00 is returned, with its `listen`

#### Scenario: Listened item of a removed feed
- **WHEN** Ann listened to an episode and no longer subscribes to its feed
- **THEN** `PUT /api/feed-items/<ID>/listen` still works for her, and the item has `feed.subscriptionId` `null`

### Requirement: Adding a feed
`POST /api/feeds` SHALL accept the URL of a feed, or of a web page that links to one. Only `http` and `https` URLs are accepted, and anything else gets 400 `invalid_url`. When the URL is already the URL of a stored feed that has been fetched successfully at least once, the server SHALL use that feed without fetching. Otherwise the server SHALL fetch the URL, trying `https` first for an `http` URL and keeping `http` only when `https` doesn't work, and following at most 5 redirects. When the response is an HTML page, the server SHALL use the first RSS, Atom or JSON feed it advertises through `<link rel="alternate">` on the same fetch rules. The resolved URL is the feed's URL, and a stored feed with that URL SHALL be used instead of creating one. A URL that can't be fetched gets 400 `feed_unreachable`. A response that is neither a valid feed nor a page advertising one gets 400 `feed_not_found`. A feed the user already subscribes to gets 409 `already_subscribed`. Adding is limited per user, and too many requests get 429 `rate_limited` with `Retry-After`.

The server SHALL apply the private address protection of the `images` capability to every fetch, read at most 32 MB of a feed, and give up after 20 seconds.

#### Scenario: Feed URL
- **WHEN** Ann adds `https://example.org/feed.xml`, a valid RSS feed
- **THEN** the server responds 201, the feed is last in her list, and its items are available right away

#### Scenario: Website URL
- **WHEN** Ann adds `https://example.org/`, whose page advertises `/feed.xml` as an RSS feed
- **THEN** she subscribes to `https://example.org/feed.xml`

#### Scenario: Not a feed
- **WHEN** Ann adds a page that advertises no feed
- **THEN** the server responds 400 with `feed_not_found`

#### Scenario: Already subscribed
- **WHEN** Ann adds a feed she already has
- **THEN** the server responds 409 with `already_subscribed` and her list is unchanged

#### Scenario: Known feed isn't fetched
- **WHEN** Bob adds a URL that is already stored because Ann subscribes to it
- **THEN** Bob's subscription is created without a request to the feed's host

#### Scenario: Private address
- **WHEN** private addresses are not allowed and Ann adds a URL whose host resolves to `192.168.1.20`
- **THEN** the server responds 400 with `feed_unreachable` and makes no connection to that address

### Requirement: Background refresh
The server SHALL refresh every feed that has at least one subscriber on the cycle set by `BOARDLY_FEED_REFRESH_MINUTES`, in the background, also while nobody is signed in. A refresh SHALL send `ETag` and `Last-Modified` values from the previous response as conditional headers and treat 304 as no change. It SHALL use the same limits and the private address protection as "Adding a feed". A failed refresh SHALL leave the stored items as they are and double the waiting time for that feed with each failure, up to 24 hours, and a `Retry-After` of a 429 or 503 response SHALL be respected when it is longer. After a success, the normal cycle applies again. A feed that failed at least 3 times in a row SHALL be reported as `failing`. A feed without subscribers SHALL NOT be fetched. Showing a dashboard or the feeds page SHALL NOT fetch a feed.

#### Scenario: New item appears
- **WHEN** a feed gets a new item and its refresh is due
- **THEN** the item is stored and shows up in the "Latest" view

#### Scenario: Not changed
- **WHEN** the host answers a refresh with 304
- **THEN** the items stay as they are and the next refresh is one cycle later

#### Scenario: Host down
- **WHEN** a feed's host fails for three refreshes in a row
- **THEN** the feed is reported as `failing`, its items are still shown, and the wait before the next try has grown

#### Scenario: No outbound request from a visitor
- **WHEN** a visitor opens a public dashboard with a feed widget
- **THEN** the server makes no request to the feed's host

#### Scenario: No subscribers
- **WHEN** the last subscriber removes a feed
- **THEN** the feed isn't fetched any more

### Requirement: Feed settings
The server SHALL read two environment variables at start:
- `BOARDLY_FEED_LATEST_ITEMS` is how many of the newest items are kept per feed, a whole number from 1 to 1000, default 100.
- `BOARDLY_FEED_REFRESH_MINUTES` is the refresh cycle in minutes, a whole number from 5 to 10080, default 60.

A value that isn't a whole number in its range SHALL stop the server at start with an error that names the variable.

#### Scenario: Defaults
- **WHEN** neither variable is set
- **THEN** 100 items are kept per feed and feeds are refreshed every 60 minutes

#### Scenario: Custom values
- **WHEN** `BOARDLY_FEED_LATEST_ITEMS=30` and `BOARDLY_FEED_REFRESH_MINUTES=15`
- **THEN** 30 items are kept per feed and feeds are refreshed every 15 minutes

#### Scenario: Out of range
- **WHEN** `BOARDLY_FEED_REFRESH_MINUTES=1`
- **THEN** the server doesn't start and the error names `BOARDLY_FEED_REFRESH_MINUTES`

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

### Requirement: Read later and like
Each user SHALL be able to flag any item they can see as "Read later" / "Später lesen" and as "Liked" / "Gefällt mir". The flags are independent of each other, belong to that user alone and stay until the user removes them. Opening an item or playing an episode SHALL NOT change them. The server SHALL store when each flag was set. Each item row on `/feeds` SHALL have a bookmark button and a heart button that toggle these flags, show their state and have accessible names ("Read later" / "Später lesen", "Remove from read later" / "Nicht mehr später lesen", "Like" / "Gefällt mir", "Remove like" / "Gefällt mir entfernen").

#### Scenario: Flag an item
- **WHEN** Ann activates the bookmark button of an item
- **THEN** the item is in her "Read later" view, and is not in Bob's

#### Scenario: Independent flags
- **WHEN** Ann likes an item that is in "Read later" and then removes "Read later"
- **THEN** the item is still in "Liked"

#### Scenario: Opening keeps the flag
- **WHEN** Ann opens an item from "Read later"
- **THEN** it stays in "Read later"

### Requirement: Item cleanup
After a refresh and at least once per hour, the server SHALL delete the items of a feed that are outside its newest `BOARDLY_FEED_LATEST_ITEMS` items, unless at least one user has a flag or a listen record on the item. The newest items are chosen by item time among all items of the feed, flagged or not. An item whose last flag and listen record are removed SHALL be deleted by the next cleanup when it is outside the newest items. A feed with no subscribers and no items with a flag or a listen record SHALL be deleted with its items. An account's deletion removes its subscriptions, flags and listen records and leaves the shared feeds and items to this cleanup. When two feeds become one because one moved to the URL of the other, the listen records of an item that was in both SHALL be kept on the remaining item, and of two records of the same user, the one changed last is kept.

#### Scenario: Old item removed
- **WHEN** 100 items are kept per feed and a feed has 101 unflagged items
- **THEN** the oldest item is deleted

#### Scenario: Flagged item kept
- **WHEN** an item that Ann flagged is older than the newest 100 items of its feed
- **THEN** the item is kept and stays in her view

#### Scenario: Listened item kept
- **WHEN** an episode that Ann listened to is older than the newest 100 items of its feed
- **THEN** the episode is kept and stays in her "Listened" view

#### Scenario: Flag of another user
- **WHEN** Ann removes her like from an old item that Bob has marked "Read later"
- **THEN** the item is kept

#### Scenario: Last flag removed
- **WHEN** Ann removes the only flag of an item outside the newest 100 items
- **THEN** the item is deleted by the next cleanup

#### Scenario: Removed from history
- **WHEN** Ann removes the only listen record of an episode outside the newest 100 items, and nobody has a flag on it
- **THEN** the episode is deleted by the next cleanup

#### Scenario: Old items don't come back
- **WHEN** a feed's document lists 500 entries and 100 items are kept per feed
- **THEN** only the newest 100 entries are stored, and refreshing doesn't add and delete the older ones

#### Scenario: Unsubscribed feed with saved items
- **WHEN** Ann, the only subscriber, removes a feed in which she liked an item
- **THEN** the feed and that item stay, the other items are deleted by the cleanup, and the feed isn't fetched

#### Scenario: Unsubscribed feed without saved items
- **WHEN** the only subscriber removes a feed with no flagged or listened items
- **THEN** the cleanup deletes the feed and its items

#### Scenario: Limit lowered
- **WHEN** `BOARDLY_FEED_LATEST_ITEMS` is lowered from 100 to 30 and the server restarts
- **THEN** the next cleanup leaves the newest 30 and the flagged and listened items of each feed

### Requirement: Add feed dialog
The "Add feed" dialog SHALL have one required field "Feed or website URL" / "Feed- oder Website-URL" and a confirm button that shows a busy state while the server checks the URL. A failed check SHALL be shown at the field with a translated message for `invalid_url`, `feed_unreachable`, `feed_not_found`, `already_subscribed` and `rate_limited`, and the dialog stays open with the entered value. After success, the dialog closes and the new feed's items are shown.

#### Scenario: Add
- **WHEN** Ann enters a valid feed URL and confirms
- **THEN** the dialog closes and the page shows the feed's items

#### Scenario: Error stays in the dialog
- **WHEN** the server responds `feed_not_found`
- **THEN** the field says that no feed was found at this address and the dialog stays open

### Requirement: Manage feeds
"Manage feeds" SHALL list each feed with its icon, its title and its URL's host, and a menu with "Edit" / "Bearbeiten" and "Remove" / "Entfernen". The icon is the user's chosen icon, otherwise the shared favicon of the feed's site, otherwise a feed icon. "Edit" opens a dialog with the title and the icon, chosen with the same icon chooser as notes. Removing asks for confirmation, and the dialog SHALL say that items flagged "Read later" or "Liked" are kept. A feed reported as `failing` SHALL show "Can't be updated right now" / "Kann gerade nicht aktualisiert werden". Feeds SHALL be reorderable by dragging a handle, also with the keyboard as on the favourites page.

#### Scenario: Rename
- **WHEN** Ann edits "Heise", changes the title to "News" and confirms
- **THEN** her list and her items show "News"

#### Scenario: Failing feed
- **WHEN** one of Ann's feeds is `failing`
- **THEN** it shows "Can't be updated right now" and its items are still shown

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

### Requirement: Podcasts are separate from the media library
Feeds and their episodes SHALL NOT appear on `/media`, in `GET /api/media` or in the media widget's source list. They share only the media player.

#### Scenario: Podcast not on the media page
- **WHEN** Ann subscribes to a podcast and opens `/media`
- **THEN** the page shows only her stations

### Requirement: Listen history
The server SHALL keep, per user and item with audio, a listen record with the position in whole seconds, the duration the player measured (or `null`), when it was created (`startedAt`), when it was last changed (`updatedAt`) and when the episode was finished (or `null`). A record belongs to that user alone and SHALL stay until the user removes it or the account is deleted. An episode is "in progress" when its record isn't finished and its position is greater than 0, and "finished" when its record is finished. When the player of a signed-in user plays an episode, it SHALL save the position as described in the `media-player` capability. Visitors who aren't signed in get no records, and their playback stores nothing.

#### Scenario: Position saved on the server
- **WHEN** Ann plays an episode to 12:30 on her laptop and pauses
- **THEN** her listen record of the episode has the position 750 seconds

#### Scenario: Other device
- **WHEN** Ann paused an episode at 12:30 on her laptop and plays it on her phone
- **THEN** it continues at 12:30

#### Scenario: Records are private
- **WHEN** Ann and Bob both subscribe to a podcast and Ann listens to an episode
- **THEN** Bob's item has no listen record

#### Scenario: Finished
- **WHEN** Ann plays an episode to its end
- **THEN** the record is finished, its position is 0, and the episode is in "Finished"

### Requirement: Listen API
These endpoints SHALL require a session and respond 401 without one:
- `PUT /api/feed-items/{id}/listen` takes `positionSeconds` (a number from 0 to 172800, rounded down to whole seconds), `durationSeconds` (optional, `null` or a number greater than 0 and at most 172800) and `finished` (optional boolean). It creates or changes the user's record of the item and responds with the item. With `finished` `true`, the record is finished at the current time and its position is set to 0. Otherwise, a finished record becomes unfinished and gets the position. A `durationSeconds` that is left out keeps the stored duration. Other values get 400 `invalid_request`. An item without audio gets 400 `invalid_request`.
- `DELETE /api/feed-items/{id}/listen` removes the user's record of the item and responds 204, also when there was none.

An item the user can't see (see "Feed API") gets 404 `not_found`.

#### Scenario: Save a position
- **WHEN** Ann puts `{ "positionSeconds": 750.6, "durationSeconds": 3600 }` to the listen endpoint of an episode
- **THEN** the response is the item with `listen.positionSeconds` 750 and `listen.durationSeconds` 3600

#### Scenario: Mark as played
- **WHEN** Ann puts `{ "positionSeconds": 0, "finished": true }`
- **THEN** the item's `listen.finishedAt` is set and its position is 0

#### Scenario: Play again
- **WHEN** an episode's record is finished and Ann's player saves the position 30
- **THEN** the record isn't finished any more and its position is 30

#### Scenario: Invalid position
- **WHEN** Ann puts `{ "positionSeconds": -1 }`
- **THEN** the server responds 400 with `invalid_request`

#### Scenario: Item of another user's feed
- **WHEN** Bob, who doesn't subscribe to "Podcast Eins" and has no flag or record on its items, puts a position to an episode of it
- **THEN** the server responds 404 with `not_found` and no record is created

#### Scenario: Remove from history
- **WHEN** Ann deletes the listen record of an episode
- **THEN** the response is 204, and the episode is in none of "Listened", "In progress" and "Finished"

### Requirement: Listened view
`/feeds` SHALL have a fourth view "Listened" / "Gehört" after "Liked", with the sub-filters "All" / "Alle" (default), "In progress" / "Angefangen" and "Finished" / "Fertig gehört" as a group of toggles below the views, shown only in this view. "All" shows every episode with a listen record of the user, "In progress" the episodes in progress, and "Finished" the finished ones, each sorted by when the record was last changed, newest first, also when the user no longer subscribes to the episode's feed. The search and the feed filter apply as in the other views. An empty "Listened" view says "Nothing here yet" / "Noch nichts hier".

#### Scenario: Continue where I stopped
- **WHEN** Ann listened to episode A yesterday and to episode B today, and opens "Listened" with "In progress"
- **THEN** B is listed before A

#### Scenario: Finished episodes
- **WHEN** Ann finished episode A and has episode B in progress, and chooses "Finished"
- **THEN** only A is listed

#### Scenario: History of a removed feed
- **WHEN** Ann listened to an episode of "Podcast Eins" and then removed "Podcast Eins"
- **THEN** "Listened" still shows the episode

### Requirement: Episode actions
Each row of an item with audio on `/feeds` SHALL have a menu button "Episode actions for <title>" / "Episodenaktionen für <title>" with:
- "Mark as played" / "Als gehört markieren", unless the episode is finished,
- "Mark as unplayed" / "Als ungehört markieren", when the episode is finished or in progress; it keeps the record with the position 0, not finished,
- "Remove from history" / "Aus dem Verlauf entfernen", when the episode has a record.

When the episode is loaded in the player, an action SHALL also stop the player, so that its next save doesn't undo the action. The changes SHALL show in every view, widget and row without a reload. Feed widgets have no episode menu.

#### Scenario: Mark as played without listening
- **WHEN** Ann chooses "Mark as played" on an episode she never played
- **THEN** the episode is in "Finished" and its row shows "Played"

#### Scenario: Mark as unplayed
- **WHEN** Ann chooses "Mark as unplayed" on a finished episode
- **THEN** the episode is in "Listened" with "All", not in "In progress" or "Finished", and playing it starts at the beginning

#### Scenario: Action on the playing episode
- **WHEN** an episode is playing and Ann chooses "Mark as played" on its row
- **THEN** the player stops and the episode stays finished

### Requirement: Listen progress on rows
For the signed-in user, a row of an item with audio and a listen record SHALL show, on `/feeds` and in feed widgets, a bar 3 px high along the bottom edge of the row. Its filled part is the position divided by the duration, where the duration is the record's measured duration, or else the episode's duration from its feed. A finished episode SHALL show a full bar. An episode in progress without a known duration shows no bar. The bar is not announced to assistive technology. Instead, the row's line with the feed's title SHALL say "<n> min left" / "Noch <n> Min." for an episode in progress with a known duration, rounded up to whole minutes, and "Played" / "Gehört" for a finished one. While an episode is loaded in the player, its rows SHALL follow the player's position without waiting for the server. Rows of visitors show no bar and no such text.

#### Scenario: Partly listened
- **WHEN** Ann stopped a 60-minute episode at 45:00
- **THEN** its row shows a bar filled to three quarters and "15 min left"

#### Scenario: Played
- **WHEN** an episode is finished
- **THEN** its row shows a full bar and "Played"

#### Scenario: Live progress
- **WHEN** Ann plays an episode from `/feeds`
- **THEN** the bar of its row grows while it plays
