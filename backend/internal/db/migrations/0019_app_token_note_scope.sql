-- Which notes an app token sees (openspec change karakeep-text-bookmarks):
-- 'all' of the user's notes, the notes of the category root_category_id and
-- its subcategories, or 'none'. A deleted category leaves 'category' with a
-- NULL root_category_id, which never widens to all notes; deleting the
-- category also revokes the token. Like root_id it references only the ID;
-- ownership is checked on write.

ALTER TABLE app_tokens ADD COLUMN note_scope TEXT NOT NULL DEFAULT 'none' CHECK (note_scope IN ('all', 'category', 'none'));
ALTER TABLE app_tokens ADD COLUMN root_category_id TEXT REFERENCES note_categories (id) ON DELETE SET NULL;

UPDATE app_tokens SET note_scope = 'all' WHERE app = 'karakeep';

CREATE INDEX app_tokens_root_category ON app_tokens (root_category_id);
