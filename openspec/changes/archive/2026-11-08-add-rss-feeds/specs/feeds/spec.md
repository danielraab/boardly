# Spec Delta

## Purpose

Lets signed-in users subscribe to RSS, Atom and JSON feeds of news and podcasts, stores every feed and item once per server, keeps them up to date in the background, and lets each user flag items to read later or as liked.

## ADDED Requirements

### Requirement: Feeds page
`/feeds` SHALL require a signed-in user and show the heading "Feeds" / "Feeds", an "Add feed" / "Feed hinzufügen" button, the views "Latest" / "Neueste" (default), "Read later" / "Später lesen" and "Liked" / "Gefällt mir", the items of the chosen view newest first, and a "Manage feeds" / "Feeds verwalten" action that lists the user's feeds in their order. Signed-out visitors SHALL be sent to `/login`. "Latest" shows the items of the user's subscribed feeds, sorted by item time. "Read later" and "Liked" show the user's items with that flag, sorted by when the flag was set, also when the user no longer subscribes to the item's feed. When the user has no feeds, the page SHALL show "No feeds yet" / "Noch keine Feeds" with a hint to add one. An empty "Read later" or "Liked" view SHALL say "Nothing here yet" / "Noch nichts hier". The items load in pages of 50 with a "Load more" / "Mehr laden" button. The document title is "Feeds - <app title>".

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
- `GET /api/feed-items?view=<latest|read_later|liked>&feedId=<subscription ID>&limit=<1-100>&cursor=<cursor>` responds with `items` and `nextCursor` (or `null`). `view` defaults to `latest`, `limit` to 50, and `feedId` is optional. An unknown view, a limit outside 1–100 or an invalid cursor gets 400 `invalid_request`. Each item has `id`, `title`, `link` (or `null`), `summary` (or `null`), `publishedAt`, `audio` (`{ "url", "type", "durationSeconds" }` or `null`), `feed` (`{ "title", "subscriptionId", "icon", "faviconId" }`, with `subscriptionId` `null` when the user doesn't subscribe), `readLaterAt` (or `null`) and `likedAt` (or `null`).
- `PATCH /api/feed-items/{id}` takes `readLater` and `liked`, each a boolean and optional, and responds with the item.

Requests for another user's subscription, or for an unknown one, get 404 `not_found`. An item the user neither subscribes to nor has flagged gets 404 `not_found`.

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
For each feed entry, the server SHALL store a title, an optional link, an optional summary, a time and optional audio. Only the newest `BOARDLY_FEED_LATEST_ITEMS` entries of a fetched document SHALL be stored. An entry with neither a title nor a summary is left out, and an entry without a title takes the first 100 characters of its summary. The title and the summary SHALL be plain text, with HTML removed and entities decoded, cut to 500 characters. A link that isn't an absolute `http` or `https` URL SHALL be dropped. The time is the entry's published or updated time, or the time it was first seen when it has none, and never later than the time it was first seen. An entry has audio when it has an enclosure of an audio type, or of a generic type with a common audio file extension, with an `http` or `https` URL. An `http` audio URL SHALL be stored as `https`. Images in entries SHALL NOT be stored or shown.

#### Scenario: HTML in the summary
- **WHEN** an entry's description is `<p>Hello <b>world</b></p><script>x()</script>`
- **THEN** its summary is "Hello world"

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
After a refresh and at least once per hour, the server SHALL delete the items of a feed that are outside its newest `BOARDLY_FEED_LATEST_ITEMS` items, unless at least one user has a flag on the item. The newest items are chosen by item time among all items of the feed, flagged or not. An item whose last flag is removed SHALL be deleted by the next cleanup when it is outside the newest items. A feed with no subscribers and no flagged items SHALL be deleted with its items. An account's deletion removes its subscriptions and flags and leaves the shared feeds and items to this cleanup.

#### Scenario: Old item removed
- **WHEN** 100 items are kept per feed and a feed has 101 unflagged items
- **THEN** the oldest item is deleted

#### Scenario: Flagged item kept
- **WHEN** an item that Ann flagged is older than the newest 100 items of its feed
- **THEN** the item is kept and stays in her view

#### Scenario: Flag of another user
- **WHEN** Ann removes her like from an old item that Bob has marked "Read later"
- **THEN** the item is kept

#### Scenario: Last flag removed
- **WHEN** Ann removes the only flag of an item outside the newest 100 items
- **THEN** the item is deleted by the next cleanup

#### Scenario: Old items don't come back
- **WHEN** a feed's document lists 500 entries and 100 items are kept per feed
- **THEN** only the newest 100 entries are stored, and refreshing doesn't add and delete the older ones

#### Scenario: Unsubscribed feed with saved items
- **WHEN** Ann, the only subscriber, removes a feed in which she liked an item
- **THEN** the feed and that item stay, the other items are deleted by the cleanup, and the feed isn't fetched

#### Scenario: Unsubscribed feed without saved items
- **WHEN** the only subscriber removes a feed with no flagged items
- **THEN** the cleanup deletes the feed and its items

#### Scenario: Limit lowered
- **WHEN** `BOARDLY_FEED_LATEST_ITEMS` is lowered from 100 to 30 and the server restarts
- **THEN** the next cleanup leaves the newest 30 and the flagged items of each feed

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

#### Scenario: News item
- **WHEN** Ann activates a news item
- **THEN** its link opens in a new tab and `/feeds` stays open

#### Scenario: Episode
- **WHEN** Ann activates the play button of an episode
- **THEN** the media player plays it and the button becomes a pause button

### Requirement: Podcasts are separate from the media library
Feeds and their episodes SHALL NOT appear on `/media`, in `GET /api/media` or in the media widget's source list. They share only the media player.

#### Scenario: Podcast not on the media page
- **WHEN** Ann subscribes to a podcast and opens `/media`
- **THEN** the page shows only her stations
