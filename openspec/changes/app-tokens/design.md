# Design

## Context

- All `/api/` requests go through `jsonMuxErrors` (`backend/internal/httpapi/router.go`), which runs `checkCSRF` on every unsafe method. The check requires a JSON or octet-stream body and an `Origin` equal to `FAMODASH_BASE_URL`. Browser extensions send `Origin: chrome-extension://…`, and Karakeep uploads use multipart, so app requests can't pass it.
- `requireUser` authenticates only by the `famodash_session` cookie. Secrets for sessions and magic links are stored as SHA-256 hashes (`auth.newSecret`), and timestamps are RFC 3339 UTC text (`db.FormatTime`).
- `favourite_nodes` uses a composite `(id, user_id)` key, so foreign keys keep references inside the owner's tree. The top-level folder can't be deleted (`ErrRootImmutable`).
- `FavouritePicker` already supports a `folder` mode that includes the top-level folder.
- Karakeep clients take a server address and call `<address>/api/v1/…`. During exploration we found that they accept an address with a path, so `<base>/api/karakeep` works as the address.

## Goals / Non-Goals

**Goals:**
- One token mechanism that later apps (Linkwarden) plug into by adding an app name and a router prefix.
- Bearer requests are strictly separate from cookie requests, so neither authentication path can be used for the other.
- One scope helper that every token endpoint must use, so subtree checks aren't reimplemented per endpoint.

**Non-Goals:**
- Karakeep bookmark, list, tag and asset endpoints (`karakeep-api`).
- Rate limiting of bearer requests beyond what exists today.
- Admin views, or tokens not bound to a folder.

## Decisions

### Data model

Migration `0011_app_tokens.sql`:

```sql
CREATE TABLE app_tokens (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	app          TEXT NOT NULL CHECK (app IN ('karakeep')),
	root_id      TEXT REFERENCES favourite_nodes (id) ON DELETE SET NULL,
	token_hash   BLOB NOT NULL UNIQUE,
	prefix       TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	last_used_at TEXT,
	expires_at   TEXT,
	revoked_at   TEXT
);
CREATE INDEX app_tokens_user ON app_tokens (user_id, created_at);
CREATE INDEX app_tokens_root ON app_tokens (root_id);
```

- `root_id` references only `favourite_nodes (id)` and not the composite `(id, user_id)` key. In SQLite, `ON DELETE SET NULL` nulls every column of the foreign key, which would also clear the `NOT NULL` `user_id`. Ownership is checked in `Create` instead, where the root must be a folder of the same user. The favourites delete revokes affected tokens **before** deleting nodes, in the same transaction, so the FK action only clears the reference afterwards.
- The app is a `CHECK` list. Adding Linkwarden later is a migration that rebuilds the table. That's acceptable for a small table and keeps invalid values out. Alternative: no CHECK, validate only in Go. Rejected because every other enum-like column in the schema uses CHECK.
- Status is derived (`revoked_at`, then `expires_at <= now`) rather than stored, so expiry needs no background job.
- A `NULL` root on an expired, never-revoked token is harmless, because the token can't authenticate again.

### Secrets

`famo_` + 20 bytes from `crypto/rand` as unpadded base32, which is 32 characters and 160 bits (`rand.Text()` has only 26 characters). The store keeps `sha256(secret)` in `token_hash` (unique index, single lookup) and `secret[:12]` as `prefix`. There's no salt or slow hash, because the secret is high-entropy, the same reasoning as for sessions. The fixed `famo_` prefix makes leaked tokens easy to grep for and to recognise in secret scanners.

### Package layout

- `internal/apptokens`: `Store` with `Create`, `List`, `Revoke`, `Delete` and `Authenticate(ctx, secret, app, now) (Token, error)`. `Authenticate` looks up by hash, checks the app and status, and updates `last_used_at` when it is older than one minute (as `auth` does with `touchInterval`). It returns one `ErrInvalidToken` for every failure, so callers can't leak which check failed.
- `internal/favourites`: new `Scope{UserID, RootID}` with `Contains(ctx, q, nodeID) (bool, error)` (recursive CTE from the node up to the root, short-circuiting at `RootID`), plus guards for create, move and delete that `karakeep-api` uses. It lives in `favourites` because it is a tree rule, not a token rule.
- `favourites.Store.Delete` gains the revocation step. In the same transaction, it collects the removed folder IDs (for `DeleteWithChildren`, the whole subtree, else only the folder itself) and runs `UPDATE app_tokens SET revoked_at = ? WHERE root_id IN (…) AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)`. The favourites package writes to another package's table here. The alternative, a hook interface injected into `favourites.Store`, adds indirection for one statement. If a second consumer appears, we can switch to a hook.

### HTTP routing

```
root mux
├── /api/karakeep/   → appAPI("karakeep", karakeepMux)   ← new, before /api/
│      cors → bearer auth (apptokens.Authenticate) → karakeepMux
│      (no cookie lookup, no checkCSRF; JSON 404/405 via muxErrorWriter)
├── /api/            → jsonMuxErrors(mux)   (cookie session + checkCSRF, unchanged)
└── /                → web
```

- Go's `ServeMux` picks the longest matching pattern, so `/api/karakeep/` wins over `/api/`. Both are registered on `root`.
- The bearer middleware puts the token (user ID, root, app) into the context. Handlers get a `favourites.Scope` from it. There's deliberately no `CurrentUser` from a session there.
- CORS: `Access-Control-Allow-Origin: *` with no credentials. That's safe because authentication is by header only and cookies are ignored. `OPTIONS` is answered before authentication, because preflights carry no `Authorization` header.
- Session endpoints `/api/app-tokens…` go on the existing mux with `requireUser`. There's no `PATCH` route, so the mux answers 405 by itself.

### Frontend

- `SettingsTab` gets `"apps"`. The route's search validation accepts it.
- `features/settings/ConnectedAppsTab.tsx` (list, revoke and delete confirmations with Base UI `AlertDialog`), `CreateAppTokenDialog.tsx` (a form step, then a secret step in the same dialog, so closing it discards the secret from memory), `appTokens/api.ts` (query options and mutations).
- The root path in the list comes from the favourites query and reuses `pathsOf` from `FavouritePicker`, which gets extracted into `tree.ts`.
- The favourites `DeleteDialog` loads the app-token list (cached query) and counts active tokens whose `rootId` is in the set of folders that the chosen delete mode removes. The server revokes regardless of what the client showed.
- The server address shown is `window.location.origin + "/api/karakeep"`. The SPA is served from `FAMODASH_BASE_URL`, so this matches without a new config endpoint.

## Risks / Trade-offs

- [The favourites code writes to `app_tokens`, which couples two packages] → Keep it to one well-tested statement in `Delete`. Revisit with a hook if a second consumer appears.
- [`last_used_at` writes on reads] → Throttled to once per minute per token.
- [CORS `*` on the app endpoints] → Safe only as long as these endpoints never accept cookie auth. A test asserts that a cookie-only request gets 401.
- [The CHECK constraint makes adding Linkwarden a table rebuild] → An accepted one-time cost in the `linkwarden` change.
- [The client-side warning count could differ from what the server revokes, e.g. with a stale cache] → The server is authoritative. The dialog refetches tokens when it opens.
- [Brute force against bearer tokens] → 160-bit secrets make guessing infeasible. There's no rate limit in this change.

## Migration Plan

Additive migration `0011_app_tokens.sql`. No existing data changes. To roll back, deploy the previous binary. The extra table is ignored.
