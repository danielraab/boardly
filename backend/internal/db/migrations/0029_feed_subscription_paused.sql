-- Paused subscriptions (openspec change pause-feeds): a user can pause a feed
-- to leave its items out of "Latest" without removing it. NULL means not
-- paused. A paused feed is still refreshed and cleaned up as any other.

ALTER TABLE feed_subscriptions ADD COLUMN paused_at TEXT;
