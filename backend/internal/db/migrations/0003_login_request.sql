-- Binds sign-in links to the browser that requested them (openspec change
-- magic-link-auth, task 9): SHA-256 of the boardly_login_request cookie.
ALTER TABLE magic_link_tokens ADD COLUMN browser_hash BLOB;
