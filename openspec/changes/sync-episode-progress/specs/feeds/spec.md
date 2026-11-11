## ADDED Requirements

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

## MODIFIED Requirements

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
