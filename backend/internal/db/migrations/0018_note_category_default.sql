-- Default category for new notes, used by the share page (openspec change
-- notes-in-import-and-share). NULL means no category. Like share_folder_id it
-- references only the category ID, so deleting the category clears it;
-- ownership is checked on write.

ALTER TABLE users ADD COLUMN note_category_id TEXT REFERENCES note_categories (id) ON DELETE SET NULL;
