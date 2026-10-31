-- Media sources (openspec change add-media-radio). Each user's media library:
-- for now internet radio streams (kind 'stream'). kind is checked by the
-- store, not by a CHECK constraint, so that new kinds need no table rebuild.
-- position orders a user's sources and is rewritten densely on reorder.
-- An icon is a symbol or an uploaded image, never both.

CREATE TABLE media_sources (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	kind          TEXT NOT NULL,
	title         TEXT NOT NULL,
	url           TEXT NOT NULL,
	icon_symbol   TEXT,
	icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
	position      INTEGER NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,
	CHECK (icon_symbol IS NULL OR icon_image_id IS NULL)
);

CREATE INDEX media_sources_user ON media_sources (user_id, position);
