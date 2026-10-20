-- Favourites tree and tags (openspec change favourites-tree).
-- Each user has one root folder (parent_id IS NULL). The composite foreign key
-- keeps every node inside its owner's tree.

CREATE TABLE favourite_nodes (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	parent_id  TEXT,
	kind       TEXT NOT NULL CHECK (kind IN ('folder', 'link')),
	position   INTEGER NOT NULL,
	title      TEXT,
	url        TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE (id, user_id),
	FOREIGN KEY (parent_id, user_id) REFERENCES favourite_nodes (id, user_id) ON DELETE CASCADE,
	CHECK ((kind = 'folder' AND url IS NULL) OR (kind = 'link' AND url IS NOT NULL AND parent_id IS NOT NULL)),
	CHECK (parent_id IS NOT NULL OR title IS NULL)
);

CREATE UNIQUE INDEX favourite_nodes_root ON favourite_nodes (user_id) WHERE parent_id IS NULL;
CREATE INDEX favourite_nodes_parent ON favourite_nodes (parent_id, position);

-- name_key is the lower-cased name; it makes names unique regardless of case,
-- also for non-ASCII letters, which NOCASE doesn't fold.
CREATE TABLE tags (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name       TEXT NOT NULL,
	name_key   TEXT NOT NULL,
	created_at TEXT NOT NULL,
	UNIQUE (user_id, name_key)
);

CREATE TABLE favourite_node_tags (
	node_id TEXT NOT NULL REFERENCES favourite_nodes (id) ON DELETE CASCADE,
	tag_id  TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
	PRIMARY KEY (node_id, tag_id)
);

CREATE INDEX favourite_node_tags_tag_id ON favourite_node_tags (tag_id);
