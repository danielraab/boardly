-- Account deletion (openspec change delete-account). delete_after is when a
-- scheduled deletion is carried out (NULL: none scheduled); deletion_locale is
-- the language of the reminder email; deletion_reminder_sent_at is set once
-- that email went out.

ALTER TABLE users ADD COLUMN delete_after TEXT;
ALTER TABLE users ADD COLUMN deletion_locale TEXT;
ALTER TABLE users ADD COLUMN deletion_reminder_sent_at TEXT;

CREATE INDEX users_delete_after ON users (delete_after) WHERE delete_after IS NOT NULL;
