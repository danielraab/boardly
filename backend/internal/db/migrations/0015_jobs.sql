-- Persistent queue of background jobs (openspec change
-- favourite-description-and-preview): favicon fetches, preview fetches and
-- preview conversions survive a restart. Jobs run in id order per kind.
-- dedup_key combines pending jobs of a kind (the origin of a favicon fetch,
-- the node of a preview fetch); started_at is set while a job runs.

CREATE TABLE jobs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL CHECK (kind IN ('favicon', 'preview_fetch', 'preview_convert')),
	dedup_key  TEXT,
	payload    TEXT NOT NULL,
	attempts   INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	started_at TEXT
);

CREATE UNIQUE INDEX jobs_pending_key ON jobs (kind, dedup_key) WHERE started_at IS NULL AND dedup_key IS NOT NULL;
CREATE INDEX jobs_pending ON jobs (kind, id) WHERE started_at IS NULL;
