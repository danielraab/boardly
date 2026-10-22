-- Passkeys, WebAuthn ceremony state and the passkey enrollment window
-- (openspec change passkeys).

CREATE TABLE passkeys (
	id              TEXT PRIMARY KEY,
	user_id         TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	credential_id   BLOB NOT NULL UNIQUE,
	public_key      BLOB NOT NULL,
	aaguid          BLOB NOT NULL,
	sign_count      INTEGER NOT NULL,
	transports      TEXT NOT NULL, -- JSON array
	backup_eligible INTEGER NOT NULL,
	backup_state    INTEGER NOT NULL,
	label           TEXT,
	created_at      TEXT NOT NULL,
	last_used_at    TEXT
);

CREATE INDEX passkeys_user_id ON passkeys (user_id, created_at);

-- id_hash is the SHA-256 of the boardly_webauthn cookie. Registration
-- challenges are bound to the session that started them.
CREATE TABLE webauthn_challenges (
	id_hash    BLOB PRIMARY KEY,
	kind       TEXT NOT NULL CHECK (kind IN ('login', 'register')),
	user_id    TEXT REFERENCES users (id) ON DELETE CASCADE,
	session_id BLOB REFERENCES sessions (id_hash) ON DELETE CASCADE,
	data       TEXT NOT NULL,
	expires_at TEXT NOT NULL
);

ALTER TABLE sessions ADD COLUMN passkey_enrollment_until TEXT;
