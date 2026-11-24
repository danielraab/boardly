## ADDED Requirements

### Requirement: Pausing feeds
A user SHALL be able to pause and resume each of their subscriptions. Pausing belongs to the user's subscription and doesn't change the feed for other subscribers. A paused feed keeps its position, custom title and icon. Its items SHALL be left out of "Latest", while items the user flagged or has a listen record of stay in "Read later", "Liked" and the listen views. A new subscription is not paused.

#### Scenario: Pause a feed
- **WHEN** Ann subscribes to "Heise" and "Podcast Eins" and pauses "Heise"
- **THEN** "Latest" shows only the items of "Podcast Eins"

#### Scenario: Resume a feed
- **WHEN** Ann resumes "Heise"
- **THEN** "Latest" shows the items of both feeds again, and "Heise" is at the same position in her list as before

#### Scenario: Flagged items stay
- **WHEN** Ann liked an item of "Heise" and then pauses "Heise"
- **THEN** "Liked" still shows the item, and "Latest" doesn't

#### Scenario: Listened episodes stay
- **WHEN** Ann has an episode of "Podcast Eins" in progress and pauses "Podcast Eins"
- **THEN** the episode is still in "Listened" and in "Continue listening"

#### Scenario: Other subscribers
- **WHEN** Ann and Bob subscribe to "Heise" and Ann pauses it
- **THEN** Bob's "Latest" still shows the items of "Heise"

#### Scenario: Add a paused feed again
- **WHEN** Ann adds the URL of a feed that she has paused
- **THEN** the server responds 409 with `already_subscribed` and the feed stays paused

### Requirement: Paused feeds stay up to date
Pausing SHALL only hide a feed. A paused subscription SHALL count as a subscriber for "Background refresh" and "Item cleanup", so a paused feed is still refreshed and its items are kept and deleted as for any other feed. A resumed feed SHALL show its current items without waiting for a refresh.

#### Scenario: Only subscriber paused
- **WHEN** Ann is the only subscriber of a feed and pauses it
- **THEN** the feed is still refreshed, and the cleanup doesn't delete it

#### Scenario: Resume shows new items
- **WHEN** a paused feed got a new item while it was paused and Ann resumes it
- **THEN** "Latest" shows the new item right away

## MODIFIED Requirements

### Requirement: Feeds page
`/feeds` SHALL require a signed-in user and show the heading "Feeds" / "Feeds", an "Add feed" / "Feed hinzufügen" button, the views "Latest" / "Neueste" (default), "Read later" / "Später lesen", "Liked" / "Gefällt mir" and "Listened" / "Gehört" (see "Listened view"), the items of the chosen view newest first, and a "Manage feeds" / "Feeds verwalten" action that lists the user's feeds in their order. Signed-out visitors SHALL be sent to `/login`. "Latest" shows the items of the user's subscribed feeds that aren't paused, sorted by item time. "Read later" and "Liked" show the user's items with that flag, sorted by when the flag was set, also when the user no longer subscribes to the item's feed or has paused it. When the user has no feeds, the page SHALL show "No feeds yet" / "Noch keine Feeds" with a hint to add one. An empty "Read later", "Liked" or "Listened" view SHALL say "Nothing here yet" / "Noch nichts hier". The items load in pages of 50 with a "Load more" / "Mehr laden" button. The document title is "Feeds - <app title>".

While the user has feeds, the page SHALL show, above the items, a search field "Search items" / "Einträge durchsuchen" and a feed filter "Feed" / "Feed" with "All feeds" / "Alle Feeds" (default) and the user's feeds that aren't paused, in their order. Both apply to the items of the current view, in the order of that view, as described for `q` and `feedId` in "Feed API". The search SHALL start shortly after typing stops, without a button, and a search field that is empty or only has spaces shows all items of the view. Switching the view SHALL keep the search and the filter. A view without items for the search or the filter SHALL say "No matching items" / "Keine passenden Einträge". The search field SHALL have a clear button "Clear search" / "Suche löschen". A filter whose feed the user removes or pauses SHALL fall back to "All feeds".

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

#### Scenario: Paused feed not in the filter
- **WHEN** Ann has paused "Heise" and opens the feed filter
- **THEN** the filter offers "All feeds" and "Podcast Eins", and not "Heise"

#### Scenario: Filtered feed paused
- **WHEN** Ann filters by "Heise" and then pauses "Heise" in "Manage feeds"
- **THEN** the filter falls back to "All feeds"

