-- Feeds (openspec change add-rss-feeds). A feed and its items exist once per
-- server, however many users have them; the two UNIQUE constraints enforce
-- that. Subscriptions and flags are what belongs to one user.

CREATE TABLE feeds (
	id              TEXT PRIMARY KEY,
	url             TEXT NOT NULL UNIQUE,
	title           TEXT NOT NULL,
	site_url        TEXT,
	etag            TEXT,
	last_modified   TEXT,
	-- Set by the first successful fetch; a feed created by an import has none.
	last_success_at TEXT,
	last_fetched_at TEXT,
	next_fetch_at   TEXT NOT NULL,
	failures        INTEGER NOT NULL DEFAULT 0,
	created_at      TEXT NOT NULL
);

CREATE INDEX feeds_next_fetch ON feeds (next_fetch_at);

-- imported_by is set on an item that came from an import and no fetch has
-- confirmed yet: only that user can see it.
CREATE TABLE feed_items (
	id            TEXT PRIMARY KEY,
	feed_id       TEXT NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
	identity      TEXT NOT NULL,
	title         TEXT NOT NULL,
	link          TEXT,
	summary       TEXT,
	item_time     TEXT NOT NULL,
	first_seen_at TEXT NOT NULL,
	audio_url     TEXT,
	audio_type    TEXT,
	audio_seconds INTEGER,
	imported_by   TEXT REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE (feed_id, identity)
);

CREATE INDEX feed_items_time ON feed_items (feed_id, item_time DESC, id DESC);

CREATE TABLE feed_subscriptions (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	feed_id       TEXT NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
	custom_title  TEXT,
	icon_symbol   TEXT,
	icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
	position      INTEGER NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,
	UNIQUE (user_id, feed_id),
	CHECK (icon_symbol IS NULL OR icon_image_id IS NULL)
);

CREATE INDEX feed_subscriptions_user ON feed_subscriptions (user_id, position);
CREATE INDEX feed_subscriptions_feed ON feed_subscriptions (feed_id);

-- A row exists only while at least one of the flags is set.
CREATE TABLE feed_item_flags (
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	item_id       TEXT NOT NULL REFERENCES feed_items (id) ON DELETE CASCADE,
	read_later_at TEXT,
	liked_at      TEXT,
	PRIMARY KEY (user_id, item_id),
	CHECK (read_later_at IS NOT NULL OR liked_at IS NOT NULL)
);

CREATE INDEX feed_item_flags_item ON feed_item_flags (item_id);
