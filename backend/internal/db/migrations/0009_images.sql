-- Images and the favicon cache (openspec change image-store).
-- The image bytes live in <data dir>/images/<id>; this table only describes
-- them. Uploads belong to a user, favicons are shared by all users.

CREATE TABLE images (
	id           TEXT PRIMARY KEY,
	source       TEXT NOT NULL CHECK (source IN ('favicon', 'upload')),
	user_id      TEXT REFERENCES users (id) ON DELETE CASCADE,
	content_type TEXT NOT NULL,
	size         INTEGER NOT NULL CHECK (size > 0),
	created_at   TEXT NOT NULL,
	CHECK ((source = 'upload') = (user_id IS NOT NULL))
);

CREATE INDEX images_upload_created ON images (created_at) WHERE source = 'upload';

-- One row per origin (scheme://host[:port]) that was fetched. image_id is
-- NULL while no favicon is known; failed_at is the last failed attempt.
CREATE TABLE favicons (
	origin     TEXT PRIMARY KEY,
	image_id   TEXT REFERENCES images (id) ON DELETE SET NULL,
	fetched_at TEXT,
	failed_at  TEXT
);
