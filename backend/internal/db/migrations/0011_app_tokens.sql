-- App tokens for connected apps such as Karakeep clients (openspec change app-tokens).
-- Only the SHA-256 of the secret is stored; prefix is its first 12 characters.
-- root_id references only the node ID: SET NULL on a composite key would also
-- clear user_id. Ownership of the root is checked when a token is created.

CREATE TABLE app_tokens (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	app          TEXT NOT NULL CHECK (app IN ('karakeep')),
	root_id      TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL,
	token_hash   BLOB NOT NULL UNIQUE,
	prefix       TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	last_used_at TEXT,
	expires_at   TEXT,
	revoked_at   TEXT
);

CREATE INDEX app_tokens_user ON app_tokens (user_id, created_at);
CREATE INDEX app_tokens_root ON app_tokens (root_id);