### Requirement: Feed API
The web app SHALL manage feeds through these endpoints, all of which require a session and respond 401 without one:
- `GET /api/feeds` lists the user's subscriptions in their order, each with `id`, `feedId`, `url`, `title` (the custom title or else the feed's title), `customTitle` (or `null`), `siteUrl` (or `null`), `icon` (as for notes, or `null`), `faviconId` (or `null`), `position`, `lastFetchedAt` (or `null`), `failing` and `paused`.
- `POST /api/feeds` subscribes from `url` (see "Adding a feed"), adds the subscription at the end of the list, not paused, and responds 201 with it.
- `PATCH /api/feeds/{id}` changes `customTitle` (`null` or empty removes it), `icon` (`null` removes it) or `paused` (a boolean) and responds with the subscription. A custom title longer than 200 characters after trimming gets 400 `invalid_title`. A `paused` that isn't a boolean gets 400 `invalid_request`.
- `PUT /api/feeds/order` takes the IDs of all the user's subscriptions in their new order. A list that isn't exactly the user's subscription IDs gets 400 `invalid_order`.
- `DELETE /api/feeds/{id}` removes the subscription and responds 204.
- `GET /api/feed-items?view=<latest|read_later|liked|listened|in_progress|finished>&feedId=<subscription ID>&q=<search>&limit=<1-100>&cursor=<cursor>` responds with `items` and `nextCursor` (or `null`). `view` defaults to `latest`, `limit` to 50, and `feedId` and `q` are optional. An unknown view, a limit outside 1–100, a `q` longer than 200 characters or an invalid cursor gets 400 `invalid_request`. Each item has `id`, `title`, `link` (or `null`), `summary` (or `null`), `publishedAt`, `audio` (`{ "url", "type", "durationSeconds" }` or `null`), `feed` (`{ "title", "subscriptionId", "icon", "faviconId" }`, with `subscriptionId` `null` when the user doesn't subscribe), `readLaterAt` (or `null`), `likedAt` (or `null`) and `listen` (`{ "positionSeconds", "durationSeconds", "startedAt", "updatedAt", "finishedAt" }` with `durationSeconds` and `finishedAt` possibly `null`, or `null` without a record).
- `PATCH /api/feed-items/{id}` takes `readLater` and `liked`, each a boolean and optional, and responds with the item.
- `PUT` and `DELETE /api/feed-items/{id}/listen` change the listen record (see "Listen API").

The view `latest` leaves out the items of paused subscriptions, also when `feedId` names one. The views `listened`, `in_progress` and `finished` show the user's items with a listen record, an unfinished record with a position greater than 0, and a finished record, sorted by the record's `updatedAt`, newest first. `feedId` limits the items of any view to the feed of that subscription of the user. A `feedId` that isn't one of the user's subscriptions gives no items. `q` is split into words at whitespace, and an item matches when every word appears in its title or its summary, ignoring case for all letters, not only ASCII. A `q` that is empty or only has whitespace matches every item. Search doesn't change the order of a view, and paging with `cursor` keeps `q` and `feedId`.

Requests for another user's subscription, or for an unknown one, get 404 `not_found`. An item the user neither subscribes to nor has flagged nor has a listen record of gets 404 `not_found`. An item of a paused subscription can still be flagged and listened to.

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

#### Scenario: Pause through the API
- **WHEN** Ann sends `PATCH /api/feeds/<Heise>` with `{ "paused": true }`
- **THEN** the response has `paused` `true`, and `GET /api/feed-items?view=latest` has no items of "Heise"

#### Scenario: Paused feed as filter of Latest
- **WHEN** Ann has paused "Heise" and requests `view=latest&feedId=<Heise>`
- **THEN** the response has no items

#### Scenario: Paused feed as filter of a saved view
- **WHEN** Ann has paused "Heise", liked one of its items and requests `view=liked&feedId=<Heise>`
- **THEN** the liked item is returned, with its `feed.subscriptionId`

#### Scenario: Invalid paused value
- **WHEN** Ann sends `PATCH /api/feeds/<Heise>` with `{ "paused": "yes" }`
- **THEN** the server responds 400 with `invalid_request` and the subscription is unchanged

### Requirement: Manage feeds
"Manage feeds" SHALL list each feed with its icon, its title and its URL's host, and a menu with "Edit" / "Bearbeiten", "Pause feed" / "Feed pausieren" (or "Resume feed" / "Feed fortsetzen" for a paused feed) and "Remove" / "Entfernen". The icon is the user's chosen icon, otherwise the shared favicon of the feed's site, otherwise a feed icon. "Edit" opens a dialog with the title and the icon, chosen with the same icon chooser as notes. Pausing and resuming take effect without confirmation, and the items on the page update without a reload. A paused feed SHALL be shown dimmed with the label "Paused" / "Pausiert". Removing asks for confirmation, and the dialog SHALL say that items flagged "Read later" or "Liked" are kept. A feed reported as `failing` SHALL show "Can't be updated right now" / "Kann gerade nicht aktualisiert werden". Feeds SHALL be reorderable by dragging a handle, also with the keyboard as on the favourites page, paused ones included.

#### Scenario: Rename
- **WHEN** Ann edits "Heise", changes the title to "News" and confirms
- **THEN** her list and her items show "News"

#### Scenario: Failing feed
- **WHEN** one of Ann's feeds is `failing`
- **THEN** it shows "Can't be updated right now" and its items are still shown

#### Scenario: Pause from the menu
- **WHEN** Ann chooses "Pause feed" in the menu of "Heise"
- **THEN** "Heise" is shown dimmed with "Paused", its menu offers "Resume feed", and the "Latest" view no longer shows its items

#### Scenario: Resume from the menu
- **WHEN** Ann chooses "Resume feed" in the menu of the paused "Heise"
- **THEN** "Heise" is no longer dimmed, and "Latest" shows its items again

#### Scenario: Reorder a paused feed
- **WHEN** Ann drags the paused "Heise" above "Podcast Eins"
- **THEN** the order is saved and "Heise" stays paused
