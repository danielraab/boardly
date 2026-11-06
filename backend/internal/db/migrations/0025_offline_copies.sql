-- Offline copies of links (openspec change add-offline-copies): one row per
-- link that had a capture queued. url is the URL the current or pending
-- capture is for; results for another URL are discarded. article_id and
-- screenshot_id name files in <data dir>/offline. status is 'pending' while
-- a capture is queued or running, 'ready' after one succeeded and 'failed'
-- when the last one got nothing; error then holds the reason.

CREATE TABLE offline_copies (
	node_id       TEXT PRIMARY KEY REFERENCES favourite_nodes (id) ON DELETE CASCADE,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	url           TEXT NOT NULL,
	status        TEXT NOT NULL CHECK (status IN ('pending', 'ready', 'failed')),
	error         TEXT,
	title         TEXT,
	article_id    TEXT,
	screenshot_id TEXT,
	captured_at   TEXT,
	updated_at    TEXT NOT NULL
);

CREATE INDEX offline_copies_user ON offline_copies (user_id);

-- The jobs table gets the new kind offline_capture. SQLite can't change a
-- CHECK constraint, so the table is rebuilt; nothing refers to it.
CREATE TABLE jobs_new (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL CHECK (kind IN ('favicon', 'preview_fetch', 'preview_convert', 'offline_capture')),
	dedup_key  TEXT,
	payload    TEXT NOT NULL,
	attempts   INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	started_at TEXT
);
INSERT INTO jobs_new (id, kind, dedup_key, payload, attempts, created_at, started_at)
	SELECT id, kind, dedup_key, payload, attempts, created_at, started_at FROM jobs;
DROP TABLE jobs;
ALTER TABLE jobs_new RENAME TO jobs;
CREATE UNIQUE INDEX jobs_pending_key ON jobs (kind, dedup_key) WHERE started_at IS NULL AND dedup_key IS NOT NULL;
CREATE INDEX jobs_pending ON jobs (kind, id) WHERE started_at IS NULL;
