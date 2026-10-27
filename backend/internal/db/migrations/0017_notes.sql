-- Notes and note categories (openspec change add-notes).
-- Categories nest like folders; the composite foreign key keeps a parent
-- inside its owner's categories. It has no ON DELETE action: the store moves
-- a category's children to its parent before deleting it. A note's category
-- belongs to the same user, which the store checks; deleting the category
-- makes the note uncategorized. content is the note's JSON document.
-- An icon is a symbol or an uploaded image, never both.

CREATE TABLE note_categories (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	parent_id     TEXT,
	title         TEXT NOT NULL,
	icon_symbol   TEXT,
	icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
	color         TEXT,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,
	UNIQUE (id, user_id),
	FOREIGN KEY (parent_id, user_id) REFERENCES note_categories (id, user_id),
	CHECK (icon_symbol IS NULL OR icon_image_id IS NULL)
);

CREATE INDEX note_categories_user ON note_categories (user_id);
CREATE INDEX note_categories_parent ON note_categories (parent_id);

CREATE TABLE notes (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	category_id   TEXT REFERENCES note_categories (id) ON DELETE SET NULL,
	title         TEXT,
	icon_symbol   TEXT,
	icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
	color         TEXT,
	content       TEXT NOT NULL,
	version       INTEGER NOT NULL DEFAULT 1,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,
	CHECK (icon_symbol IS NULL OR icon_image_id IS NULL)
);

CREATE INDEX notes_user_updated ON notes (user_id, updated_at DESC, id);
CREATE INDEX notes_category ON notes (category_id);
