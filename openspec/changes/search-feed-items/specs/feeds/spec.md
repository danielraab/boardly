## MODIFIED Requirements

### Requirement: Feeds page
`/feeds` SHALL require a signed-in user and show the heading "Feeds" / "Feeds", an "Add feed" / "Feed hinzufügen" button, the views "Latest" / "Neueste" (default), "Read later" / "Später lesen" and "Liked" / "Gefällt mir", the items of the chosen view newest first, and a "Manage feeds" / "Feeds verwalten" action that lists the user's feeds in their order. Signed-out visitors SHALL be sent to `/login`. "Latest" shows the items of the user's subscribed feeds, sorted by item time. "Read later" and "Liked" show the user's items with that flag, sorted by when the flag was set, also when the user no longer subscribes to the item's feed. When the user has no feeds, the page SHALL show "No feeds yet" / "Noch keine Feeds" with a hint to add one. An empty "Read later" or "Liked" view SHALL say "Nothing here yet" / "Noch nichts hier". The items load in pages of 50 with a "Load more" / "Mehr laden" button. The document title is "Feeds - <app title>".

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

### Requirement: Feed API
The web app SHALL manage feeds through these endpoints, all of which require a session and respond 401 without one:
- `GET /api/feeds` lists the user's subscriptions in their order, each with `id`, `feedId`, `url`, `title` (the custom title or else the feed's title), `customTitle` (or `null`), `siteUrl` (or `null`), `icon` (as for notes, or `null`), `faviconId` (or `null`), `position`, `lastFetchedAt` (or `null`) and `failing`.
- `POST /api/feeds` subscribes from `url` (see "Adding a feed"), adds the subscription at the end of the list and responds 201 with it.
- `PATCH /api/feeds/{id}` changes `customTitle` (`null` or empty removes it) or `icon` (`null` removes it) and responds with the subscription. A custom title longer than 200 characters after trimming gets 400 `invalid_title`.
- `PUT /api/feeds/order` takes the IDs of all the user's subscriptions in their new order. A list that isn't exactly the user's subscription IDs gets 400 `invalid_order`.
- `DELETE /api/feeds/{id}` removes the subscription and responds 204.
- `GET /api/feed-items?view=<latest|read_later|liked>&feedId=<subscription ID>&q=<search>&limit=<1-100>&cursor=<cursor>` responds with `items` and `nextCursor` (or `null`). `view` defaults to `latest`, `limit` to 50, and `feedId` and `q` are optional. An unknown view, a limit outside 1–100, a `q` longer than 200 characters or an invalid cursor gets 400 `invalid_request`. Each item has `id`, `title`, `link` (or `null`), `summary` (or `null`), `publishedAt`, `audio` (`{ "url", "type", "durationSeconds" }` or `null`), `feed` (`{ "title", "subscriptionId", "icon", "faviconId" }`, with `subscriptionId` `null` when the user doesn't subscribe), `readLaterAt` (or `null`) and `likedAt` (or `null`).
- `PATCH /api/feed-items/{id}` takes `readLater` and `liked`, each a boolean and optional, and responds with the item.

`feedId` limits the items of any view to the feed of that subscription of the user. A `feedId` that isn't one of the user's subscriptions gives no items. `q` is split into words at whitespace, and an item matches when every word appears in its title or its summary, ignoring case for all letters, not only ASCII. A `q` that is empty or only has whitespace matches every item. Search doesn't change the order of a view, and paging with `cursor` keeps `q` and `feedId`.

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
