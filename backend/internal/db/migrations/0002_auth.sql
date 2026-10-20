-- Accounts, sessions and magic-link tokens (openspec change magic-link-auth).
-- Timestamps are RFC 3339 UTC text; secrets are stored only as SHA-256 hashes.

CREATE TABLE users (
	id         TEXT PRIMARY KEY,
	email      TEXT NOT NULL UNIQUE,
	name       TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE sessions (
	id_hash      BLOB PRIMARY KEY,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	auth_method  TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	last_used_at TEXT NOT NULL,
	expires_at   TEXT NOT NULL,
	user_agent   TEXT NOT NULL
);

CREATE INDEX sessions_user_id ON sessions (user_id);

CREATE TABLE magic_link_tokens (
	token_hash BLOB PRIMARY KEY,
	email      TEXT NOT NULL,
	locale     TEXT NOT NULL,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	used_at    TEXT,
	request_ip TEXT NOT NULL
);

CREATE INDEX magic_link_tokens_email_created ON magic_link_tokens (email, created_at);
