-- Listen history (openspec change sync-episode-progress): where a user is in
-- an episode, and whether they finished it. Positions are whole seconds.

CREATE TABLE feed_item_listens (
	user_id          TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	item_id          TEXT NOT NULL REFERENCES feed_items (id) ON DELETE CASCADE,
	position_seconds INTEGER NOT NULL,
	duration_seconds INTEGER,
	started_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL,
	finished_at      TEXT,
	PRIMARY KEY (user_id, item_id)
);

CREATE INDEX feed_item_listens_item ON feed_item_listens (item_id);
CREATE INDEX feed_item_listens_user ON feed_item_listens (user_id, updated_at DESC);
