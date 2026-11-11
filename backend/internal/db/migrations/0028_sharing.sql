-- Shares, groups and email invitations (openspec change
-- add-sharing-groups-dashboards). SQLite can't change a CHECK without
-- rebuilding the table, so shares and invitations have the columns of every
-- resource type that later changes will share, although only dashboards and
-- groups are used for now.

-- GROUPS is an SQL keyword, hence the names.
CREATE TABLE user_groups (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE user_group_members (
	group_id  TEXT NOT NULL REFERENCES user_groups (id) ON DELETE CASCADE,
	user_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	role      TEXT NOT NULL CHECK (role IN ('admin', 'member')),
	joined_at TEXT NOT NULL,
	PRIMARY KEY (group_id, user_id)
);

CREATE INDEX user_group_members_user ON user_group_members (user_id);

-- owner_id repeats the owner of the resource. The store sets it from the
-- resource, never from the request.
CREATE TABLE shares (
	id                  TEXT PRIMARY KEY,
	owner_id            TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	dashboard_id        TEXT REFERENCES dashboards (id) ON DELETE CASCADE,
	note_id             TEXT REFERENCES notes (id) ON DELETE CASCADE,
	note_category_id    TEXT REFERENCES note_categories (id) ON DELETE CASCADE,
	favourite_folder_id TEXT REFERENCES favourite_nodes (id) ON DELETE CASCADE,
	user_id             TEXT REFERENCES users (id) ON DELETE CASCADE,
	group_id            TEXT REFERENCES user_groups (id) ON DELETE CASCADE,
	level               TEXT NOT NULL CHECK (level IN ('read', 'edit')),
	created_at          TEXT NOT NULL,
	CHECK ((dashboard_id IS NOT NULL) + (note_id IS NOT NULL) + (note_category_id IS NOT NULL) + (favourite_folder_id IS NOT NULL) = 1),
	CHECK ((user_id IS NOT NULL) + (group_id IS NOT NULL) = 1)
);

CREATE UNIQUE INDEX shares_dashboard_user ON shares (dashboard_id, user_id) WHERE dashboard_id IS NOT NULL AND user_id IS NOT NULL;
CREATE UNIQUE INDEX shares_dashboard_group ON shares (dashboard_id, group_id) WHERE dashboard_id IS NOT NULL AND group_id IS NOT NULL;
CREATE INDEX shares_user ON shares (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX shares_group ON shares (group_id) WHERE group_id IS NOT NULL;
CREATE INDEX shares_owner ON shares (owner_id);

-- token_hash stays NULL until the background job has mailed a link to an
-- existing account, so an invitation without one can only be accepted from
-- the signed-in user's list.
CREATE TABLE invitations (
	id                  TEXT PRIMARY KEY,
	inviter_id          TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	email               TEXT NOT NULL,
	token_hash          TEXT,
	locale              TEXT NOT NULL,
	dashboard_id        TEXT REFERENCES dashboards (id) ON DELETE CASCADE,
	note_id             TEXT REFERENCES notes (id) ON DELETE CASCADE,
	note_category_id    TEXT REFERENCES note_categories (id) ON DELETE CASCADE,
	favourite_folder_id TEXT REFERENCES favourite_nodes (id) ON DELETE CASCADE,
	group_id            TEXT REFERENCES user_groups (id) ON DELETE CASCADE,
	level               TEXT CHECK (level IN ('read', 'edit')),
	role                TEXT CHECK (role IN ('admin', 'member')),
	created_at          TEXT NOT NULL,
	expires_at          TEXT NOT NULL,
	CHECK ((dashboard_id IS NOT NULL) + (note_id IS NOT NULL) + (note_category_id IS NOT NULL) + (favourite_folder_id IS NOT NULL) + (group_id IS NOT NULL) = 1),
	CHECK ((group_id IS NOT NULL) = (role IS NOT NULL)),
	CHECK ((group_id IS NULL) = (level IS NOT NULL))
);

CREATE UNIQUE INDEX invitations_dashboard_email ON invitations (dashboard_id, email) WHERE dashboard_id IS NOT NULL;
CREATE UNIQUE INDEX invitations_group_email ON invitations (group_id, email) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX invitations_token ON invitations (token_hash) WHERE token_hash IS NOT NULL;
CREATE INDEX invitations_email ON invitations (email);
CREATE INDEX invitations_expires ON invitations (expires_at);

-- When an address without an account last got a notice. This is separate from
-- the invitations, so revoking and repeating one can't send more notices.
CREATE TABLE invitation_notices (
	email   TEXT PRIMARY KEY,
	sent_at TEXT NOT NULL
);

-- The jobs table gets the new kind invitation_mail. SQLite can't change a
-- CHECK constraint, so the table is rebuilt; nothing refers to it.
CREATE TABLE jobs_new (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL CHECK (kind IN ('favicon', 'preview_fetch', 'preview_convert', 'offline_capture', 'invitation_mail')),
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
