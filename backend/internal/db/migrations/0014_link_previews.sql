-- Descriptions and preview images of links (openspec change
-- favourite-description-and-preview). Previews are images with the source
-- 'preview' that belong to the link's owner.
--
-- SQLite can't change a CHECK constraint, so images is rebuilt. favicons
-- references images, and with foreign keys on, dropping images sets every
-- favicons.image_id to NULL. The pairs are kept in a temporary table and put
-- back after the rebuild.

CREATE TEMP TABLE favicon_images AS SELECT origin, image_id FROM favicons WHERE image_id IS NOT NULL;

CREATE TABLE images_new (
	id           TEXT PRIMARY KEY,
	source       TEXT NOT NULL CHECK (source IN ('favicon', 'upload', 'preview')),
	user_id      TEXT REFERENCES users (id) ON DELETE CASCADE,
	content_type TEXT NOT NULL,
	size         INTEGER NOT NULL CHECK (size > 0),
	created_at   TEXT NOT NULL,
	CHECK ((source IN ('upload', 'preview')) = (user_id IS NOT NULL))
);

INSERT INTO images_new SELECT id, source, user_id, content_type, size, created_at FROM images;
DROP TABLE images;
ALTER TABLE images_new RENAME TO images;

CREATE INDEX images_upload_created ON images (created_at) WHERE source = 'upload';
CREATE INDEX images_preview_created ON images (created_at) WHERE source = 'preview';

UPDATE favicons SET image_id = (SELECT f.image_id FROM favicon_images AS f WHERE f.origin = favicons.origin)
WHERE origin IN (SELECT origin FROM favicon_images);
DROP TABLE favicon_images;

ALTER TABLE favourite_nodes ADD COLUMN description TEXT CHECK (description IS NULL OR kind = 'link');
ALTER TABLE favourite_nodes ADD COLUMN preview_image_id TEXT REFERENCES images (id) ON DELETE SET NULL
	CHECK (preview_image_id IS NULL OR kind = 'link');

CREATE INDEX favourite_nodes_preview ON favourite_nodes (preview_image_id) WHERE preview_image_id IS NOT NULL;
