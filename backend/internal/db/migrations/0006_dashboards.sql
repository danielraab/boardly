-- User slugs and dashboards (openspec change dashboard-management).
-- users.slug stays nullable here because adding NOT NULL would need a rebuild of
-- users; the store always sets it. Existing users get 10 random hex characters.

ALTER TABLE users ADD COLUMN slug TEXT;
UPDATE users SET slug = lower(hex(randomblob(5)));
CREATE UNIQUE INDEX users_slug ON users (slug);

-- At most one default per user (partial unique index); none is fine.
CREATE TABLE dashboards (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	slug       TEXT NOT NULL CHECK (slug = lower(slug)),
	title      TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1)),
	is_public  INTEGER NOT NULL DEFAULT 0 CHECK (is_public IN (0, 1)),
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE (user_id, slug)
);

CREATE UNIQUE INDEX dashboards_default ON dashboards (user_id) WHERE is_default = 1;
