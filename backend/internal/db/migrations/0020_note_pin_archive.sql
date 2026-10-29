-- Pinned and archived notes (openspec change note-pin-archive). Both are
-- independent flags; existing notes are neither pinned nor archived.

ALTER TABLE notes ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1));
ALTER TABLE notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0, 1));
