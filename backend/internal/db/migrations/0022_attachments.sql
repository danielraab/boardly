-- Note attachments (openspec change add-note-attachments). Files a user
-- uploads to notes, stored under <data dir>/attachments, separate from images.
-- deleted_at is the deletion mark the cleanup sets on unused attachments;
-- keep stops the cleanup from deleting one.
--
-- note_attachments is derived from a note's content on every save. position
-- is where the attachment first appears in the content. Deleting an
-- attachment that a note refers to fails: the store checks it first, and the
-- foreign key is the safety net. It is deferred so that deleting a user, which
-- cascades to their notes and their attachments, is checked only at the end.

CREATE TABLE attachments (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	content_type TEXT NOT NULL,
	size         INTEGER NOT NULL CHECK (size > 0),
	created_at   TEXT NOT NULL,
	deleted_at   TEXT,
	keep         INTEGER NOT NULL DEFAULT 0 CHECK (keep IN (0, 1))
);

CREATE INDEX attachments_user ON attachments (user_id, created_at);

CREATE TABLE note_attachments (
	note_id       TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
	attachment_id TEXT NOT NULL REFERENCES attachments (id) DEFERRABLE INITIALLY DEFERRED,
	position      INTEGER NOT NULL,
	PRIMARY KEY (note_id, attachment_id)
);

CREATE INDEX note_attachments_attachment ON note_attachments (attachment_id);
