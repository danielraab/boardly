-- One-time codes next to sign-in links (openspec change magic-link-code):
-- SHA-256 of token_hash || code, and the number of wrong codes entered.
-- Rows issued before this migration have no code.
ALTER TABLE magic_link_tokens ADD COLUMN code_hash BLOB;
ALTER TABLE magic_link_tokens ADD COLUMN code_attempts INTEGER NOT NULL DEFAULT 0;
