-- App tokens can also be for Linkwarden clients (openspec change linkwarden-api).
-- SQLite can't change a CHECK constraint, so the table is rebuilt. No other
-- table references app_tokens.

CREATE TABLE app_tokens_new (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	app          TEXT NOT NULL CHECK (app IN ('karakeep', 'linkwarden')),
	root_id      TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL,
	token_hash   BLOB NOT NULL UNIQUE,
	prefix       TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	last_used_at TEXT,
	expires_at   TEXT,
	revoked_at   TEXT
);

INSERT INTO app_tokens_new SELECT id, user_id, name, app, root_id, token_hash, prefix, created_at, last_used_at, expires_at, revoked_at FROM app_tokens;
DROP TABLE app_tokens;
ALTER TABLE app_tokens_new RENAME TO app_tokens;

CREATE INDEX app_tokens_user ON app_tokens (user_id, created_at);
CREATE INDEX app_tokens_root ON app_tokens (root_id);
