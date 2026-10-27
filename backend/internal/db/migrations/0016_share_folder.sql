-- Default folder of the "Save link" page (openspec change share-default-folder).
-- NULL means the root folder. Like app_tokens.root_id it references only the
-- node ID, so deleting the folder clears it; ownership is checked on write.

ALTER TABLE users ADD COLUMN share_folder_id TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL;
